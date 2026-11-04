package backend

import (
	"testing"
	"time"
)

func TestTrafficMeterRepeatedReadersAndReset(t *testing.T) {
	var meter trafficMeter
	now := time.Now()
	meter.sample(0, 0, now)
	tx, rx := meter.sample(1024, 2048, now.Add(time.Second))
	if tx != 1024 || rx != 2048 {
		t.Fatalf("rates: %d %d", tx, rx)
	}
	tx, rx = meter.sample(1024, 2048, now.Add(1100*time.Millisecond))
	if tx != 1024 || rx != 2048 {
		t.Fatal("another reader consumed the rates")
	}
	tx, rx = meter.sample(0, 0, now.Add(2*time.Second))
	if tx != 0 || rx != 0 {
		t.Fatal("counter reset did not clear rates")
	}
}
