package updates

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTransferEstimate(t *testing.T) {
	for _, test := range []struct {
		received, total int64
		elapsed         time.Duration
		rate, remaining int64
	}{
		{1024, 4096, 2 * time.Second, 512, 6},
		{0, 4096, time.Second, 0, -1},
		{1024, 4096, 10 * time.Millisecond, 0, -1},
		{1000, 1000, time.Second, 1000, 0},
		{2000, 1000, time.Second, 2000, 0},
		{10, 0, time.Second, 10, -1},
	} {
		rate, remaining := transferEstimate(test.received, test.total, test.elapsed)
		if rate != test.rate || remaining != test.remaining {
			t.Fatalf("estimate %+v: rate=%d, remaining=%d", test, rate, remaining)
		}
	}
}

type phaseReader struct {
	io.ReadCloser
	manager *Manager
	t       *testing.T
}

func (r phaseReader) Read(data []byte) (int, error) {
	if state := r.manager.State(); state.Phase != "downloading" {
		r.t.Fatalf("streaming phase = %s", state.Phase)
	}
	return r.ReadCloser.Read(data)
}

func TestDownloadReportsActualPhases(t *testing.T) {
	m, _, _ := fixture(t, "v2.0.0")
	original := m.client.Transport
	m.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		state := m.State()
		response, err := original.RoundTrip(request)
		if err != nil {
			return nil, err
		}
		switch {
		case request.URL.String() == m.apiURL:
			if state.Phase != "checking" {
				t.Fatalf("checking phase = %s", state.Phase)
			}
		case strings.HasSuffix(request.URL.Path, "checksums.txt"):
			if state.Phase != "checksum" {
				t.Fatalf("checksum phase = %s", state.Phase)
			}
		default:
			if state.Phase != "connecting" {
				t.Fatalf("connecting phase = %s", state.Phase)
			}
			response.Body = phaseReader{response.Body, m, t}
		}
		return response, nil
	})
	if err := m.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.Download(context.Background(), filepath.Join(t.TempDir(), "update.tar.gz"), "v2.1.0"); err != nil {
		t.Fatal(err)
	}
	if state := m.State(); state.Phase != "downloaded" || state.BytesPerSecond != 0 || state.RemainingSeconds != -1 {
		t.Fatalf("stale transfer: %+v", state)
	}
}

func TestRestartAndErrorsClearStaleProgress(t *testing.T) {
	m := New("dev", "amd64", "example/fork")
	m.state.Received, m.state.Total = 1024, 2048
	if err := m.Restart(func() error {
		if state := m.State(); state.Phase != "restarting" || state.Busy != "restarting" || state.Received != 0 {
			t.Fatalf("restart state: %+v", state)
		}
		return io.ErrClosedPipe
	}); err == nil {
		t.Fatal("lost worker error")
	}
	if state := m.State(); state.Phase != "error" || state.Busy != "" || state.Error == "" {
		t.Fatalf("error state: %+v", state)
	}
}
