package backend

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/samosvalishe/free-turn-proxy/mobile"
)

type PingResult struct {
	ServerID     string  `json:"serverId"`
	Host         string  `json:"host"`
	Milliseconds float64 `json:"milliseconds"`
}

var dnsPingHost = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.?$`)
var pingTime = regexp.MustCompile(`\btime[=<]\s*([0-9]+(?:\.[0-9]+)?)\s*ms\b`)

func pingHost(peer string) (string, error) {
	host, _, err := net.SplitHostPort(peer)
	if err != nil || host == "" || len(host) > 253 || strings.ContainsAny(host, "\r\n\x00") {
		return "", errors.New("invalid ping host")
	}
	if net.ParseIP(host) == nil && !dnsPingHost.MatchString(host) {
		return "", errors.New("invalid ping host")
	}
	return host, nil
}

func parsePingTime(output string) (float64, error) {
	match := pingTime.FindStringSubmatch(output)
	if len(match) != 2 {
		return 0, errors.New("no ICMP echo reply")
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil || value < 0 || value > 60000 {
		return 0, errors.New("invalid ICMP latency")
	}
	return value, nil
}

// MeasurePing probes only the current profile's server, never an arbitrary UI
// destination. It runs without elevation and does not change VPN routes.
func (e *Engine) MeasurePing(serverID string) (PingResult, error) {
	e.mu.Lock()
	if serverID == "" || e.serverID != serverID || mobile.GetState().State != mobile.StateConnected || e.store == nil {
		e.mu.Unlock()
		return PingResult{}, errors.New("ping requires an active connection")
	}
	if e.pingCancel != nil {
		e.mu.Unlock()
		return PingResult{}, errors.New("ping is already running")
	}
	server, ok := e.store.Get(serverID)
	if !ok {
		e.mu.Unlock()
		return PingResult{}, errors.New("server profile not found")
	}
	host, err := pingHost(server.Peer)
	if err != nil {
		e.mu.Unlock()
		return PingResult{}, err
	}
	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	e.pingGeneration++
	generation := e.pingGeneration
	e.pingCancel = cancel
	e.mu.Unlock()
	defer cancel()
	latency, probeErr := probeServerPing(ctx, host)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pingGeneration != generation || e.serverID != serverID || mobile.GetState().State != mobile.StateConnected {
		return PingResult{}, context.Canceled
	}
	e.pingCancel = nil
	if probeErr != nil {
		return PingResult{}, probeErr
	}
	return PingResult{ServerID: serverID, Host: host, Milliseconds: latency}, nil
}

func (e *Engine) cancelPingLocked() {
	if e.pingCancel != nil {
		e.pingCancel()
		e.pingCancel = nil
	}
	e.pingGeneration++
}
