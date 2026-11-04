//go:build !linux

package backend

import (
	"context"
	"errors"
)

func probeServerPing(context.Context, string) (float64, error) {
	return 0, errors.New("server ICMP probe is only supported on Linux")
}
