package routing

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"
)

type fakeResolver struct {
	calls   map[string]int
	answers map[string][]netip.Addr
	err     error
}

func (r *fakeResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.calls == nil {
		r.calls = map[string]int{}
	}
	r.calls[host]++
	return r.answers[host], r.err
}

func has(routes, address string) bool {
	ip := netip.MustParseAddr(address)
	for _, route := range strings.Split(routes, ",") {
		prefix, err := netip.ParsePrefix(route)
		if err == nil && prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func TestBlackAndWhiteListRouting(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name, base, dns string
		lists           []List
		through, bypass []string
	}{
		{name: "blacklist", base: "0.0.0.0/0,::/0", lists: []List{{Mode: Blacklist, Enabled: true, Entries: []string{"203.0.113.5", "2001:db8::/32"}}}, through: []string{"203.0.113.4", "1.1.1.1", "2001:db9::1"}, bypass: []string{"203.0.113.5", "2001:db8::1"}},
		{name: "whitelist", base: "0.0.0.0/0,::/0", lists: []List{{Mode: Whitelist, Enabled: true, Entries: []string{"198.51.100.0/24", "2001:db8::/32"}}}, through: []string{"198.51.100.42", "2001:db8::1"}, bypass: []string{"1.1.1.1", "2001:db9::1"}},
		{name: "black beats white", base: "0.0.0.0/0", lists: []List{{Mode: Whitelist, Enabled: true, Entries: []string{"10.0.0.0/8"}}, {Mode: Blacklist, Enabled: true, Entries: []string{"10.1.0.0/16"}}}, through: []string{"10.2.0.1"}, bypass: []string{"10.1.0.1", "8.8.8.8"}},
		{name: "disabled whitelist ignored", base: "0.0.0.0/0", lists: []List{{Mode: Whitelist, Enabled: false, Entries: []string{"10.0.0.0/8"}}}, through: []string{"8.8.8.8"}},
		{name: "DNS infrastructure retained", base: "0.0.0.0/0", dns: "10.9.0.1", lists: []List{{Mode: Whitelist, Enabled: true, Entries: []string{"203.0.113.0/24"}}, {Mode: Blacklist, Enabled: true, Entries: []string{"10.9.0.1"}}}, through: []string{"203.0.113.5", "10.9.0.1"}, bypass: []string{"10.9.0.2"}},
		{name: "within WireGuard AllowedIPs", base: "10.0.0.0/8", lists: []List{{Mode: Whitelist, Enabled: true, Entries: []string{"0.0.0.0/0"}}}, through: []string{"10.1.1.1"}, bypass: []string{"1.1.1.1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			routes, err := Build(ctx, test.base, test.dns, test.lists, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, address := range test.through {
				if !has(routes, address) {
					t.Fatalf("%s not through VPN: %s", address, routes)
				}
			}
			for _, address := range test.bypass {
				if has(routes, address) {
					t.Fatalf("%s not bypassed: %s", address, routes)
				}
			}
		})
	}
}

