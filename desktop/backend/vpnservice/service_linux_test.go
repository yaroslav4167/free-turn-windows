//go:build linux

package vpnservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitsRestrictAccessAndExecutable(t *testing.T) {
	socket, service, err := units(1000)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"SocketUser=1000\n", "SocketMode=0600\n", "Accept=yes\n", "MaxConnections=2\n", "ListenStream=/run/freeturn-vpn/1000.sock\n", "Service=freeturn-vpn-1000@.service\n"} {
		if !strings.Contains(socket, required) {
			t.Fatalf("missing %q in socket unit", required)
		}
	}
	for _, required := range []string{"ExecStart=/usr/local/libexec/freeturn-vpn-helper --uid 1000\n", "StandardInput=socket\n", "NoNewPrivileges=yes\n", "ProtectHome=yes\n", "ProtectSystem=strict\n", "CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW\n"} {
		if !strings.Contains(service, required) {
			t.Fatalf("missing %q in service unit", required)
		}
	}
	if strings.Contains(service, "/home/") || strings.Contains(service, "pkexec") || strings.Contains(service, "/bin/sh") {
		t.Fatal("service must execute only the protected system binary")
	}
	for _, uid := range []uint32{0, ^uint32(0)} {
		if _, _, err := units(uid); err == nil {
			t.Fatalf("accepted UID %d", uid)
		}
	}
}

func TestAtomicWriteDoesNotFollowTargetSymlink(t *testing.T) {
	dir := t.TempDir()
	victim, target := filepath.Join(dir, "victim"), filepath.Join(dir, "target")
	if err := os.WriteFile(victim, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, target); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(target, strings.NewReader("helper"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(victim)
	if err != nil || string(data) != "unchanged" {
		t.Fatalf("symlink target was modified: %q, %v", data, err)
	}
	info, err := os.Lstat(target)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o755 {
		t.Fatalf("invalid installed file: %v, %v", info, err)
	}
}

func TestRejectWritableSystemDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := secureDirectory(dir); err == nil {
		t.Fatal("accepted a user-writable temporary path as a system directory")
	}
}

func TestInstallerRequiresRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("test specifically checks unprivileged callers")
	}
	if err := Install(uint32(os.Getuid()), "/nonexistent"); err == nil {
		t.Fatal("unprivileged system installation accepted")
	}
	if err := Remove(uint32(os.Getuid())); err == nil {
		t.Fatal("unprivileged system removal accepted")
	}
}
