// Package routing builds split-tunnel IP routes from named domain/IP lists.
// Domains are a DNS snapshot at connection time, not a DNS/HTTPS interceptor.
package routing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

const (
	Blacklist  = "blacklist"
	Whitelist  = "whitelist"
	MaxLists   = 16
	MaxEntries = 256
	MaxRoutes  = 1024
)

type List struct {
	Name    string   `json:"name"`
	Enabled bool     `json:"enabled"`
	Mode    string   `json:"mode"`
	Entries []string `json:"entries"`
}

func Clone(lists []List) []List {
	out := make([]List, len(lists))
	copy(out, lists)
	for i := range out {
		out[i].Entries = append([]string{}, lists[i].Entries...)
	}
	return out
}

func Normalize(lists []List) ([]List, error) {
	if len(lists) > MaxLists {
		return nil, fmt.Errorf("не более %d списков", MaxLists)
	}
	out := Clone(lists)
	count := 0
	for i := range out {
		list := &out[i]
		if len(list.Entries) > MaxEntries*4 {
			return nil, fmt.Errorf("список %q содержит слишком много строк", list.Name)
		}
		list.Name = strings.TrimSpace(list.Name)
		if list.Name == "" {
			list.Name = fmt.Sprintf("Список %d", i+1)
		}
		if len([]rune(list.Name)) > 80 {
			return nil, errors.New("имя списка длиннее 80 символов")
		}
		if list.Mode != Blacklist && list.Mode != Whitelist {
			return nil, fmt.Errorf("список %q: неизвестный режим", list.Name)
		}
		seen := map[string]bool{}
		entries := []string{}
		for _, raw := range list.Entries {
			if len(raw) > 1024 {
				return nil, fmt.Errorf("слишком длинная запись в списке %q", list.Name)
			}
			raw = strings.TrimSpace(raw)
			if raw == "" || strings.HasPrefix(raw, "#") {
				continue
			}
			entry, err := normalizeEntry(raw)
			if err != nil {
				return nil, fmt.Errorf("список %q: %q: %w", list.Name, raw, err)
			}
			if !seen[entry] {
				entries = append(entries, entry)
				seen[entry] = true
				count++
			}
			if count > MaxEntries {
				return nil, fmt.Errorf("не более %d записей во всех списках", MaxEntries)
			}
		}
		list.Entries = entries
		if list.Enabled && list.Mode == Whitelist && len(entries) == 0 {
			return nil, fmt.Errorf("белый список %q пуст", list.Name)
		}
	}
	return out, nil
}

func normalizeEntry(raw string) (string, error) {
	if ip, err := netip.ParseAddr(raw); err == nil {
		if ip.Zone() != "" {
			return "", errors.New("IPv6 zone ID не поддерживается")
		}
		return ip.Unmap().String(), nil
	}
	if prefix, err := netip.ParsePrefix(raw); err == nil {
		if prefix.Addr().Is4In6() {
			return "", errors.New("используйте обычную IPv4-подсеть вместо IPv4-mapped IPv6")
		}
		return prefix.Masked().String(), nil
	}
	if strings.ContainsAny(raw, "/:*?#@[]\\ \t\r\n") {
		return "", errors.New("ожидается домен без URL/маски, IP или CIDR-подсеть")
	}
	domain, err := idna.Lookup.ToASCII(strings.TrimSuffix(strings.ToLower(raw), "."))
	if err != nil || domain == "" || len(domain) > 253 {
		return "", errors.New("некорректный домен")
	}
	numeric := true
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("некорректный домен")
		}
		for _, c := range label {
			if c < '0' || c > '9' {
				numeric = false
			}
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", errors.New("некорректный домен")
			}
		}
	}
	if numeric {
		return "", errors.New("некорректный IP-адрес")
	}
	return domain, nil
}