func TestDomainSnapshotAndCache(t *testing.T) {
	resolver := &fakeResolver{answers: map[string][]netip.Addr{"example.com": {netip.MustParseAddr("203.0.113.10"), netip.MustParseAddr("2001:db8::10")}}}
	lists := []List{{Mode: Whitelist, Enabled: true, Entries: []string{"EXAMPLE.COM.", "example.com"}}, {Mode: Blacklist, Enabled: false, Entries: []string{"never-looked-up.example"}}}
	routes, err := Build(context.Background(), "0.0.0.0/0,::/0", "", lists, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if !has(routes, "203.0.113.10") || !has(routes, "2001:db8::10") || has(routes, "203.0.113.11") {
		t.Fatal(routes)
	}
	if resolver.calls["example.com"] != 1 || len(resolver.calls) != 1 {
		t.Fatal("domain not cached or disabled list resolved")
	}
	// Reconnecting resolves again, instead of keeping a permanent/stale domain cache.
	resolver.answers["example.com"] = []netip.Addr{netip.MustParseAddr("203.0.113.20")}
	routes, err = Build(context.Background(), "0.0.0.0/0", "", lists, resolver)
	if err != nil || !has(routes, "203.0.113.20") || has(routes, "203.0.113.10") {
		t.Fatalf("stale snapshot: %s, %v", routes, err)
	}
}

func TestRoutingFailuresDoNotSilentlyIgnoreRules(t *testing.T) {
	for _, lists := range [][]List{
		{{Mode: Whitelist, Enabled: true}},
		{{Mode: Whitelist, Enabled: true, Entries: []string{"2001:db8::1"}}},
		{{Mode: Blacklist, Enabled: true, Entries: []string{"0.0.0.0/0"}}},
	} {
		if _, err := Build(context.Background(), "0.0.0.0/0", "", lists, nil); err == nil {
			t.Fatal("empty routing plan accepted")
		}
	}
	for _, resolver := range []*fakeResolver{{err: errors.New("offline")}, {answers: map[string][]netip.Addr{}}} {
		if _, err := Build(context.Background(), "0.0.0.0/0", "", []List{{Mode: Blacklist, Enabled: true, Entries: []string{"example.com"}}}, resolver); err == nil {
			t.Fatal("DNS failure silently ignored")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Build(ctx, "0.0.0.0/0", "", []List{{Mode: Whitelist, Enabled: true, Entries: []string{"example.com"}}}, &fakeResolver{}); err == nil {
		t.Fatal("cancelled DNS lookup accepted")
	}
}

func TestNormalizeEntriesAndRejectAmbiguousInput(t *testing.T) {
	lists, err := Normalize([]List{{Name: "  Test  ", Mode: Blacklist, Enabled: true, Entries: []string{"", "# comment", "EXAMPLE.com.", "example.com", "192.0.2.99/24", "::ffff:192.0.2.1", "пример.рф"}}})
	if err != nil {
		t.Fatal(err)
	}
	want := "example.com,192.0.2.0/24,192.0.2.1,xn--e1afmkfd.xn--p1ai"
	if lists[0].Name != "Test" || strings.Join(lists[0].Entries, ",") != want {
		t.Fatalf("normalized: %+v", lists)
	}
	for _, bad := range []string{"https://example.com", "*.example.com", "example.com/path", "example.com:443", "999.0.0.1", "bad_domain", "-bad.com", "fe80::1%eth0", "::ffff:192.0.2.0/120"} {
		if _, err := Normalize([]List{{Mode: Blacklist, Entries: []string{bad}}}); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestSubtractCoversEveryAddressExceptExclusion(t *testing.T) {
	base := netip.MustParsePrefix("192.0.2.0/24")
	for excluded := 0; excluded < 256; excluded++ {
		cut := netip.MustParsePrefix(fmt.Sprintf("192.0.2.%d/32", excluded))
		parts := subtract(base, cut)
		var values []string
		for _, prefix := range parts {
			values = append(values, prefix.String())
		}
		routes := strings.Join(values, ",")
		for i := 0; i < 256; i++ {
			if has(routes, fmt.Sprintf("192.0.2.%d", i)) != (i != excluded) {
				t.Fatalf("bad subtraction for %d/%d", excluded, i)
			}
		}
	}
}

func TestCompactAndLimits(t *testing.T) {
	input := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/25"), netip.MustParsePrefix("10.0.0.128/25"), netip.MustParsePrefix("10.0.0.1/32")}
	got := compact(input)
	if len(got) != 1 || got[0].String() != "10.0.0.0/24" {
		t.Fatalf("not compacted: %v", got)
	}
	var entries []string
	for i := 0; i < 48; i++ {
		entries = append(entries, fmt.Sprintf("2001:db8:%x::1", i*31))
	}
	if _, err := Build(context.Background(), "::/0", "", []List{{Mode: Blacklist, Enabled: true, Entries: entries}}, nil); err == nil {
		t.Fatal("route explosion accepted")
	}
	lists := make([]List, MaxLists+1)
	if _, err := Normalize(lists); err == nil {
		t.Fatal("list limit ignored")
	}
	entries = nil
	for i := 0; i < MaxEntries+1; i++ {
		entries = append(entries, fmt.Sprintf("site-%d.example", i))
	}
	if _, err := Normalize([]List{{Mode: Blacklist, Entries: entries}}); err == nil {
		t.Fatal("entry limit ignored")
	}
}

func TestRelayAndSSHAddressesAlwaysBypassVPN(t *testing.T) {
	lists := []List{{Mode: Whitelist, Enabled: true, Entries: []string{"203.0.113.0/24", "2001:db8::/32"}}}
	routes, err := Build(context.Background(), "0.0.0.0/0,::/0", "", lists, nil, "203.0.113.10", "2001:db8::1")
	if err != nil {
		t.Fatal(err)
	}
	if has(routes, "203.0.113.10") || has(routes, "2001:db8::1") || !has(routes, "203.0.113.11") || !has(routes, "2001:db8::2") {
		t.Fatal(routes)
	}
}
