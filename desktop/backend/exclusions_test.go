package backend

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"testing"

	"github.com/samosvalishe/free-turn-proxy/desktop/backend/serversetup"
)

type fakeEndpointResolver func(context.Context, string, string) ([]netip.Addr, error)

func (f fakeEndpointResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

func TestResolveEndpointDomains(t *testing.T) {
	calls := 0
	r := fakeEndpointResolver(func(ctx context.Context, network, host string) ([]netip.Addr, error) {
		calls++
		if host != "s4.logixy.net" || network != "ip" {
			t.Fatalf("unexpected lookup %s %s", network, host)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded DNS lookup")
		}
		return []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("::ffff:192.0.2.1"), netip.MustParseAddr("2001:db8::1")}, nil
	})
	srv := Server{Peer: "s4.logixy.net:56000", SSH: &serversetup.SSHConfig{IP: "s4.logixy.net"}}
	got, err := resolveExclusions(context.Background(), srv, r)
	if err != nil || calls != 1 || !reflect.DeepEqual(got, []string{"192.0.2.1", "2001:db8::1"}) {
		t.Fatalf("addresses=%v calls=%d error=%v", got, calls, err)
	}
	srv.Peer, srv.SSH.IP = "[2001:db8::1]:56000", "192.0.2.1"
	if _, err := resolveExclusions(context.Background(), srv, r); err != nil || calls != 1 {
		t.Fatalf("literal IP lookup: %v", err)
	}
}

func TestEndpointResolutionFailsClosed(t *testing.T) {
	for _, scenario := range []string{"dns", "empty", "invalid", "scoped", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if scenario == "cancel" {
				cancel()
			}
			r := fakeEndpointResolver(func(context.Context, string, string) ([]netip.Addr, error) {
				switch scenario {
				case "dns":
					return nil, errors.New("offline")
				case "invalid":
					return []netip.Addr{{}}, nil
				case "scoped":
					return []netip.Addr{netip.MustParseAddr("fe80::1%eth0")}, nil
				case "cancel":
					t.Fatal("lookup after cancellation")
				}
				return nil, nil
			})
			if _, err := resolveExclusions(ctx, Server{Peer: "example.invalid:56000"}, r); err == nil {
				t.Fatal("expected planning error")
			}
		})
	}
}
