package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"debug/pe"
	"encoding/binary"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWindowsReleaseSelectionAndVerifiedDownload(t *testing.T) {
	m, _, checksum := fixture(t, "v2.0.0")
	m.platform = "windows"
	if err := m.Check(context.Background()); err == nil {
		t.Fatal("Windows updater accepted Linux release assets")
	}
	transport := m.client.Transport
	const linuxName = "FreeTurn-Linux-amd64-v2.1.0.tar.gz"
	const windowsName = "FreeTurn-Windows-amd64-v2.1.0.zip"
	*checksum = strings.ReplaceAll(*checksum, linuxName, windowsName)
	m.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		response, err := transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if req.URL.String() == m.apiURL {
			data, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				return nil, err
			}
			response.Body = io.NopCloser(strings.NewReader(strings.ReplaceAll(string(data), linuxName, windowsName)))
		}
		return response, nil
	})
	if err := m.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if m.State().AssetName != windowsName {
		t.Fatalf("asset: %+v", m.State())
	}
	if err := m.Download(context.Background(), filepath.Join(t.TempDir(), "update.zip"), "v2.1.0"); err != nil {
		t.Fatal(err)
	}
}

func fakePE(machine uint16, protocol bool) []byte {
	data := make([]byte, 1024)
	copy(data, "MZ")
	binary.LittleEndian.PutUint32(data[0x3c:], 128)
	copy(data[128:], "PE\x00\x00")
	var header bytes.Buffer
	_ = binary.Write(&header, binary.LittleEndian, pe.FileHeader{Machine: machine, NumberOfSections: 1, SizeOfOptionalHeader: 240, Characteristics: pe.IMAGE_FILE_EXECUTABLE_IMAGE})
	_ = binary.Write(&header, binary.LittleEndian, pe.OptionalHeader64{Magic: 0x20b, AddressOfEntryPoint: 4096, ImageBase: 0x140000000, SectionAlignment: 4096, FileAlignment: 512, SizeOfImage: 8192, SizeOfHeaders: 512, Subsystem: 2, NumberOfRvaAndSizes: 16})
	section := pe.SectionHeader32{VirtualSize: 512, VirtualAddress: 4096, SizeOfRawData: 512, PointerToRawData: 512, Characteristics: 0x60000020}
	copy(section.Name[:], ".text")
	_ = binary.Write(&header, binary.LittleEndian, section)
	copy(data[132:], header.Bytes())
	if protocol {
		copy(data[512:], InstallProtocolMarker)
	}
	return data
}

func windowsZIP(t *testing.T, stage string, names []string, payload []byte, mode os.FileMode) string {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for _, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(mode)
		entry, err := writer.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(stage, "update.zip")
	if err := os.WriteFile(filename, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return filename
}

func TestWindowsZIPStagingAndRollback(t *testing.T) {
	target := t.TempDir()
	stage, err := os.MkdirTemp(target, ".freeturn-update-")
	if err != nil {
		t.Fatal(err)
	}
	archive := windowsZIP(t, stage, []string{"freeturn-desktop.exe"}, fakePE(pe.IMAGE_FILE_MACHINE_AMD64, true), 0o755)
	if err := stageWindowsArchive(context.Background(), archive, stage, "v2.0.0", "amd64"); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		return
	}
	old := filepath.Join(target, "freeturn-desktop.exe")
	if err := os.WriteFile(old, []byte("old app"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := InstallPrepared(stage, target, "amd64"); err != nil {
		t.Fatal(err)
	}
	if err := RestorePrevious(stage, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(old)
	if err != nil || string(got) != "old app" {
		t.Fatalf("rollback: %q, %v", got, err)
	}
}

func TestWindowsZIPRejectsUnsafeUpdates(t *testing.T) {
	for _, scenario := range []string{"traversal", "absolute", "backslash", "ads", "symlink", "duplicate", "architecture", "protocol", "cancelled", "not-pe"} {
		t.Run(scenario, func(t *testing.T) {
			stage := t.TempDir()
			names, payload, mode := []string{"freeturn-desktop.exe"}, fakePE(pe.IMAGE_FILE_MACHINE_AMD64, true), os.FileMode(0o755)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch scenario {
			case "traversal":
				names = []string{"../freeturn-desktop.exe"}
			case "absolute":
				names = []string{"/freeturn-desktop.exe"}
			case "backslash":
				names = []string{"a\\freeturn-desktop.exe"}
			case "ads":
				names = []string{"freeturn-desktop.exe:stream"}
			case "symlink":
				mode |= os.ModeSymlink
			case "duplicate":
				names = append(names, names[0])
			case "architecture":
				payload = fakePE(pe.IMAGE_FILE_MACHINE_ARM64, true)
			case "protocol":
				payload = fakePE(pe.IMAGE_FILE_MACHINE_AMD64, false)
			case "cancelled":
				cancel()
			case "not-pe":
				payload = []byte("not an executable")
			}
			archive := windowsZIP(t, stage, names, payload, mode)
			if err := stageWindowsArchive(ctx, archive, stage, "v2.0.0", "amd64"); err == nil {
				t.Fatal("unsafe update accepted")
			}
		})
	}
}
