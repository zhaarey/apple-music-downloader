package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"main/internal/platform"
)

// RunAppleSyncUnlock stops stuck Apple Devices sync agents on Windows (AMPDevicesAgent, etc.).
// restartService requests Apple Mobile Device Service restart (may need elevation).
func RunAppleSyncUnlock(restartService, elevated bool) (ok bool, logTail string, needElevated bool, err error) {
	if runtime.GOOS != "windows" {
		return false, "", false, fmt.Errorf("sync unlock is only supported on Windows")
	}
	logPath := platform.AppleSyncUnlockLogPath()
	_ = os.WriteFile(logPath, nil, 0644)

	script, err := findRepairScript("apple-sync-unlock-windows.ps1")
	if err != nil {
		ok, logTail, err = runAppleSyncUnlockNative(logPath, restartService)
		return ok, logTail, false, err
	}
	if _, err := os.Stat(script); err != nil {
		ok, logTail, err = runAppleSyncUnlockNative(logPath, restartService)
		return ok, logTail, false, err
	}

	args := []string{
		"-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", script,
		"-LogPath", logPath,
	}
	if restartService {
		args = append(args, "-RestartService")
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
		return true, tail, restartService, nil
	}
	return true, tail, false, nil
}

func runAppleSyncUnlockNative(logPath string, restartService bool) (bool, string, error) {
	var lines []string
	appendLog := func(msg string) {
		lines = append(lines, msg)
		_ = os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	}

	names := []string{"AMPDevicesAgent.exe", "AMPDeviceDiscoveryAgent.exe", "AppleMobileDeviceHelper.exe"}
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
		appendLog("No Apple sync agent processes were running (already idle).")
	} else {
		appendLog(fmt.Sprintf("Released %d process tree(s).", killed))
	}

	if !restartService {
		appendLog("Sync unlock finished.")
		return true, readLogTail(logPath), nil
	}

	svc := exec.Command("powershell", "-NoProfile", "-Command",
		`Restart-Service -Name 'Apple Mobile Device Service' -Force -ErrorAction Stop`)
	if out, err := svc.CombinedOutput(); err != nil {
		appendLog("Service restart failed: " + strings.TrimSpace(string(out)))
		return true, readLogTail(logPath), nil
	}
	appendLog("Apple Mobile Device Service restarted.")
	appendLog("Sync unlock finished.")
	return true, readLogTail(logPath), nil
}
