//go:build windows

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsInstallPathAndUnmanagedFiles(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	target, err := InstalledPath()
	if err != nil || filepath.Base(target) != "freeturn-desktop.exe" {
		t.Fatalf("target %q: %v", target, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("user-owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(); err == nil {
		t.Fatal("overwrote an unmanaged file")
	}
	if err := Uninstall(); err == nil {
		t.Fatal("uninstalled an unmanaged directory")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "user-owned" {
		t.Fatal("unmanaged file changed")
	}
}

func TestWindowsPowerShellPathQuoting(t *testing.T) {
	path := `C:\Users\O'Brien\FreeTurn;Remove-Item.txt`
	if got := psLiteral(path); got != `'C:\Users\O''Brien\FreeTurn;Remove-Item.txt'` {
		t.Fatalf("literal: %s", got)
	}
	script := shortcutScript(path, "DesktopDirectory")
	if !strings.Contains(script, psLiteral(path)) || !strings.Contains(script, managedLauncher) {
		t.Fatal("shortcut lacks escaped path or ownership check")
	}
}

func TestWindowsUpdateAcknowledgementIsConstrainedAndExclusive(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	target, _ := InstalledPath()
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	stage, err := os.MkdirTemp(filepath.Dir(target), ".freeturn-update-")
	if err != nil {
		t.Fatal(err)
	}
	ack := filepath.Join(stage, "restart-ready")
	t.Setenv("FREETURN_UPDATE_ACK_FILE", ack)
	AcknowledgeUpdate()
	data, err := os.ReadFile(ack)
	if err != nil || string(data) != "\x01" {
		t.Fatalf("ack: %q, %v", data, err)
	}
	if err := os.WriteFile(ack, []byte("do-not-overwrite"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FREETURN_UPDATE_ACK_FILE", ack)
	AcknowledgeUpdate()
	data, _ = os.ReadFile(ack)
	if string(data) != "do-not-overwrite" {
		t.Fatal("acknowledgement overwrote an existing file")
	}
	outside := filepath.Join(t.TempDir(), "restart-ready")
	t.Setenv("FREETURN_UPDATE_ACK_FILE", outside)
	AcknowledgeUpdate()
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatal("wrote acknowledgement outside the installation stage")
	}
}
