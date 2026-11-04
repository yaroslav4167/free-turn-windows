//go:build windows

package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/samosvalishe/free-turn-proxy/desktop/updates"
	"golang.org/x/sys/windows"
)

type UpdateResult struct {
	Message    string `json:"message"`
	BackupPath string `json:"backupPath"`
}

func updateResultPath() (string, error) {
	dir, err := os.UserConfigDir()
	return filepath.Join(dir, "FreeTurnDesktop", "update-result.json"), err
}

func LoadUpdateResult() UpdateResult {
	var result UpdateResult
	filename, err := updateResultPath()
	if err != nil {
		return result
	}
	data, err := os.ReadFile(filename)
	if err == nil && len(data) < 16384 {
		_ = json.Unmarshal(data, &result)
	}
	return result
}

func saveUpdateResult(result UpdateResult) {
	filename, err := updateResultPath()
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(filename), 0o700) != nil {
		return
	}
	data, _ := json.Marshal(result)
	_ = os.WriteFile(filename, data, 0o600)
}

// Windows locks running executables. Use a separate OLD updater copy and keep
// the old GUI alive until it signals readiness. Stdin EOF gates installation.
func StartUpdateWorker(stage string) (*os.File, error) {
	target, err := InstalledPath()
	if err != nil {
		return nil, err
	}
	if err := updates.ValidatePrepared(stage, filepath.Dir(target), runtime.GOARCH); err != nil {
		return nil, err
	}
	current, err := os.Executable()
	if err != nil {
		return nil, err
	}
	worker := filepath.Join(stage, "update-worker.exe")
	if err := copyExecutable(current, worker); err != nil {
		return nil, err
	}
	gateRead, gateWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer gateRead.Close()
	cmd := exec.Command(worker, "--apply-update", stage, fmt.Sprint(os.Getpid()), current)
	cmd.Stdin = gateRead
	ready, err := cmd.StdoutPipe()
	if err != nil {
		gateWrite.Close()
		return nil, err
	}
	defer ready.Close()
	log, err := os.OpenFile(filepath.Join(stage, "worker.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		gateWrite.Close()
		return nil, err
	}
	defer log.Close()
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		gateWrite.Close()
		return nil, err
	}
	result := make(chan error, 1)
	go func() {
		var marker [1]byte
		_, err := io.ReadFull(ready, marker[:])
		if err == nil && marker[0] != 1 {
			err = errors.New("invalid updater readiness marker")
		}
		result <- err
	}()
	select {
	case err = <-result:
	case <-time.After(15 * time.Second):
		err = errors.New("updater did not become ready")
	}
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		gateWrite.Close()
		return nil, err
	}
	_ = cmd.Process.Release()
	return gateWrite, nil
}

func HandleUpdateProcess() bool {
	if len(os.Args) == 2 && os.Args[1] == "--update-protocol" {
		fmt.Println(updates.InstallProtocolMarker)
		return true
	}
	if len(os.Args) == 5 && os.Args[1] == "--apply-update" {
		if err := runWindowsUpdate(os.Args[2], os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return true
	}
	return false
}

func AcknowledgeUpdate() {
	ack := os.Getenv("FREETURN_UPDATE_ACK_FILE")
	_ = os.Unsetenv("FREETURN_UPDATE_ACK_FILE")
	target, err := InstalledPath()
	stage := filepath.Dir(ack)
	if err != nil || !filepath.IsAbs(ack) || filepath.Base(ack) != "restart-ready" || filepath.Dir(stage) != filepath.Dir(target) || !strings.HasPrefix(filepath.Base(stage), ".freeturn-update-") {
		return
	}
	info, err := os.Lstat(stage)
	if err != nil || !info.IsDir() {
		return
	}
	file, err := os.OpenFile(ack, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		_, _ = file.Write([]byte{1})
		_ = file.Close()
	}
}

func runWindowsUpdate(stage, parentID, fallback string) error {
	if !filepath.IsAbs(fallback) {
		return errors.New("invalid fallback executable")
	}
	target, err := InstalledPath()
	if err != nil {
		return err
	}
	targetDir := filepath.Dir(target)
	if err := updates.ValidatePrepared(stage, targetDir, runtime.GOARCH); err != nil {
		return err
	}
	var pid uint32
	if _, err := fmt.Sscan(parentID, &pid); err != nil || pid == 0 {
		return errors.New("invalid update parent")
	}
	parent, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(parent)
	if _, err := os.Stdout.Write([]byte{1}); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		return err
	}
	state, err := windows.WaitForSingleObject(parent, 30000)
	if err != nil || state != windows.WAIT_OBJECT_0 {
		return errors.New("old GUI did not exit; installation unchanged")
	}
	result := UpdateResult{BackupPath: filepath.Join(stage, "previous")}
	if err := updates.InstallPrepared(stage, targetDir, runtime.GOARCH); err != nil {
		result.Message = "Обновление не установлено: " + err.Error()
		saveUpdateResult(result)
		return restartWindows(fallback)
	}
	ack := filepath.Join(stage, "restart-ready")
	cmd := exec.Command(target, "--updated")
	cmd.Env = append(os.Environ(), "FREETURN_UPDATE_ACK_FILE="+ack)
	if output, installErr := exec.Command(target, "--install-desktop").CombinedOutput(); installErr != nil {
		err = fmt.Errorf("refresh installation: %w: %s", installErr, output)
	} else {
		err = cmd.Start()
	}
	if err == nil {
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.NewTimer(45 * time.Second)
		defer timeout.Stop()
	wait:
		for {
			select {
			case <-done:
				err = errors.New("new GUI exited before confirming startup")
				break wait
			case <-timeout.C:
				_ = cmd.Process.Kill()
				<-done
				err = errors.New("new GUI did not confirm startup")
				break wait
			case <-ticker.C:
				if data, readErr := os.ReadFile(ack); readErr == nil && string(data) == "\x01" {
					result.Message = "Обновление установлено. Предыдущий клиент сохранён для отката."
					saveUpdateResult(result)
					return nil
				}
			}
		}
	}
	rollback := updates.RestorePrevious(stage, targetDir)
	result.Message = fmt.Sprintf("Новая версия не запустилась; откат: %v; %v", err, rollback)
	saveUpdateResult(result)
	if rollback != nil {
		return errors.New(result.Message)
	}
	return restartWindows(fallback)
}

func restartWindows(target string) error {
	cmd := exec.Command(target)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
