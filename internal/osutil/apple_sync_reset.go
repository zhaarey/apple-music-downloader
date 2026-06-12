package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"main/internal/platform"
)

// RunAppleSyncReset stops Apple Music and sync agents, then restarts Apple Mobile Device Service.
// Does not delete artwork caches or library files.
func RunAppleSyncReset(elevated bool) (ok bool, logTail string, needElevated bool, err error) {
	if runtime.GOOS != "windows" {
		return false, "", false, fmt.Errorf("sync reset is only supported on Windows")
	}
	logPath := platform.AppleSyncResetLogPath()
	_ = os.WriteFile(logPath, nil, 0644)

	script, scriptErr := findRepairScript("apple-sync-reset-windows.ps1")
	if scriptErr != nil {
		ok, logTail, needElevated, err = runAppleSyncResetNative(logPath)
		return ok, logTail, needElevated, err
	}
	if _, err := os.Stat(script); err != nil {
		ok, logTail, needElevated, err = runAppleSyncResetNative(logPath)
		return ok, logTail, needElevated, err
	}

	args := []string{
		"-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", script,
		"-LogPath", logPath,
	}

	if elevated {
		argLine := strings.Join(args, " ")
		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("Start-Process powershell -Verb RunAs -Wait -ArgumentList '%s'", argLine))
		if err := cmd.Run(); err != nil {
			return false, readLogTail(logPath), false, err
		}
	} else {
		cmdArgs := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass"}, args...)
		cmd := exec.Command("powershell", cmdArgs...)
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
	if strings.Contains(strings.ToLower(tail), "service restart failed") ||
		strings.Contains(strings.ToLower(tail), "needs administrator") {
		return true, tail, true, nil
	}
	return true, tail, false, nil
}

func runAppleSyncResetNative(logPath string) (bool, string, bool, error) {
	var lines []string
	appendLog := func(msg string) {
		lines = append(lines, msg)
		_ = os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	}

	names := []string{
		"AppleMusic.exe", "iTunes.exe", "iTunesHelper.exe", "distnoted.exe",
		"AMPDevicesAgent.exe", "AMPDeviceDiscoveryAgent.exe", "AppleMobileDeviceHelper.exe",
	}
	killed := 0
	for _, name := range names {
		out, err := exec.Command("taskkill", "/F", "/IM", name, "/T").CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err == nil {
			killed++
			appendLog(fmt.Sprintf("taskkill %s: %s", name, text))
		} else if !strings.Contains(strings.ToLower(text), "not found") {
			appendLog(fmt.Sprintf("taskkill %s: %s", name, text))
		}
	}
	if killed == 0 {
		appendLog("No Apple sync processes were running (already idle).")
	} else {
		appendLog(fmt.Sprintf("Stopped %d process tree(s).", killed))
	}

	svc := exec.Command("powershell", "-NoProfile", "-Command",
		`Restart-Service -Name 'Apple Mobile Device Service' -Force -ErrorAction Stop`)
	if out, err := svc.CombinedOutput(); err != nil {
		appendLog("Service restart failed: " + strings.TrimSpace(string(out)))
		appendLog("Sync reset finished with warnings.")
		return true, readLogTail(logPath), true, nil
	}
	appendLog("Apple Mobile Device Service restarted.")
	appendLog("Sync reset finished.")
	return true, readLogTail(logPath), false, nil
}
