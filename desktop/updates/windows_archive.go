package updates

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func validatePE(filename, arch string) error {
	machine := map[string]uint16{"amd64": pe.IMAGE_FILE_MACHINE_AMD64, "arm64": pe.IMAGE_FILE_MACHINE_ARM64, "386": pe.IMAGE_FILE_MACHINE_I386}[arch]
	binary, err := pe.Open(filename)
	if err != nil {
		return errors.New("update is not a PE executable")
	}
	defer binary.Close()
	if machine == 0 || binary.Machine != machine || binary.OptionalHeader == nil || len(binary.Sections) == 0 || binary.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 || binary.Characteristics&pe.IMAGE_FILE_DLL != 0 {
		return errors.New("Windows update has the wrong architecture or is not an application")
	}
	return nil
}

func stageWindowsArchive(ctx context.Context, archive, stage, version, arch string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > 2048 {
		return errors.New("too many update archive entries")
	}
	if err := os.Mkdir(filepath.Join(stage, "new"), 0o700); err != nil {
		return err
	}
	plan := installPlan{Version: version, Hashes: map[string]string{}, Previous: map[string]bool{}}
	var total uint64
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		clean := path.Clean(entry.Name)
		if strings.ContainsAny(entry.Name, "\\:\x00") || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || entry.Mode()&os.ModeSymlink != 0 {
			return errors.New("unsafe Windows archive entry")
		}
		if entry.UncompressedSize64 > maxArchive {
			return errors.New("expanded Windows archive is too large")
		}
		total += entry.UncompressedSize64
		if total > 1<<30 {
			return errors.New("expanded Windows archive is too large")
		}
		if clean != "freeturn-desktop.exe" {
			continue
		}
		if plan.Hashes[clean] != "" || entry.UncompressedSize64 == 0 || !entry.Mode().IsRegular() {
			return errors.New("duplicate or invalid Windows executable")
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		target := filepath.Join(stage, "new", clean)
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
		if err != nil {
			input.Close()
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(io.MultiWriter(output, hash), contextReader{ctx, io.LimitReader(input, maxArchive+1)})
		err = errors.Join(copyErr, input.Close(), output.Sync(), output.Close())
		if err != nil {
			return err
		}
		if err := validatePE(target, arch); err != nil {
			return err
		}
		if !supportsInstallProtocol(target) {
			return errors.New("Windows update does not support safe restart")
		}
		plan.Hashes[clean] = hex.EncodeToString(hash.Sum(nil))
	}
	if plan.Hashes["freeturn-desktop.exe"] == "" {
		return errors.New("Windows archive has no freeturn-desktop.exe")
	}
	return writePlan(stage, plan)
}
