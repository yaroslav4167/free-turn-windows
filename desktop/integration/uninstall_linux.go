//go:build linux

package integration

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Uninstall revokes system access before removing user files. Profiles, unknown
// files and update backups are deliberately retained. Run with the GUI closed.
func Uninstall() error {
	if os.Getuid() == 0 {
		return errors.New("запустите деинсталлятор от обычного пользователя")
	}
	uid := strconv.Itoa(os.Getuid())
	for _, suffix := range []string{".socket", "@.service"} {
		_, err := os.Lstat("/etc/systemd/system/freeturn-vpn-" + uid + suffix)
		if err == nil {
			if err := ConfigureVPNHelper(true); err != nil {
				return err
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
	}
	return uninstallUserFiles()
}

func removeManagedEntry(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), "\nX-FreeTurn-Managed=true\n") {
		return nil
	}
	return os.Remove(path)
}

func uninstallUserFiles() error {
	data, err := dataDir()
	if err != nil {
		return err
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	entries := []string{filepath.Join(data, "applications", "freeturn.desktop"), filepath.Join(config, "autostart", "freeturn.desktop")}
	if desktop, err := desktopDirectory(); err == nil {
		entries = append(entries, filepath.Join(desktop, "freeturn.desktop"))
	}
	for _, entry := range entries {
		if err := removeManagedEntry(entry); err != nil {
			return err
		}
	}
	for _, name := range []string{
		"FreeTurnDesktop/freeturn-desktop", "FreeTurnDesktop/freeturn-vpn-helper",
		"icons/hicolor/scalable/apps/freeturn.svg", "icons/hicolor/512x512/apps/freeturn.png", "icons/hicolor/64x64/apps/freeturn.png",
	} {
		if err := os.Remove(filepath.Join(data, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	// Never recurse: user files and update backups must survive.
	_ = os.Remove(filepath.Join(data, "FreeTurnDesktop"))
	return nil
}
