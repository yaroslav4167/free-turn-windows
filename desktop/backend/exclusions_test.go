package backend

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"testing"

	"github.com/samosvalishe/free-turn-proxy/desktop/backend/routing"
	"github.com/samosvalishe/free-turn-proxy/desktop/backend/serversetup"
)

func TestLinuxManagementFollowsVPNRoutes(t *testing.T) {
	ips := []string{"138.124.29.112"}
	if got := routeExclusions("linux", ips); len(got) != 0 {
		t.Fatalf("Linux must not force SSH over a blocked physical route: %v", got)
	}
	if got := routeExclusions("windows", ips); !reflect.DeepEqual(got, ips) {
		t.Fatalf("Windows route handling changed: %v", got)
	}
	plan, err := routing.Build(context.Background(), "0.0.0.0/0", "", []routing.List{{
		Name: "VPS", Enabled: true, Mode: routing.Whitelist, Entries: ips,
	}}, nil, routeExclusions("linux", ips)...)
	if err != nil || plan != "138.124.29.112/32" {
		t.Fatalf("explicit VPS VPN route was removed: %q, %v", plan, err)
	}
}

type exclusionResolver func(context.Context, string, string) ([]netip.Addr, error)

func (f exclusionResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

func TestResolveExclusions(t *testing.T) {
	calls := 0
	resolver := exclusionResolver(func(ctx context.Context, network, host string) ([]netip.Addr, error) {
		calls++
		if host != "s4.logixy.net" || network != "ip" {
			t.Fatalf("unexpected lookup: %s %s", network, host)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("missing DNS deadline")
		}
		return []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("::ffff:192.0.2.1"), netip.MustParseAddr("2001:db8::1")}, nil
	})
	srv := Server{Peer: "s4.logixy.net:56000", SSH: &serversetup.SSHConfig{IP: "s4.logixy.net"}}
	got, err := resolveExclusions(context.Background(), srv, resolver)
	if err != nil || !reflect.DeepEqual(got, []string{"192.0.2.1", "2001:db8::1"}) || calls != 1 {
		t.Fatalf("got %v, %v; calls %d", got, err, calls)
	}
	srv.Peer = "[2001:db8::1]:56000"
	srv.SSH.IP = "192.0.2.1"
	if _, err := resolveExclusions(context.Background(), srv, resolver); err != nil || calls != 1 {
		t.Fatalf("IP literals: %v", err)
	}
}

func TestResolveExclusionsFailures(t *testing.T) {
	for _, test := range []string{"dns", "empty", "invalid", "scoped", "cancelled"} {
		t.Run(test, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if test == "cancelled" {
				cancel()
			}
			resolver := exclusionResolver(func(context.Context, string, string) ([]netip.Addr, error) {
				switch test {
				case "dns":
					return nil, errors.New("offline")
				case "invalid":
					return []netip.Addr{{}}, nil
				case "scoped":
					return []netip.Addr{netip.MustParseAddr("fe80::1%eth0")}, nil
				case "cancelled":
					t.Fatal("lookup after cancellation")
				}
				return nil, nil
			})
			if _, err := resolveExclusions(ctx, Server{Peer: "example.invalid:56000"}, resolver); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
