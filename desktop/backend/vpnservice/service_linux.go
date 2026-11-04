//go:build linux

// Package vpnservice installs an optional, per-user systemd socket for the
// root-owned headless helper. It never grants privilege to the GUI executable.
package vpnservice

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const HelperPath = "/usr/local/libexec/freeturn-vpn-helper"
const unitDir = "/etc/systemd/system"
const systemctl = "/usr/bin/systemctl"

func SocketPath(uid int) string     { return fmt.Sprintf("/run/freeturn-vpn/%d.sock", uid) }
func socketUnit(uid uint32) string  { return fmt.Sprintf("freeturn-vpn-%d.socket", uid) }
func serviceUnit(uid uint32) string { return fmt.Sprintf("freeturn-vpn-%d@.service", uid) }

func validateUID(uid uint32) error {
	if uid == 0 || uid == ^uint32(0) {
		return errors.New("a non-root user UID is required")
	}
	return nil
}

func units(uid uint32) (string, string, error) {
	if err := validateUID(uid); err != nil {
		return "", "", err
	}
	socket := fmt.Sprintf(`[Unit]
Description=FreeTurn VPN socket for UID %d

[Socket]
ListenStream=%s
SocketUser=%d
SocketGroup=root
SocketMode=0600
DirectoryMode=0755
Accept=yes
MaxConnections=2
RemoveOnStop=yes
Service=%s

[Install]
WantedBy=sockets.target
`, uid, SocketPath(int(uid)), uid, serviceUnit(uid))
	service := fmt.Sprintf(`[Unit]
Description=FreeTurn VPN helper for UID %d

[Service]
Type=simple
ExecStart=%s --uid %d
User=root
Group=root
StandardInput=socket
StandardOutput=journal
StandardError=journal
Environment=PATH=/usr/sbin:/usr/bin:/sbin:/bin
UMask=0077
NoNewPrivileges=yes
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictSUIDSGID=yes
LockPersonality=yes
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK
DevicePolicy=closed
DeviceAllow=/dev/net/tun rw
TasksMax=32
MemoryMax=128M
LimitNOFILE=128
KillMode=mixed
TimeoutStopSec=15
`, uid, HelperPath, uid)
	return socket, service, nil
}

// secureDirectory rejects symlinks and directories writable by non-root
// accounts, including any ancestor. Only these directories receive root files.
func secureDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return errors.New("system path must be absolute")
	}
	current := "/"
	for _, component := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.IsDir() || stat.Uid != 0 || info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("unsafe system directory %s: must be root-owned and not writable by other users", current)
		}
	}
	return nil
}

func atomicWrite(path string, source io.Reader, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".freeturn-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, source); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func control(args ...string) error {
	cmd := exec.Command(systemctl, args...) //nolint:noctx
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(out))
	}
	return nil
}

func checkInstaller(uid uint32) error {
	if os.Geteuid() != 0 {
		return errors.New("system VPN helper installation requires administrator authorization")
	}
	if err := validateUID(uid); err != nil {
		return err
	}
	if _, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err != nil {
		return err
	}
	// pkexec/sudo provide the identity that authorized the installation. Do not
	// silently install access for another account from the GUI.
	for _, variable := range []string{"PKEXEC_UID", "SUDO_UID"} {
		if value := os.Getenv(variable); value != "" && value != "0" && value != strconv.FormatUint(uint64(uid), 10) {
			return errors.New("VPN helper user does not match the authorizing account")
		}
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return errors.New("systemd is required; the existing pkexec mode remains available")
	}
	return secureDirectory(unitDir)
}

// Install is called only after explicit administrator authorization. The source
// belongs to the application package being installed; subsequent connections
// execute only the protected system copy, never this user-writable source.
func Install(uid uint32, source string) error {
	if err := checkInstaller(uid); err != nil {
		return err
	}
	if err := secureDirectory(filepath.Dir(filepath.Dir(HelperPath))); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(HelperPath), 0o755); err != nil {
		return err
	}
	if err := secureDirectory(filepath.Dir(HelperPath)); err != nil {
		return err
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("VPN helper source must be a regular executable")
	}
	var magic [4]byte
	if _, err := io.ReadFull(file, magic[:]); err != nil || string(magic[:]) != "\x7fELF" {
		return errors.New("VPN helper must be a compiled Linux executable")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := atomicWrite(HelperPath, file, 0o755); err != nil {
		return err
	}
	socket, service, err := units(uid)
	if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(unitDir, serviceUnit(uid)), strings.NewReader(service), 0o644); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(unitDir, socketUnit(uid)), strings.NewReader(socket), 0o644); err != nil {
		return err
	}
	if err := control("daemon-reload"); err != nil {
		return err
	}
	return control("enable", "--now", socketUnit(uid))
}

// Remove revokes only this user's socket access, including active connections.
// The shared root-owned binary is retained for other users and future updates.
func Remove(uid uint32) error {
	if err := checkInstaller(uid); err != nil {
		return err
	}
	if err := control("disable", "--now", socketUnit(uid)); err != nil {
		return err
	}
	if err := control("stop", fmt.Sprintf("freeturn-vpn-%d@*.service", uid)); err != nil {
		return err
	}
	for _, name := range []string{socketUnit(uid), serviceUnit(uid)} {
		if err := os.Remove(filepath.Join(unitDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return control("daemon-reload")
}

func Available(uid int) bool {
	info, err := os.Stat(SocketPath(uid))
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(uid) && info.Mode().Perm() == 0o600
}
