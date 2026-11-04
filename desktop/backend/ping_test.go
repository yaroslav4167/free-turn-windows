package backend

import (
	"context"
	"testing"
)

func TestPingHostValidation(t *testing.T) {
	for _, peer := range []string{"example.com:56000", "192.0.2.1:56000", "[2001:db8::1]:56000"} {
		if _, err := pingHost(peer); err != nil {
			t.Fatalf("valid peer %s: %v", peer, err)
		}
	}
	for _, peer := range []string{"", "example.com", "-c:1", "bad host:56000", "bad\nhost:56000", ";command:1"} {
		if _, err := pingHost(peer); err == nil {
			t.Fatalf("invalid peer accepted: %q", peer)
		}
	}
}

func TestPingTime(t *testing.T) {
	for output, want := range map[string]float64{
		"64 bytes from 192.0.2.1: icmp_seq=1 ttl=64 time=42.7 ms": 42.7,
		"64 bytes from ::1: icmp_seq=1 ttl=64 time=0.032 ms":      0.032,
		"64 bytes from host: time<1 ms":                           1,
	} {
		got, err := parsePingTime(output)
		if err != nil || got != want {
			t.Fatalf("%q: %v, %v", output, got, err)
		}
	}
	for _, output := range []string{"100% packet loss", "time=NaN ms", "time=-1 ms", "time=90000 ms", ""} {
		if _, err := parsePingTime(output); err == nil {
			t.Fatalf("invalid latency accepted: %q", output)
		}
	}
}

func TestPingCancellationAndInactiveGate(t *testing.T) {
	e := &Engine{}
	if _, err := e.MeasurePing("not-connected"); err == nil {
		t.Fatal("ping allowed without an active profile")
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.pingCancel = cancel
	e.cancelPingLocked()
	if ctx.Err() != context.Canceled || e.pingCancel != nil || e.pingGeneration != 1 {
		t.Fatal("probe was not cancelled and invalidated")
	}
}
