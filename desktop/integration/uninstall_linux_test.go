//go:build linux

package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallPreservesProfilesBackupsAndUnmanagedFiles(t *testing.T) {
	isolateDesktop(t)
	if _, err := CreateDesktopShortcut(); err != nil {
		t.Fatal(err)
	}
	if err := SetAutostart(true); err != nil {
		t.Fatal(err)
	}
	data := os.Getenv("XDG_DATA_HOME")
	profile := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "FreeTurnDesktop", "servers.json")
	backup := filepath.Join(data, "FreeTurnDesktop", ".freeturn-update-test", "previous", "freeturn-desktop")
	for _, path := range []string{profile, backup} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	menu := filepath.Join(data, "applications", "freeturn.desktop")
	if err := os.WriteFile(menu, []byte("[Desktop Entry]\nName=Custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := uninstallUserFiles(); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{profile, backup, menu} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("preserved file %s: %v", path, err)
		}
	}
	desktop, _ := desktopDirectory()
	for _, path := range []string{
		filepath.Join(data, "FreeTurnDesktop", "freeturn-desktop"),
		filepath.Join(data, "icons/hicolor/scalable/apps/freeturn.svg"),
		filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "autostart/freeturn.desktop"),
		filepath.Join(desktop, "freeturn.desktop"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("not removed: %s (%v)", path, err)
		}
	}
}
