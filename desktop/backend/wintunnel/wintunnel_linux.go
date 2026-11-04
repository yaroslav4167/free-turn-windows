//go:build linux

package wintunnel

import (
	"errors"
	"fmt"
	"sync"

	amtun "github.com/amnezia-vpn/amneziawg-go/v3/tun"
	"golang.org/x/sys/unix"

	"github.com/samosvalishe/free-turn-proxy/mobile"
)

const DefaultMTU = 1280

var mu sync.Mutex
var active *helperClient
var connecting bool
var generation uint64

// Available reports whether this build supports the embedded VPN mode.
func Available() bool { return true }

type linuxProtector struct{ v4, v6 string }

// Protect keeps the core's own sockets on the physical interface while the
// host's default routes point to the TUN device.
func (p linuxProtector) Protect(fd int) bool {
	dev := p.v4
	if family, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_DOMAIN); err == nil && family == unix.AF_INET6 && p.v6 != "" {
		dev = p.v6
	}
	return dev != "" && unix.SetsockoptString(fd, unix.SOL_SOCKET, unix.SO_BINDTODEVICE, dev) == nil
}

func Connect(configJSON, wgConfigText string, mtu int, exclusionIPs ...string) (err error) {
	return ConnectWithRoutes(configJSON, wgConfigText, mtu, "", exclusionIPs...)
}

func ConnectWithRoutes(configJSON, wgConfigText string, mtu int, routes string, exclusionIPs ...string) (err error) {
	mu.Lock()
	if active != nil || connecting {
		mu.Unlock()
		return errors.New("linux VPN is already connecting or active")
	}
	connecting = true
	currentGeneration := generation
	mu.Unlock()
	defer func() {
		mu.Lock()
		connecting = false
		mu.Unlock()
	}()

	if mtu <= 0 {
		mtu = DefaultMTU
	}
	params, err := mobile.ParseTunnelConfig(wgConfigText, mtu)
	if err != nil {
		return err
	}
	if routes != "" {
		params.AllowedIPs = routes
	}
	client, tunFD, dev4, dev6, err := startHelper(params)
	if err != nil {
		return err
	}
	mu.Lock()
	if generation != currentGeneration {
		mu.Unlock()
		_ = unix.Close(tunFD)
		client.Close()
		return errors.New("VPN connection was cancelled")
	}
	active = client
	mu.Unlock()
	go watchHelper(client)
	defer func() {
		if err != nil {
			Disconnect()
		}
	}()

	protector := linuxProtector{v4: dev4, v6: dev6}
	// A failed bind would strand the relay after installing default routes.
	probe, socketErr := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if socketErr != nil {
		_ = unix.Close(tunFD)
		return socketErr
	}
	protected := protector.Protect(probe)
	_ = unix.Close(probe)
	if !protected {
		_ = unix.Close(tunFD)
		return fmt.Errorf("cannot bind relay sockets to physical interface %q", dev4)
	}
	mobile.SetProtect(protector)

	dev, _, devErr := amtun.CreateUnmonitoredTUNFromFD(tunFD)
	if devErr != nil {
		return fmt.Errorf("open Linux TUN: %w", devErr)
	}
	if err = mobile.StartTunnelDevice(configJSON, dev, nil); err != nil {
		return err
	}
	if err = waitConnected(connectTimeout); err != nil {
		return err
	}
	if err = client.call(helperRequest{Op: "routes", Params: params, Exclusions: exclusionIPs}); err != nil {
		return fmt.Errorf("configure VPN routes: %w", err)
	}
	return nil
}

func watchHelper(client *helperClient) {
	<-client.exited
	mu.Lock()
	if active == client {
		mobile.Stop()
		mobile.SetProtect(nil)
		active = nil
		generation++
	}
	mu.Unlock()
}

func Disconnect() {
	mu.Lock()
	generation++
	client := active
	active = nil
	mu.Unlock()
	mobile.Stop()
	mobile.SetProtect(nil)
	if client != nil {
		client.Close()
	}
}
