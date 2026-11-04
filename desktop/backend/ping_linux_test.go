//go:build linux

package backend

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeServerPingUsesOneBoundedUnprivilegedCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	t.Setenv("LC_ALL", "ru_RU.UTF-8")
	script := "#!/bin/sh\n[ \"$LC_ALL\" = C ] || exit 1\n[ \"$*\" = '-n -c 1 -W 2 -w 2 -- example.com' ] || exit 1\necho '64 bytes from host: time=21.5 ms'\n"
	if err := os.WriteFile(filepath.Join(dir, "ping"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	value, err := probeServerPing(context.Background(), "example.com")
	if err != nil || value != 21.5 {
		t.Fatalf("unexpected result %v: %v", value, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ping"), []byte("#!/bin/sh\nexec /bin/sleep 10\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := probeServerPing(ctx, "example.com"); err == nil {
		t.Fatal("cancelled ping succeeded")
	}
}
