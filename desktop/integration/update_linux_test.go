//go:build linux

package integration

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateResultRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := UpdateResult{Message: "Обновление установлено", BackupPath: "/home/demo/previous"}
	if err := saveUpdateResult(want); err != nil {
		t.Fatal(err)
	}
	if got := LoadUpdateResult(); got != want {
		t.Fatalf("result = %+v", got)
	}
	filename, _ := updateResultPath()
	info, err := os.Stat(filename)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("result permissions: %v", err)
	}
}

func TestNewClientMustAcknowledgeGUIStartup(t *testing.T) {
	for _, success := range []bool{true, false} {
		directory := t.TempDir()
		executable := filepath.Join(directory, "test app")
		script := "#!/bin/sh\nexit 1\n"
		if success {
			script = "#!/bin/sh\nprintf '\\001' >&3\n"
		}
		if err := os.WriteFile(executable, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
		err := restartUpdated(executable, directory)
		if (err == nil) != success {
			t.Fatalf("startup ack success=%v, error=%v", success, err)
		}
	}
}

func TestAcknowledgeOnlyAfterDOMReady(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	updateAck = write
	t.Cleanup(func() { write.Close(); updateAck = nil })
	AcknowledgeUpdate()
	marker := make([]byte, 1)
	if _, err := io.ReadFull(read, marker); err != nil || marker[0] != 1 {
		t.Fatalf("missing startup acknowledgement: %v", err)
	}
	if updateAck != nil {
		t.Fatal("ack pipe was not closed")
	}
	AcknowledgeUpdate() // Idempotent if the DOM-ready callback repeats.
}
