package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"amdl/internal/version"
)

type CheckResult struct {
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	ReleaseName    string    `json:"release_name"`
	ReleaseURL     string    `json:"release_url"`
	LastChecked    time.Time `json:"last_checked"`
	HasUpdate      bool      `json:"has_update"`
}

func getCacheFilePath() string {
	cacheDir, err := os.UserCacheDir()
	if err == nil && cacheDir != "" {
		amdlDir := filepath.Join(cacheDir, "amdl")
		_ = os.MkdirAll(amdlDir, 0755)
		return filepath.Join(amdlDir, "update_cache.json")
	}
	return ".amdl_update_cache.json"
}

func readCache() (*CheckResult, error) {
	path := getCacheFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var res CheckResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func saveCache(res *CheckResult) {
	path := getCacheFilePath()
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

// CheckUpdateCached checks if a newer version is available, using a local cache to throttle checks.
// It is meant for the startup notice, so the request uses a short timeout.
func CheckUpdateCached(currentVersion, proxyURL string, interval time.Duration) (*CheckResult, error) {
	cached, _ := readCache()
	if cached != nil && cached.CurrentVersion != currentVersion {
		// Result computed for another version (e.g. before an upgrade) says nothing about this one.
		cached = nil
	}
	if cached != nil && time.Since(cached.LastChecked) < interval {
		return cached, nil
	}

	// Fetch fresh release from GitHub
	client, err := NewHTTPClient(proxyURL, startupCheckTimeout)
	if err != nil {
		return nil, err
	}
	rel, err := fetchLatestRelease(client)
	if err != nil {
		// Record the failed attempt too, so an unreachable GitHub is not retried
		// (and does not delay startup) on every run within the interval.
		failed := &CheckResult{CurrentVersion: currentVersion}
		if cached != nil {
			failed = cached
		}
		failed.LastChecked = time.Now()
		saveCache(failed)
		if cached != nil {
			return cached, nil
		}
		return nil, err
	}

	hasUpdate := version.Compare(currentVersion, rel.TagName) < 0
	result := &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  rel.TagName,
		ReleaseName:    rel.Name,
		ReleaseURL:     rel.HTMLURL,
		LastChecked:    time.Now(),
		HasUpdate:      hasUpdate,
	}

	saveCache(result)
	return result, nil
}
