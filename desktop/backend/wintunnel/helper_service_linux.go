//go:build linux

package wintunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
)

// RunActivatedHelper serves exactly one systemd Accept=yes connection. The
// socket ACL and SO_PEERCRED independently restrict access to the installed UID.
// There is no command execution, arbitrary file access or user-supplied path API.
func RunActivatedHelper(ctx context.Context, allowedUID uint32) error {
	if os.Geteuid() != 0 || allowedUID == 0 || allowedUID == ^uint32(0) {
		return errors.New("VPN service requires root and a non-root client UID")
	}
	file := os.NewFile(0, "systemd-vpn-connection")
	conn, err := net.FileConn(file)
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("open systemd connection: %w", err)
	}
	defer conn.Close()
	unixConn, ok := conn.(*net.UnixConn)
	if !ok || unixConn.LocalAddr().Network() != "unix" {
		return errors.New("VPN service requires a Unix stream socket")
	}
	if err := requirePeerUID(unixConn, allowedUID); err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	return serveHelper(unixConn)
}
