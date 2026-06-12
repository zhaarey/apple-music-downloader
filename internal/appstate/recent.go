package appstate

import (
	"encoding/json"
	"os"
	"path/filepath"

	appconfig "main/internal/config"
)

const maxRecentFiles = 15

type recentStore struct {
	Paths []string `json:"paths"`
}

func recentPath() string {
	return filepath.Join(appconfig.AppDataDir(), "recent-files.json")
}

// GetRecentFiles returns recently opened tag-editor paths (newest first).
func GetRecentFiles() []string {
	data, err := os.ReadFile(recentPath())
	if err != nil {
		return nil
	}
	var store recentStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil
	}
	return append([]string(nil), store.Paths...)
}

// RememberRecentFile adds a path to the recent list.
func RememberRecentFile(path string) {
	path = filepath.Clean(path)
	if path == "" {
		return
	}
	paths := GetRecentFiles()
	out := []string{path}
	for _, p := range paths {
		if filepath.Clean(p) == path {
			continue
		}
		out = append(out, p)
		if len(out) >= maxRecentFiles {
			break
		}
	}
	_ = appconfig.EnsureAppDataDir()
	data, _ := json.Marshal(recentStore{Paths: out})
	_ = os.WriteFile(recentPath(), data, 0644)
}

type setupState struct {
	Complete bool `json:"complete"`
}

func setupPath() string {
	return filepath.Join(appconfig.AppDataDir(), "setup-checklist.json")
}

// GetSetupComplete reports whether the first-run checklist was finished.
func GetSetupComplete() bool {
	data, err := os.ReadFile(setupPath())
	if err != nil {
		return false
	}
	var state setupState
	if err := json.Unmarshal(data, &state); err != nil {
		return false
	}
	return state.Complete
}

// SetSetupComplete marks the first-run checklist complete.
func SetSetupComplete(complete bool) error {
	_ = appconfig.EnsureAppDataDir()
	data, err := json.Marshal(setupState{Complete: complete})
	if err != nil {
		return err
	}
	return os.WriteFile(setupPath(), data, 0644)
}
