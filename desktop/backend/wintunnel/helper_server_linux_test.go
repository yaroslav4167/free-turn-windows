//go:build linux

package wintunnel

import (
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnableIPv6(t *testing.T) {
	for _, initial := range []string{"1\n", "0\n"} {
		t.Run(initial[:1], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "disable_ipv6")
			if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := enableIPv6(path); err != nil {
				t.Fatal(err)
			}
			value, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(value) != "0\n" {
				t.Fatalf("disable_ipv6 = %q, want 0", value)
			}
		})
	}
}

func TestEnableIPv6Unavailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disable_ipv6")
	if err := enableIPv6(path); !os.IsNotExist(err) {
		t.Fatalf("expected missing IPv6 sysctl error, got %v", err)
	}
}

func TestSplitDefaultRoutes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"0.0.0.0/0", []string{"0.0.0.0/1", "128.0.0.0/1"}},
		{"::/0", []string{"::/1", "8000::/1"}},
		{"10.0.0.0/8", []string{"10.0.0.0/8"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			prefix := netip.MustParsePrefix(tc.in)
			got := splitDefault(prefix)
			strings := make([]string, len(got))
			for i, route := range got {
				strings[i] = route.String()
			}
			if !reflect.DeepEqual(strings, tc.want) {
				t.Fatalf("splitDefault(%s) = %v, want %v", tc.in, strings, tc.want)
			}
		})
	}
}

func TestParsePrefixesRejectsInvalidRoute(t *testing.T) {
	if _, err := parsePrefixes("0.0.0.0/0,garbage"); err == nil {
		t.Fatal("expected invalid route to be rejected before invoking ip")
	}
}
