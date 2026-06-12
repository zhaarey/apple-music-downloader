package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveExistingPath cleans, absolutizes, and verifies a filesystem path exists.
func ResolveExistingPath(path string) (string, os.FileInfo, error) {
	abs, err := normalizePath(path)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return abs, nil, fmt.Errorf("path not found: %w", err)
	}
	return abs, info, nil
}

func normalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"`)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "/", `\`)
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return abs, nil
}

// OpenInFileManager opens a folder in the system file manager.
// If path is a file, its parent directory is opened.
func OpenInFileManager(path string) error {
	abs, info, err := ResolveExistingPath(path)
	if err != nil {
		// Folder may not exist yet — try opening the normalized path anyway on Windows/macOS.
		if abs2, normErr := normalizePath(path); normErr == nil {
			if st, statErr := os.Stat(abs2); statErr == nil {
				abs, info = abs2, st
				err = nil
			} else if filepath.Ext(abs2) != "" {
				parent := filepath.Dir(abs2)
				if _, parentErr := os.Stat(parent); parentErr == nil {
					return openDirectory(parent)
				}
			}
		}
		if err != nil {
			return err
		}
	}

	dir := abs
	if !info.IsDir() {
		dir = filepath.Dir(abs)
	}
	return openDirectory(dir)
}

// RevealInFileManager opens the system file manager and selects path when supported.
func RevealInFileManager(path string) error {
	abs, info, err := ResolveExistingPath(path)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		if info.IsDir() {
			return openDirectory(abs)
		}
		// Separate arguments — required for paths with spaces.
		return exec.Command("explorer.exe", "/select,", abs).Start()
	case "darwin":
		return exec.Command("open", "-R", abs).Start()
	default:
		if info.IsDir() {
			return exec.Command("xdg-open", abs).Start()
		}
		return exec.Command("xdg-open", filepath.Dir(abs)).Start()
	}
}

func openDirectory(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("empty directory")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", dir).Start()
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}
