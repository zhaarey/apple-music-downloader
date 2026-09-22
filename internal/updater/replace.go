package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ApplyBinaryUpdate atomically replaces the current running binary with newBinary.
func ApplyBinaryUpdate(newBinary []byte) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determine current executable path: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolve symlink of executable: %w", err)
	}

	dir := filepath.Dir(exePath)
	base := filepath.Base(exePath)
	tempPath := filepath.Join(dir, "."+base+".new")

	// Write new binary to temp file
	if err := os.WriteFile(tempPath, newBinary, 0755); err != nil {
		return fmt.Errorf("write temporary binary file %s: %w", tempPath, err)
	}

	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		_ = os.Remove(oldPath) // remove any previous .old file if exists

		// 1. Rename running executable to .old
		if err := os.Rename(exePath, oldPath); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("rename running binary to %s: %w", oldPath, err)
		}

		// 2. Move new binary into original executable path
		if err := os.Rename(tempPath, exePath); err != nil {
			// Attempt rollback
			_ = os.Rename(oldPath, exePath)
			_ = os.Remove(tempPath)
			return fmt.Errorf("move new binary to %s: %w (restored original binary)", exePath, err)
		}

		return nil
	}

	// Linux & macOS: ensure executable permissions and atomic rename
	if err := os.Chmod(tempPath, 0755); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("chmod new binary: %w", err)
	}

	if err := os.Rename(tempPath, exePath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("atomic rename %s to %s: %w", tempPath, exePath, err)
	}

	return nil
}
