package session

import (
	"context"
	"testing"
	"time"
)

func TestWatchCaptchaKeepsWorkingStreamsConnected(t *testing.T) {
	s := newTestSession(t, Options{
		ConnectTimeout: 20 * time.Millisecond,
		StatusInterval: 5 * time.Millisecond,
	}, func() bool { return true })
	s.connected.Store(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.watch(ctx, cancel) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("watch: %v", err)
		}
	}()
	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("working stream lost connected state: %+v", s.Snapshot())
		case <-tick.C:
			snap := s.Snapshot()
			if snap.Phase == PhaseConnected && snap.Streams == 1 {
				return
			}
		}
	}
}
