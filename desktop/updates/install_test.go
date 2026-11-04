package updates

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeELF(machine uint16, protocol bool) []byte {
	data := make([]byte, 64)
	copy(data, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], machine)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint16(data[52:], 64)
	binary.LittleEndian.PutUint16(data[54:], 56)
	binary.LittleEndian.PutUint16(data[58:], 64)
	if protocol {
		data = append(data, []byte(InstallProtocolMarker)...)
	}
	return data
}

type archiveEntry struct {
	name string
	data []byte
	kind byte
}

func archiveFixture(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gz)
	for _, entry := range entries {
		kind := entry.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0o7777, Typeflag: kind, Size: int64(len(entry.data))}
		if kind == tar.TypeSymlink || kind == tar.TypeLink {
			header.Linkname = "../outside"
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := tarWriter.Write(entry.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func validEntries() []archiveEntry {
	return []archiveEntry{{"./freeturn-desktop", fakeELF(62, true), 0}, {"./freeturn-vpn-helper", fakeELF(62, false), 0}, {"./README.md", []byte("not executed"), 0}}
}

func prepareFixture(t *testing.T) (stage, target string) {
	t.Helper()
	target = t.TempDir()
	var err error
	stage, err = os.MkdirTemp(target, ".freeturn-update-")
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(stage, "archive.tar.gz")
	if err := os.WriteFile(archive, archiveFixture(t, validEntries()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := stageArchive(context.Background(), archive, stage, "v3.0.0", "amd64"); err != nil {
		t.Fatal(err)
	}
	return stage, target
}

func TestArchiveRejectsUnsafeOrInvalidContents(t *testing.T) {
	for _, test := range []struct {
		name    string
		entries []archiveEntry
	}{
		{"traversal", append(validEntries(), archiveEntry{"../../outside", []byte("bad"), 0})},
		{"absolute", append(validEntries(), archiveEntry{"/outside", []byte("bad"), 0})},
		{"symlink", append(validEntries(), archiveEntry{"link", nil, tar.TypeSymlink})},
		{"hardlink", append(validEntries(), archiveEntry{"link", nil, tar.TypeLink})},
		{"duplicate", append(validEntries(), validEntries()[0])},
		{"helper missing", validEntries()[:1]},
		{"not ELF", []archiveEntry{{"freeturn-desktop", []byte("#!/bin/sh"), 0}, validEntries()[1]}},
		{"wrong arch", []archiveEntry{{"freeturn-desktop", fakeELF(183, true), 0}, validEntries()[1]}},
		{"no restart protocol", []archiveEntry{{"freeturn-desktop", fakeELF(62, false), 0}, validEntries()[1]}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stage := t.TempDir()
			archive := filepath.Join(stage, "archive.tar.gz")
			if err := os.WriteFile(archive, archiveFixture(t, test.entries), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := stageArchive(context.Background(), archive, stage, "v3.0.0", "amd64"); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}

func TestInstallAndRestoreBothBinaries(t *testing.T) {
	stage, target := prepareFixture(t)
	for _, name := range binaryNames {
		if err := os.WriteFile(filepath.Join(target, name), []byte("old:"+name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := InstallPrepared(stage, target, "amd64"); err != nil {
		t.Fatal(err)
	}
	for _, name := range binaryNames {
		backup, err := os.ReadFile(filepath.Join(stage, "previous", name))
		if err != nil || string(backup) != "old:"+name {
			t.Fatalf("backup failed for %s: %v", name, err)
		}
		info, err := os.Stat(filepath.Join(target, name))
		if err != nil || info.Mode().Perm() != 0o755 {
			t.Fatalf("wrong permissions: %v", err)
		}
	}
	if err := RestorePrevious(stage, target); err != nil {
		t.Fatal(err)
	}
	for _, name := range binaryNames {
		contents, _ := os.ReadFile(filepath.Join(target, name))
		if string(contents) != "old:"+name {
			t.Fatalf("rollback failed for %s", name)
		}
	}
}

func TestFailedSecondReplacementRollsBackPair(t *testing.T) {
	stage, target := prepareFixture(t)
	for _, name := range binaryNames {
		_ = os.WriteFile(filepath.Join(target, name), []byte("old:"+name), 0o755)
	}
	calls := 0
	err := installPrepared(stage, target, "amd64", func(source, destination string) error {
		calls++
		if calls == 2 {
			return errors.New("simulated disk failure")
		}
		return os.Rename(source, destination)
	})
	if err == nil || calls != 2 {
		t.Fatal("failure not tested")
	}
	for _, name := range binaryNames {
		contents, _ := os.ReadFile(filepath.Join(target, name))
		if string(contents) != "old:"+name {
			t.Fatalf("incomplete rollback for %s", name)
		}
	}
}

func TestNewInstallationRollbackRemovesNewPair(t *testing.T) {
	stage, target := prepareFixture(t)
	if err := InstallPrepared(stage, target, "amd64"); err != nil {
		t.Fatal(err)
	}
	if err := RestorePrevious(stage, target); err != nil {
		t.Fatal(err)
	}
	for _, name := range binaryNames {
		if _, err := os.Stat(filepath.Join(target, name)); !os.IsNotExist(err) {
			t.Fatalf("new binary remained: %s", name)
		}
	}
}

func TestModifiedStagingAndTargetSymlinkAreRejected(t *testing.T) {
	t.Run("staged tampering", func(t *testing.T) {
		stage, target := prepareFixture(t)
		_ = os.WriteFile(filepath.Join(stage, "new", binaryNames[0]), fakeELF(183, true), 0o700)
		if err := InstallPrepared(stage, target, "amd64"); err == nil {
			t.Fatal("installed modified binary")
		}
	})
	t.Run("symlink target", func(t *testing.T) {
		stage, target := prepareFixture(t)
		outside := filepath.Join(t.TempDir(), "outside")
		_ = os.WriteFile(outside, []byte("untouched"), 0o600)
		if err := os.Symlink(outside, filepath.Join(target, binaryNames[1])); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(target, binaryNames[0]), []byte("old"), 0o700)
		if err := InstallPrepared(stage, target, "amd64"); err == nil {
			t.Fatal("installed over symlink")
		}
		data, _ := os.ReadFile(outside)
		if string(data) != "untouched" {
			t.Fatal("symlink target changed")
		}
		data, _ = os.ReadFile(filepath.Join(target, binaryNames[0]))
		if string(data) != "old" {
			t.Fatal("old app changed on preflight failure")
		}
	})
}

func TestPrepareInstallDownloadsVerifiedArchiveAndCleansFailure(t *testing.T) {
	for _, broken := range []bool{false, true} {
		m := New("v2.0.0", "amd64", "example/fork")
		entries := validEntries()
		if broken {
			entries = entries[:1]
		}
		payload := archiveFixture(t, entries)
		name := "FreeTurn-Linux-amd64-v3.0.0.tar.gz"
		m.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			var data []byte
			if req.URL.String() == m.apiURL {
				data, _ = json.Marshal(release{Tag: "v3.0.0", Assets: []asset{
					{Name: name, Size: int64(len(payload)), URL: "https://github.com/example/fork/releases/download/v3.0.0/" + name},
					{Name: "checksums.txt", URL: "https://github.com/example/fork/releases/download/v3.0.0/checksums.txt"},
				}})
			} else if strings.HasSuffix(req.URL.Path, "checksums.txt") {
				data = []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(payload), name))
			} else {
				data = payload
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data))}, nil
		})
		if err := m.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		target := t.TempDir()
		_ = os.WriteFile(filepath.Join(target, binaryNames[0]), []byte("running app"), 0o700)
		stage, err := m.PrepareInstall(context.Background(), target)
		if (err != nil) != broken {
			t.Fatalf("unexpected preparation result: %v", err)
		}
		old, _ := os.ReadFile(filepath.Join(target, binaryNames[0]))
		if string(old) != "running app" {
			t.Fatal("preparation replaced running app")
		}
		files, _ := os.ReadDir(target)
		if broken && len(files) != 1 {
			t.Fatal("failed stage not cleaned")
		}
		if !broken {
			if err := ValidatePrepared(stage, target, "amd64"); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(stage, "archive.tar.gz")); !os.IsNotExist(err) {
				t.Fatal("downloaded archive not cleaned")
			}
		}
		if m.State().Busy != "" || m.State().DownloadedPath != "" {
			t.Fatal("temporary state leaked")
		}
	}
}

func TestCancelledExtraction(t *testing.T) {
	stage := t.TempDir()
	archive := filepath.Join(stage, "archive.tar.gz")
	_ = os.WriteFile(archive, archiveFixture(t, validEntries()), 0o600)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := stageArchive(ctx, archive, stage, "v3.0.0", "amd64"); err == nil {
		t.Fatal("ignored cancellation")
	}
}
