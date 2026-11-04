//go:build linux

package wintunnel

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func helperSocketPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	connections := make([]*net.UnixConn, 2)
	for i, fd := range fds {
		file := os.NewFile(uintptr(fd), "helper-test")
		conn, err := net.FileConn(file)
		_ = file.Close()
		if err != nil {
			t.Fatal(err)
		}
		connections[i] = conn.(*net.UnixConn)
		t.Cleanup(func() { _ = conn.Close() })
	}
	return connections[0], connections[1]
}

func TestHelperClientWaitsForCleanup(t *testing.T) {
	local, remote := helperSocketPair(t)
	client := newHelperClient(local)
	cleaned := make(chan struct{})
	go func() {
		defer remote.Close()
		reader := json.NewDecoder(remote)
		var req helperRequest
		if reader.Decode(&req) != nil {
			return
		}
		_ = json.NewEncoder(remote).Encode(helperReply{})
		if err := reader.Decode(&req); err == io.EOF {
			time.Sleep(10 * time.Millisecond) // simulate deferred route/DNS cleanup
			close(cleaned)
		}
	}()
	if err := client.call(helperRequest{Op: "routes"}); err != nil {
		t.Fatal(err)
	}
	client.Close()
	select {
	case <-cleaned:
	default:
		t.Fatal("Close returned before helper cleanup finished")
	}
}

func TestHelperClientDetectsServiceExit(t *testing.T) {
	local, remote := helperSocketPair(t)
	client := newHelperClient(local)
	_ = remote.Close()
	select {
	case <-client.exited:
	case <-time.After(time.Second):
		t.Fatal("service exit was not detected")
	}
	if err := client.call(helperRequest{Op: "routes"}); err == nil {
		t.Fatal("accepted request after service exit")
	}
}

func TestHelperPeerUID(t *testing.T) {
	local, _ := helperSocketPair(t)
	if err := requirePeerUID(local, uint32(os.Getuid())); err != nil {
		t.Fatal(err)
	}
	if err := requirePeerUID(local, uint32(os.Getuid()+1)); err == nil {
		t.Fatal("accepted another user's connection")
	}
}

func TestHelperRequestSizeBounded(t *testing.T) {
	reader := bufio.NewReaderSize(strings.NewReader(strings.Repeat("x", 128*1024)), 64*1024+1)
	if err := readRequest(reader, &helperRequest{}); err == nil {
		t.Fatal("accepted oversized request")
	}
}
