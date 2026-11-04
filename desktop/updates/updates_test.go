package updates

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fixture(t *testing.T, current string) (*Manager, *[]byte, *string) {
	t.Helper()
	m := New(current, "amd64", "example/fork")
	archive := []byte("archive fixture")
	name := "FreeTurn-Linux-amd64-v2.1.0.tar.gz"
	checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), name)
	m.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := r.Context().Err(); err != nil {
			return nil, err
		}
		var body string
		switch {
		case r.URL.String() == m.apiURL:
			if r.URL.Path != "/repos/example/fork/releases/latest" {
				t.Fatal("repository not injected")
			}
			data, _ := json.Marshal(release{Tag: "v2.1.0", Assets: []asset{
				{Name: name, URL: "https://github.com/example/fork/releases/download/v2.1.0/" + name, Size: int64(len("archive fixture"))},
				{Name: "checksums.txt", URL: "https://github.com/example/fork/releases/download/v2.1.0/checksums.txt"},
			}})
			body = string(data)
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			body = checksum
		default:
			body = string(archive)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	return m, &archive, &checksum
}

func TestVersionComparison(t *testing.T) {
	for _, test := range []struct {
		latest, current string
		want            bool
	}{
		{"v2.10.0", "v2.9.0", true}, {"v2.0", "2.0.0", false},
		{"v2.1.0", "v3.0.0", false}, {"v2.1.1", "v2.1.0", true},
	} {
		if got := newer(test.latest, test.current); got != test.want {
			t.Fatalf("%s > %s = %v", test.latest, test.current, got)
		}
	}
	for _, value := range []string{"dev", "v2.0.0-rc1", "v2.0.0+build", "v2", "v02.0.0", "v2.0.0/evil", "v999999999999999999999.0.0"} {
		if _, ok := parseVersion(value); ok {
			t.Fatalf("accepted unstable or invalid version %s", value)
		}
	}
}

func TestCheck(t *testing.T) {
	for _, test := range []struct {
		current          string
		available, known bool
	}{
		{"v2.0", true, true}, {"v2.1.0", false, true}, {"v3.0.0", false, true}, {"dev", true, false},
	} {
		m, _, _ := fixture(t, test.current)
		if err := m.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		state := m.State()
		if state.Available != test.available || state.CurrentKnown != test.known || state.Busy != "" || state.CheckedAt == "" {
			t.Fatalf("bad state: %+v", state)
		}
		if state.Repository != "example/fork" || !strings.Contains(state.ReleaseURL, "/example/fork/") {
			t.Fatal("wrong repository")
		}
	}
}

func TestCheckRejectsUnusableReleases(t *testing.T) {
	for _, test := range []struct {
		name   string
		modify func(*release)
	}{
		{"no assets", func(r *release) { r.Assets = nil }},
		{"prerelease", func(r *release) { r.Prerelease = true }},
		{"draft", func(r *release) { r.Draft = true }},
		{"bad version", func(r *release) { r.Tag = "nightly" }},
		{"other architecture", func(r *release) { r.Assets[0].Name = "FreeTurn-Linux-arm64-v2.1.0.tar.gz" }},
		{"external asset", func(r *release) { r.Assets[0].URL = "https://evil.example/file" }},
		{"other repo", func(r *release) { r.Assets[0].URL = "https://github.com/other/repo/releases/download/v2.1.0/file" }},
		{"oversize", func(r *release) { r.Assets[0].Size = maxArchive + 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, _, _ := fixture(t, "v2.0.0")
			original := m.client.Transport
			m.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				response, err := original.RoundTrip(req)
				if err != nil {
					return nil, err
				}
				var r release
				if err := json.NewDecoder(response.Body).Decode(&r); err != nil {
					t.Fatal(err)
				}
				response.Body.Close()
				test.modify(&r)
				body, _ := json.Marshal(r)
				response.Body = io.NopCloser(strings.NewReader(string(body)))
				return response, nil
			})
			if err := m.Check(context.Background()); err == nil {
				t.Fatal("accepted unusable release")
			}
			if state := m.State(); state.Available || state.Error == "" || state.Busy != "" {
				t.Fatalf("bad state: %+v", state)
			}
		})
	}
}

