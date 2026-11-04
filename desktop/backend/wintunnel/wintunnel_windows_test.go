//go:build windows

package wintunnel

import (
	"errors"
	"net/netip"
	"testing"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

type fakeRouteAdapter struct {
	routes map[netip.Prefix]netip.Addr
	err    error
}

func (a *fakeRouteAdapter) AddRoute(destination netip.Prefix, nextHop netip.Addr, _ uint32) error {
	if a.err != nil {
		return a.err
	}
	if _, exists := a.routes[destination]; exists {
		return windows.ERROR_OBJECT_ALREADY_EXISTS
	}
	a.routes[destination] = nextHop
	return nil
}

func TestAddRoutesPreservesLocalAddress(t *testing.T) {
	local := netip.MustParsePrefix("10.13.13.9/32")
	subnet := netip.MustParsePrefix("10.13.13.0/24")
	a := &fakeRouteAdapter{routes: map[netip.Prefix]netip.Addr{
		local: netip.IPv4Unspecified(), subnet: netip.IPv4Unspecified(),
	}}
	routes := []*winipcfg.RouteData{
		{Destination: netip.MustParsePrefix("0.0.0.0/0"), NextHop: netip.IPv4Unspecified()},
		{Destination: subnet, NextHop: netip.IPv4Unspecified()},
		{Destination: netip.MustParsePrefix("::/0"), NextHop: netip.IPv6Unspecified()},
	}
	if err := addRoutesPreservingLocal(a, routes); err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []netip.Prefix{local, subnet, routes[0].Destination, routes[2].Destination} {
		if _, ok := a.routes[prefix]; !ok {
			t.Errorf("missing route %s", prefix)
		}
	}
}

func TestAddRoutesReturnsFailure(t *testing.T) {
	want := windows.ERROR_ACCESS_DENIED
	a := &fakeRouteAdapter{err: want}
	err := addRoutesPreservingLocal(a, []*winipcfg.RouteData{
		{Destination: netip.MustParsePrefix("0.0.0.0/0"), NextHop: netip.IPv4Unspecified()},
	})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}
