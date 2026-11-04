//go:build linux

package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appassets "github.com/samosvalishe/free-turn-proxy/desktop/assets"
)

func isolateDesktop(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data with spaces"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("HOME", t.TempDir())
	// Do not modify the real desktop's GIO trust metadata from unit tests.
	t.Setenv("PATH", t.TempDir())
}

func TestDesktopEntry(t *testing.T) {
	entry, err := desktopEntry("/home/user/My Apps/freeturn", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(entry, "Exec=\"/home/user/My Apps/freeturn\" --autostart\n") {
		t.Fatalf("invalid Exec: %s", entry)
	}
	if !strings.Contains(entry, "Icon=freeturn\n") || !strings.Contains(entry, "StartupWMClass=freeturn\n") {
		t.Fatal("launcher is missing application branding")
	}
	entry, err = desktopEntry("/home/$user/100%/free`turn", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(entry, "Exec=\"/home/\\\\$user/100%%/free\\\\`turn\"\n") {
		t.Fatalf("unsafe Exec: %s", entry)
	}
	for _, path := range []string{"relative/path", "/bad\nExec=command", "/bad\rpath", "/bad\x00path"} {
		if _, err := desktopEntry(path, false); err == nil {
			t.Fatalf("accepted invalid path %q", path)
		}
	}
}

func TestInstallAndAutostart(t *testing.T) {
	isolateDesktop(t)
	if enabled, err := AutostartEnabled(); err != nil || enabled {
		t.Fatalf("initial autostart: %v, %v", enabled, err)
	}
	if err := SetAutostart(false); err != nil {
		t.Fatal(err)
	}
	path, err := Install()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("installed mode: %v", info.Mode())
	}
	entryPath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "applications", "freeturn.desktop")
	entry, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(entry), "--autostart") {
		t.Fatal("application launcher should not hide the window")
	}
	icon, err := os.ReadFile(filepath.Join(os.Getenv("XDG_DATA_HOME"), "icons", "hicolor", "scalable", "apps", "freeturn.svg"))
	if err != nil || !bytes.Equal(icon, appassets.IconSVG) {
		t.Fatalf("application icon was not installed: %v", err)
	}
	if err := SetAutostart(true); err != nil {
		t.Fatal(err)
	}
	if enabled, err := AutostartEnabled(); err != nil || !enabled {
		t.Fatalf("enabled autostart: %v, %v", enabled, err)
	}
	autoPath, _ := autostartPath()
	entry, err = os.ReadFile(autoPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(entry), "--autostart") {
		t.Fatal("autostart flag missing")
	}
	if err := SetAutostart(false); err != nil {
		t.Fatal(err)
	}
	if enabled, err := AutostartEnabled(); err != nil || enabled {
		t.Fatalf("disabled autostart: %v, %v", enabled, err)
	}
}

func TestCreateDesktopShortcut(t *testing.T) {
	isolateDesktop(t)
	config := os.Getenv("XDG_CONFIG_HOME")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "user-dirs.dirs"), []byte("XDG_DESKTOP_DIR=\"$HOME/Рабочий стол\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := CreateDesktopShortcut()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(os.Getenv("HOME"), "Рабочий стол", "freeturn.desktop") {
		t.Fatalf("wrong desktop directory: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("invalid shortcut mode: %v, %v", info, err)
	}
	if _, err := CreateDesktopShortcut(); err != nil {
		t.Fatalf("cannot update managed shortcut: %v", err)
	}
}

func TestDesktopShortcutDoesNotOverwriteUnknownFile(t *testing.T) {
	isolateDesktop(t)
	dir := filepath.Join(os.Getenv("HOME"), "Desktop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "freeturn.desktop")
	if err := os.WriteFile(path, []byte("custom launcher\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateDesktopShortcut(); err == nil {
		t.Fatal("overwrote an unmanaged shortcut")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "custom launcher\n" {
		t.Fatal("existing shortcut changed")
	}
}

func TestDesktopDisabled(t *testing.T) {
	isolateDesktop(t)
	config := os.Getenv("XDG_CONFIG_HOME")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "user-dirs.dirs"), []byte("XDG_DESKTOP_DIR=\"$HOME\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := desktopDirectory(); err == nil {
		t.Fatal("created shortcut in HOME despite disabled desktop")
	}
}

func TestPreferences(t *testing.T) {
	isolateDesktop(t)
	prefs, err := LoadPreferences()
	if err != nil || !prefs.CloseToTray {
		t.Fatalf("defaults: %+v, %v", prefs, err)
	}
	if err := SavePreferences(Preferences{CloseToTray: false}); err != nil {
		t.Fatal(err)
	}
	prefs, err = LoadPreferences()
	if err != nil || prefs.CloseToTray {
		t.Fatalf("saved preferences: %+v, %v", prefs, err)
	}
}
