package backend

import "time"

// trafficMeter samples cumulative counters, independently of mobile.GetState's
// read-to-read rates. Multiple UI readers must not consume each other's sample.
type trafficMeter struct {
	at             time.Time
	tx, rx         int64
	txRate, rxRate int64
}

func (m *trafficMeter) sample(tx, rx int64, now time.Time) (int64, int64) {
	if m.at.IsZero() || tx < m.tx || rx < m.rx {
		*m = trafficMeter{at: now, tx: tx, rx: rx}
		return 0, 0
	}
	elapsed := now.Sub(m.at)
	if elapsed >= time.Second {
		m.txRate = int64(float64(tx-m.tx) / elapsed.Seconds())
		m.rxRate = int64(float64(rx-m.rx) / elapsed.Seconds())
		m.at, m.tx, m.rx = now, tx, rx
	}
	return m.txRate, m.rxRate
}
