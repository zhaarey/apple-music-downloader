package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"main/internal/platform"
)

// RunAppleMusicDeepPurge runs the bundled Windows purge script (stops Apple Music, clears PC caches).
func RunAppleMusicDeepPurge(elevated bool) (ok bool, logTail string, needElevated bool, err error) {
	if runtime.GOOS != "windows" {
		return false, "", false, fmt.Errorf("deep purge is only supported on Windows")
	}
	logPath := platform.ApplePurgeLogPath()
	_ = os.WriteFile(logPath, nil, 0644)

	script, err := findRepairScript("apple-purge-windows.ps1")
	if err != nil {
		return false, "", false, err
	}
	if _, err := os.Stat(script); err != nil {
		return false, "", false, fmt.Errorf("purge script not found: %w", err)
	}

	if elevated {
		args := strings.Join([]string{
			"-NoProfile", "-ExecutionPolicy", "Bypass",
			"-File", script,
			"-LogPath", logPath,
		}, " ")
		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("Start-Process powershell -Verb RunAs -Wait -ArgumentList '%s'", args))
		if err := cmd.Run(); err != nil {
			return false, readLogTail(logPath), false, err
		}
	} else {
		cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
			"-File", script, "-LogPath", logPath)
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			tail := readLogTail(logPath)
			if len(strings.TrimSpace(tail)) == 0 {
				tail = strings.TrimSpace(string(out))
			}
			if exitErr, okErr := runErr.(*exec.ExitError); okErr && exitErr.ExitCode() == 2 {
				return true, tail, true, nil
			}
			return false, tail, false, runErr
		}
	}

	tail := readLogTail(logPath)
	if strings.Contains(strings.ToLower(tail), "service restart failed") {
		return true, tail, true, nil
	}
	return true, tail, false, nil
}
