package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

const AppName = "AuraAudioDownloader"

// LegacyAppName is the previous product folder name.
const LegacyAppName = "AppleMusicDownloader"

// GOOS returns the current operating system (windows, darwin, linux, …).
func GOOS() string {
	return runtime.GOOS
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// AppDataDir returns the per-user application data directory for the current OS.
func AppDataDir() string {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, AppName)
		}
		return filepath.Join(homeDir(), AppName)
	case "darwin":
		return filepath.Join(homeDir(), "Library", "Application Support", AppName)
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "aura-audio-downloader")
		}
		return filepath.Join(homeDir(), ".config", "aura-audio-downloader")
	}
}

// LegacyAppDataDir returns the old product data directory before the Aura rename.
func LegacyAppDataDir() string {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, LegacyAppName)
		}
		return filepath.Join(homeDir(), LegacyAppName)
	case "darwin":
		return filepath.Join(homeDir(), "Library", "Application Support", LegacyAppName)
	default:
		return filepath.Join(homeDir(), "."+LegacyAppName)
	}
}

// DotLegacyAppDataDir is the pre–Application Support macOS path (~/.AuraAudioDownloader).
func DotLegacyAppDataDir() string {
	return filepath.Join(homeDir(), "."+AppName)
}

// MigrateAppDataDir copies config from older locations into AppDataDir once.
func MigrateAppDataDir() error {
	target := AppDataDir()
	sources := []string{LegacyAppDataDir(), DotLegacyAppDataDir()}
	for _, src := range sources {
		if src == target {
			continue
		}
		if err := migrateDirContents(src, target); err != nil {
			return err
		}
	}
	return nil
}

func migrateDirContents(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	copyIfMissing := func(name string) {
		from := filepath.Join(src, name)
		to := filepath.Join(dst, name)
		if _, err := os.Stat(to); err == nil {
			return
		}
		data, err := os.ReadFile(from)
		if err != nil {
			return
		}
		_ = os.WriteFile(to, data, 0644)
	}
	copyIfMissing("config.yaml")
	copyIfMissing("wizard.json")
	oldLogs := filepath.Join(src, "logs")
	newLogs := filepath.Join(dst, "logs")
	if _, err := os.Stat(newLogs); err != nil {
		if entries, err := os.ReadDir(oldLogs); err == nil && len(entries) > 0 {
			_ = os.MkdirAll(newLogs, 0755)
			for _, ent := range entries {
				if ent.IsDir() {
					continue
				}
				data, err := os.ReadFile(filepath.Join(oldLogs, ent.Name()))
				if err != nil {
					continue
				}
				_ = os.WriteFile(filepath.Join(newLogs, ent.Name()), data, 0644)
			}
		}
	}
	return nil
}
