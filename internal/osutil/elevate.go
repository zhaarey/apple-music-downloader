package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	appconfig "main/internal/config"
	"main/internal/platform"
)

// RunElevatedCacheClear re-runs artwork cache deletion with admin rights (Windows UAC / macOS admin).
func RunElevatedCacheClear() (ok bool, message string, err error) {
	paths := platform.AppleMusicArtworkCachePaths()
	if len(paths) == 0 {
		return true, "No cache paths configured for this platform.", nil
	}
	logPath := platform.SyncRepairLogPath()

	switch runtime.GOOS {
	case "windows":
		absScript, err := findRepairScript("sync-repair-windows.ps1")
		if err != nil {
			return false, "", err
		}
		if _, err := os.Stat(absScript); err != nil {
			return false, "", fmt.Errorf("repair script not found: %w", err)
		}
		args := strings.Join([]string{
			"-NoProfile", "-ExecutionPolicy", "Bypass",
			"-File", absScript,
			"-LogPath", logPath,
		}, " ")
		for _, p := range paths {
			args += " -CachePath " + quotePS(p)
		}
		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("Start-Process powershell -Verb RunAs -Wait -ArgumentList '%s'", args))
		if err := cmd.Run(); err != nil {
			return false, readLogTail(logPath), err
		}
		return true, readLogTail(logPath), nil
	case "darwin":
		absScript, err := findRepairScript("sync-repair-macos.sh")
		if err != nil {
			return false, "", err
		}
		var args []string
		for _, p := range paths {
			args = append(args, p)
		}
		escaped := strings.ReplaceAll(absScript, `"`, `\"`)
		inner := fmt.Sprintf(`"%s" "%s" %s`, escaped, logPath, strings.Join(args, `" "`))
		cmd := exec.Command("osascript", "-e",
			fmt.Sprintf(`do shell script "%s" with administrator privileges`, inner))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, string(out), err
		}
		return true, readLogTail(logPath), nil
	default:
		return false, "", fmt.Errorf("elevated cache clear not supported on %s", runtime.GOOS)
	}
}

func readLogTail(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "Elevated repair completed (no log output)."
	}
	s := strings.TrimSpace(string(data))
	if len(s) > 2000 {
		return s[len(s)-2000:]
	}
	return s
}

func findRepairScript(name string) (string, error) {
	candidates := []string{
		filepath.Join(appconfig.InstallDir(), "scripts", name),
		filepath.Join(appconfig.InstallDir(), "..", "scripts", name),
		filepath.Join("scripts", name),
	}
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			}
		}
	}
	return "", fmt.Errorf("repair script %q not found", name)
}

func quotePS(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}
