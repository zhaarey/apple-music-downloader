package platform

import (
	"os/exec"
	"runtime"
	"strings"
)

// IsAppleMusicRunning reports whether Apple Music / iTunes is running.
func IsAppleMusicRunning() bool {
	switch runtime.GOOS {
	case "windows":
		return processRunning("AppleMusic.exe") || processRunning("iTunes.exe")
	case "darwin":
		return processRunning("Music") || processRunning("iTunes")
	default:
		return false
	}
}

func processRunning(name string) bool {
	if name == "" {
		return false
	}
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+name, "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(strings.ToLower(string(out)), strings.ToLower(name))
	case "darwin":
		out, err := exec.Command("pgrep", "-x", name).Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	default:
		return false
	}
}
