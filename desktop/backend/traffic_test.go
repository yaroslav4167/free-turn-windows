package backend

import (
	"testing"
	"time"
)

func TestTrafficMeter(t *testing.T) {
	var meter trafficMeter
	now := time.Now()
	meter.sample(0, 0, now)
	if tx, rx := meter.sample(1024, 2048, now.Add(time.Second)); tx != 1024 || rx != 2048 {
		t.Fatalf("rates = %d/%d, want 1024/2048", tx, rx)
	}
	// Another reader a millisecond later must not overwrite the displayed rates
	// with zero (or consume the next sample's counters).
	if tx, rx := meter.sample(1024, 2048, now.Add(time.Second+time.Millisecond)); tx != 1024 || rx != 2048 {
		t.Fatalf("second reader consumed rates: %d/%d", tx, rx)
	}
	if tx, rx := meter.sample(3072, 6144, now.Add(2*time.Second)); tx != 2048 || rx != 4096 {
		t.Fatalf("next rates = %d/%d, want 2048/4096", tx, rx)
	}
	if tx, rx := meter.sample(3072, 6144, now.Add(3*time.Second)); tx != 0 || rx != 0 {
		t.Fatalf("idle rates = %d/%d, want zero", tx, rx)
	}
}

func TestTrafficMeterReconnect(t *testing.T) {
	var meter trafficMeter
	now := time.Now()
	meter.sample(4096, 8192, now)
	if tx, rx := meter.sample(0, 0, now.Add(time.Second)); tx != 0 || rx != 0 {
		t.Fatalf("reset rates = %d/%d, want zero", tx, rx)
	}
	if tx, rx := meter.sample(1000, 2000, now.Add(2*time.Second)); tx != 1000 || rx != 2000 {
		t.Fatalf("new session rates = %d/%d", tx, rx)
	}
}
