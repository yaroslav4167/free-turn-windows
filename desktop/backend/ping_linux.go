//go:build linux

package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func probeServerPing(ctx context.Context, host string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ping", "-n", "-c", "1", "-W", "2", "-w", "2", "--", host)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ICMP probe unavailable: %w", err)
	}
	return parsePingTime(string(output))
}