func TestDownloadVerifiedAndAtomic(t *testing.T) {
	for _, mode := range []string{"valid", "tampered", "truncated", "checksum missing", "checksum duplicate", "checksum invalid", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			m, archive, checksum := fixture(t, "v2.0.0")
			if err := m.Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "tampered":
				*archive = []byte("corrupt fixture")
			case "truncated":
				*archive = []byte("short")
			case "checksum missing":
				*checksum = ""
			case "checksum duplicate":
				*checksum += *checksum
			case "checksum invalid":
				*checksum = "nope  " + m.State().AssetName
			}
			directory := t.TempDir()
			destination := filepath.Join(directory, "download.tar.gz")
			if err := os.WriteFile(destination, []byte("existing file"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			err := m.Download(ctx, destination, "v2.1.0")
			if (err == nil) != (mode == "valid") {
				t.Fatalf("unexpected download error: %v", err)
			}
			contents, _ := os.ReadFile(destination)
			if mode == "valid" {
				if string(contents) != string(*archive) || m.State().DownloadedPath != destination || m.State().Received != int64(len(*archive)) {
					t.Fatal("download not published")
				}
			} else if string(contents) != "existing file" || m.State().DownloadedPath != "" {
				t.Fatal("failed download overwrote existing destination")
			}
			files, _ := os.ReadDir(directory)
			if len(files) != 1 || m.State().Busy != "" {
				t.Fatal("temporary file or busy state leaked")
			}
		})
	}
}

func TestDownloadNeedsCheckAndMatchingTag(t *testing.T) {
	m, _, _ := fixture(t, "v2.0.0")
	if err := m.Download(context.Background(), "/unused", "v2.1.0"); err == nil {
		t.Fatal("downloaded unchecked release")
	}
	if err := m.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.Download(context.Background(), "/unused", "v2.2.0"); err == nil {
		t.Fatal("downloaded changed release")
	}
	if err := m.begin("checking"); err != nil {
		t.Fatal(err)
	}
	if err := m.Check(context.Background()); err == nil {
		t.Fatal("allowed concurrent operation")
	}
	m.finish(nil)
}

func TestRepositoryAndRedirectValidation(t *testing.T) {
	for _, repo := range []string{"", "../evil", "https://github.com/example/fork", "example/fork/extra"} {
		m := New("dev", "amd64", repo)
		if m.State().Repository != "" {
			t.Fatalf("invalid repo: %s", repo)
		}
		if err := m.Check(context.Background()); err == nil {
			t.Fatal("checked invalid repository")
		}
	}
	m := New("v2.0.0", "amd64", "example/fork")
	for _, address := range []string{"http://github.com/file", "https://evil.example/file", "https://github.com.evil.example/file", "https://github.com:8443/file", "https://user:pass@github.com/file"} {
		request, _ := http.NewRequest("GET", address, nil)
		if err := m.client.CheckRedirect(request, nil); err == nil {
			t.Fatalf("unsafe redirect: %s", address)
		}
	}
	for _, address := range []string{"https://github.com/file", "https://release-assets.githubusercontent.com/file", "https://objects.githubusercontent.com/file"} {
		request, _ := http.NewRequest("GET", address, nil)
		if err := m.client.CheckRedirect(request, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHTTPFailures(t *testing.T) {
	for _, code := range []int{403, 404, 429, 500} {
		m := New("v2.0.0", "amd64", "example/fork")
		m.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("private response"))}, nil
		})
		if err := m.Check(context.Background()); err == nil {
			t.Fatal("ignored HTTP failure")
		}
		if strings.Contains(m.State().Error, "private response") {
			t.Fatal("response body leaked")
		}
	}
}
