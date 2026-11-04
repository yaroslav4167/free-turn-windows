//go:build windows

package backend

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// .NET's ICMP API produces a numeric RTT independently of Windows UI language
// and supports both IPv4 and IPv6 without parsing localized ping.exe output.
func probeServerPing(ctx context.Context, host string) (float64, error) {
	if _, err := pingHost("[" + host + "]:1"); err != nil {
		// SplitHostPort requires unbracketed DNS/IPv4 hosts.
		if _, err := pingHost(host + ":1"); err != nil {
			return 0, err
		}
	}
	script := "$ErrorActionPreference='Stop';$p=New-Object System.Net.NetworkInformation.Ping;try{$r=$p.Send('" + host + "',1500);if($r.Status -ne 'Success'){exit 1};[Console]::Write($r.RoundtripTime)}finally{$p.Dispose()}"
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ICMP: %w", err)
	}
	rtt, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || rtt < 0 || rtt > 60000 {
		return 0, fmt.Errorf("invalid ICMP response")
	}
	return rtt, nil
}
