//go:build linux

package wintunnel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/samosvalishe/free-turn-proxy/desktop/backend/vpnservice"
	"github.com/samosvalishe/free-turn-proxy/mobile"
)

type helperRequest struct {
	Op         string               `json:"op"`
	Params     *mobile.TunnelParams `json:"params,omitempty"`
	Exclusions []string             `json:"exclusions,omitempty"`
}

type helperReply struct {
	Error string `json:"error,omitempty"`
	Dev4  string `json:"dev4,omitempty"`
	Dev6  string `json:"dev6,omitempty"`
}

type helperClient struct {
	conn    *net.UnixConn
	mu      sync.Mutex
	exited  <-chan struct{}
	replies <-chan helperReply
}

func newHelperClient(conn *net.UnixConn) *helperClient {
	exited := make(chan struct{})
	replies := make(chan helperReply, 1)
	client := &helperClient{conn: conn, exited: exited, replies: replies}
	go func() {
		defer close(exited)
		defer close(replies)
		decoder := json.NewDecoder(conn)
		for {
			var reply helperReply
			if err := decoder.Decode(&reply); err != nil {
				return
			}
			replies <- reply
		}
	}()
	return client
}

func (c *helperClient) Close() {
	// EOF requests cleanup, while the read side stays open until the helper
	// has restored routes/DNS and closed its endpoint. This also works when
	// systemd, rather than pkexec, owns the helper process.
	_ = c.conn.CloseWrite()
	select {
	case <-c.exited:
	case <-time.After(10 * time.Second):
	}
	_ = c.conn.Close()
}

func (c *helperClient) call(req helperRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := json.NewEncoder(c.conn).Encode(req); err != nil {
		return err
	}
	reply, ok := <-c.replies
	if !ok {
		return io.EOF
	}
	if reply.Error != "" {
		return errors.New(reply.Error)
	}
	return nil
}

func startHelper(params *mobile.TunnelParams) (*helperClient, int, string, string, error) {
	// A system-installed helper is trusted root-owned code, not a passwordless
	// execution permission for a replaceable binary in the user's home.
	service, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: vpnservice.SocketPath(os.Getuid()), Net: "unix"})
	if err == nil {
		return initializeHelper(service, params)
	}
	if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, unix.ECONNREFUSED) {
		return nil, -1, "", "", fmt.Errorf("connect system VPN helper: %w", err)
	}
	if _, err := exec.LookPath("pkexec"); err != nil {
		return nil, -1, "", "", errors.New("polkit (pkexec) is required for one-click VPN")
	}
	workDir, err := os.MkdirTemp("", "freeturn-vpn-")
	if err != nil {
		return nil, -1, "", "", err
	}
	defer func() { _ = os.RemoveAll(workDir) }()
	socketPath := filepath.Join(workDir, "helper.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return nil, -1, "", "", err
	}
	defer func() { _ = listener.Close() }()

	exe, err := os.Executable()
	if err != nil {
		return nil, -1, "", "", err
	}
	cmd := exec.Command("pkexec", exe, "--vpn-helper", socketPath) //nolint:gosec,noctx
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if startErr := cmd.Start(); startErr != nil {
		return nil, -1, "", "", fmt.Errorf("start Polkit: %w", startErr)
	}
	exited := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(exited)
	}()
	type accepted struct {
		conn *net.UnixConn
		err  error
	}
	acceptCh := make(chan accepted, 1)
	go func() {
		c, e := listener.AcceptUnix()
		acceptCh <- accepted{conn: c, err: e}
	}()

	var conn *net.UnixConn
	select {
	case a := <-acceptCh:
		if a.err != nil {
			return nil, -1, "", "", a.err
		}
		conn = a.conn
	case <-exited:
		return nil, -1, "", "", fmt.Errorf("VPN authorization failed: %v: %s", waitErr, stderr.String())
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill()
		return nil, -1, "", "", errors.New("VPN authorization timed out")
	}
	return initializeHelper(conn, params)
}

func initializeHelper(conn *net.UnixConn, params *mobile.TunnelParams) (*helperClient, int, string, string, error) {
	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		_ = conn.Close()
		return nil, -1, "", "", err
	}
	// The helper must actually be privileged; a local process must not be able
	// to impersonate it on the temporary socket.
	if peerErr := requireRootPeer(conn); peerErr != nil {
		_ = conn.Close()
		return nil, -1, "", "", peerErr
	}
	if encodeErr := json.NewEncoder(conn).Encode(helperRequest{Op: "init", Params: params}); encodeErr != nil {
		_ = conn.Close()
		return nil, -1, "", "", encodeErr
	}
	data := make([]byte, 4096)
	oob := make([]byte, unix.CmsgSpace(4))
	n, oobn, _, _, err := conn.ReadMsgUnix(data, oob)
	if err != nil {
		_ = conn.Close()
		return nil, -1, "", "", err
	}
	var reply helperReply
	if decodeErr := json.Unmarshal(bytes.TrimSpace(data[:n]), &reply); decodeErr != nil {
		_ = conn.Close()
		return nil, -1, "", "", decodeErr
	}
	if reply.Error != "" {
		_ = conn.Close()
		return nil, -1, "", "", errors.New(reply.Error)
	}
	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil || len(msgs) != 1 {
		_ = conn.Close()
		return nil, -1, "", "", errors.New("VPN helper did not pass a TUN descriptor")
	}
	fds, err := unix.ParseUnixRights(&msgs[0])
	if err != nil || len(fds) != 1 {
		for _, fd := range fds {
			_ = unix.Close(fd)
		}
		_ = conn.Close()
		return nil, -1, "", "", errors.New("invalid TUN descriptor from VPN helper")
	}
	unix.CloseOnExec(fds[0])
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = unix.Close(fds[0])
		_ = conn.Close()
		return nil, -1, "", "", err
	}
	return newHelperClient(conn), fds[0], reply.Dev4, reply.Dev6, nil
}

func requireRootPeer(conn *net.UnixConn) error {
	return requirePeerUID(conn, 0)
}

func requirePeerUID(conn *net.UnixConn, uid uint32) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var peer *unix.Ucred
	if controlErr := raw.Control(func(fd uintptr) { peer, err = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) }); controlErr != nil {
		return controlErr
	}
	if err != nil {
		return err
	}
	if peer == nil || peer.Uid != uid {
		return fmt.Errorf("unauthorized VPN socket peer (expected UID %d)", uid)
	}
	return nil
}
