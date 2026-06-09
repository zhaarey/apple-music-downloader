package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// RevealInFileManager opens the system file manager and highlights path when supported.
func RevealInFileManager(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("path not found: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		arg := "/select," + abs
		if strings.Contains(abs, " ") {
			arg = `/select,"` + abs + `"`
		}
		return exec.Command("explorer", arg).Start()
	case "darwin":
		return exec.Command("open", "-R", abs).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(abs)).Start()
	}
}