func Enabled(lists []List) bool {
	for _, list := range lists {
		if list.Enabled {
			return true
		}
	}
	return false
}

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// Build returns OS routes only. WireGuard's cryptographic AllowedIPs are never
// expanded/reassigned to another peer. Blacklists win over whitelist entries;
// DNS server routes are retained as necessary VPN infrastructure.
func Build(ctx context.Context, allowed, dns string, lists []List, resolver Resolver, protectedIPs ...string) (string, error) {
	normalized, err := Normalize(lists)
	if err != nil {
		return "", err
	}
	if !Enabled(normalized) {
		return allowed, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	base, err := parsePrefixes(allowed)
	if err != nil {
		return "", err
	}
	var white, black []netip.Prefix
	hasWhite := false
	cache := map[string][]netip.Prefix{}
	for _, list := range normalized {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if !list.Enabled {
			continue
		}
		hasWhite = hasWhite || list.Mode == Whitelist
		for _, entry := range list.Entries {
			prefixes, exists := cache[entry]
			if !exists {
				if prefix, err := netip.ParsePrefix(entry); err == nil {
					prefixes = []netip.Prefix{prefix}
				} else if ip, err := netip.ParseAddr(entry); err == nil {
					prefixes = []netip.Prefix{netip.PrefixFrom(ip, ip.BitLen())}
				} else {
					ips, err := resolver.LookupNetIP(ctx, "ip", entry)
					if err != nil {
						return "", fmt.Errorf("DNS для %q: %w", entry, err)
					}
					for _, ip := range ips {
						ip = ip.Unmap()
						if ip.IsValid() && ip.Zone() == "" {
							prefixes = append(prefixes, netip.PrefixFrom(ip, ip.BitLen()))
						}
					}
					if len(prefixes) == 0 {
						return "", fmt.Errorf("DNS для %q не вернул IP-адресов", entry)
					}
				}
				cache[entry] = prefixes
			}
			if list.Mode == Whitelist {
				white = append(white, prefixes...)
			} else {
				black = append(black, prefixes...)
			}
			if len(white)+len(black) > 4096 {
				return "", errors.New("DNS вернул слишком много адресов")
			}
		}
	}
	selected := base
	if hasWhite {
		selected = intersect(base, compact(white))
	}
	for _, exclusion := range compact(black) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		var next []netip.Prefix
		for _, prefix := range selected {
			next = append(next, subtract(prefix, exclusion)...)
		}
		selected = compact(next)
		if len(selected) > MaxRoutes {
			return "", errors.New("списки создают слишком много маршрутов; используйте более крупные подсети")
		}
	}
	if len(selected) == 0 {
		return "", errors.New("списки не оставляют ни одного VPN-маршрута в AllowedIPs; проверьте белые списки и исключения")
	}
	// Keep the configured VPN DNS reachable even with a narrow whitelist.
	for _, raw := range strings.Split(dns, ",") {
		ip, err := netip.ParseAddr(strings.TrimSpace(raw))
		if err != nil {
			if strings.TrimSpace(raw) == "" {
				continue
			}
			return "", fmt.Errorf("DNS профиля: %w", err)
		}
		ip = ip.Unmap()
		selected = append(selected, intersect(base, []netip.Prefix{netip.PrefixFrom(ip, ip.BitLen())})...)
	}
	selected = compact(selected)
	// Relay/SSH endpoints always keep their physical path. In particular, do
	// not add a competing /32 route when a whitelist mentions the VPS itself.
	for _, raw := range protectedIPs {
		ip, err := netip.ParseAddr(raw)
		if err != nil {
			return "", fmt.Errorf("служебный адрес VPN: %w", err)
		}
		ip = ip.Unmap()
		var next []netip.Prefix
		for _, prefix := range selected {
			next = append(next, subtract(prefix, netip.PrefixFrom(ip, ip.BitLen()))...)
		}
		selected = compact(next)
	}
	if len(selected) == 0 {
		return "", errors.New("списки содержат только служебные адреса, которые должны идти вне VPN")
	}
	if len(selected) > MaxRoutes {
		return "", errors.New("слишком много VPN-маршрутов")
	}
	result := make([]string, len(selected))
	for i, prefix := range selected {
		result[i] = prefix.String()
	}
	return strings.Join(result, ","), nil
}

func parsePrefixes(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, value := range strings.Split(raw, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("AllowedIPs: %w", err)
		}
		out = append(out, prefix.Masked())
	}
	return compact(out), nil
}

func intersect(a, b []netip.Prefix) []netip.Prefix {
	var out []netip.Prefix
	for _, first := range a {
		for _, second := range b {
			if first.Overlaps(second) {
				if first.Bits() >= second.Bits() {
					out = append(out, first)
				} else {
					out = append(out, second)
				}
			}
		}
	}
	return compact(out)
}

func subtract(base, cut netip.Prefix) []netip.Prefix {
	if !base.Overlaps(cut) {
		return []netip.Prefix{base}
	}
	if cut.Bits() <= base.Bits() {
		return nil
	}
	bits := base.Bits() + 1
	left := netip.PrefixFrom(base.Addr(), bits).Masked()
	var rightAddr netip.Addr
	if base.Addr().Is4() {
		value := base.Addr().As4()
		value[(bits-1)/8] |= 1 << (7 - (bits-1)%8)
		rightAddr = netip.AddrFrom4(value)
	} else {
		value := base.Addr().As16()
		value[(bits-1)/8] |= 1 << (7 - (bits-1)%8)
		rightAddr = netip.AddrFrom16(value)
	}
	right := netip.PrefixFrom(rightAddr, bits)
	return append(subtract(left, cut), subtract(right, cut)...)
}

func compact(prefixes []netip.Prefix) []netip.Prefix {
	values := slices.Clone(prefixes)
	sort.Slice(values, func(i, j int) bool {
		if values[i].Addr() == values[j].Addr() {
			return values[i].Bits() < values[j].Bits()
		}
		return values[i].Addr().Less(values[j].Addr())
	})
	var out []netip.Prefix
	for _, prefix := range values {
		if len(out) > 0 && out[len(out)-1].Contains(prefix.Addr()) && out[len(out)-1].Bits() <= prefix.Bits() {
			continue
		}
		out = append(out, prefix)
		for len(out) >= 2 {
			a, b := out[len(out)-2], out[len(out)-1]
			if a.Bits() == 0 || a.Bits() != b.Bits() || a.Addr().BitLen() != b.Addr().BitLen() {
				break
			}
			parent := netip.PrefixFrom(a.Addr(), a.Bits()-1).Masked()
			if !parent.Contains(b.Addr()) {
				break
			}
			out = append(out[:len(out)-2], parent)
		}
	}
	return out
}
