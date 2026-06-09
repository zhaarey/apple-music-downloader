package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"main/internal/platform"
)

// GetAppleMusicCacheInfo returns known Apple Music artwork cache folders on this system.
func GetAppleMusicCacheInfo() AppleMusicCacheInfo {
	paths := platform.AppleMusicArtworkCachePaths()
	existing := make([]string, 0, len(paths))
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			existing = append(existing, p)
		}
	}
	return AppleMusicCacheInfo{
		Paths:    existing,
		Platform: platform.GOOS(),
		Note:     platform.AppleMusicCacheNote(len(existing)),
	}
}

func removeTree(path string) (bool, string) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, ""
		}
		return false, err.Error()
	}
	if !info.IsDir() {
		if err := os.Remove(path); err != nil {
			return false, err.Error()
		}
		return true, ""
	}
	if err := os.RemoveAll(path); err != nil {
		return false, err.Error()
	}
	return true, ""
}

// ClearAppleMusicArtworkCache removes cached Apple Music artwork.
func ClearAppleMusicArtworkCache() CacheClearResult {
	paths := platform.AppleMusicArtworkCachePaths()
	if len(paths) == 0 {
		return CacheClearResult{
			OK:       true,
			Message:  "No Apple Music artwork cache paths are configured for this platform.",
			Cleared:  []string{},
			Errors:   []string{},
			Platform: platform.GOOS(),
		}
	}
	cleared := []string{}
	errs := []string{}
	for _, p := range paths {
		ok, errMsg := removeTree(p)
		if ok {
			cleared = append(cleared, p)
		} else if errMsg != "" {
			errs = append(errs, fmt.Sprintf("%s: %s", p, errMsg))
		}
	}
	res := CacheClearResult{Cleared: cleared, Errors: errs, Platform: platform.GOOS()}
	switch {
	case len(cleared) == 0 && len(errs) == 0:
		res.OK = true
		res.Message = "No artwork cache folders were present (nothing to clear)."
	case len(errs) > 0:
		res.OK = len(cleared) > 0
		if platform.GOOS() == "darwin" {
			res.Message = fmt.Sprintf("Cleared %d folder(s) with %d error(s). Quit Music and re-import albums.", len(cleared), len(errs))
		} else {
			res.Message = fmt.Sprintf("Cleared %d folder(s) with %d error(s). Quit Apple Music and re-import albums.", len(cleared), len(errs))
		}
	default:
		res.OK = true
		if platform.GOOS() == "darwin" {
			res.Message = fmt.Sprintf("Cleared %d Music artwork cache folder(s). Quit Music, then re-import your albums.", len(cleared))
		} else {
			res.Message = fmt.Sprintf("Cleared %d Apple Music artwork cache folder(s). Quit Apple Music, then re-import your albums.", len(cleared))
		}
	}
	return res
}

func clearTempPrefix(dir, prefix string) (int, []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, []string{err.Error()}
	}
	removed := 0
	errs := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		path := filepath.Join(dir, name)
		if err := os.Remove(path); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		removed++
	}
	return removed, errs
}

// ClearAppTempCache removes temporary cover/tag files created by this app.
func ClearAppTempCache() CacheClearResult {
	cleared := []string{}
	errs := []string{}
	dirs := []string{os.TempDir()}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, home)
	}
	total := 0
	for _, dir := range dirs {
		for _, prefix := range []string{".amd-cover-", ".amd-tag-"} {
			n, e := clearTempPrefix(dir, prefix)
			total += n
			errs = append(errs, e...)
			if n > 0 {
				cleared = append(cleared, fmt.Sprintf("%s (%d %s* files)", dir, n, prefix))
			}
		}
	}
	res := CacheClearResult{Cleared: cleared, Errors: errs, OK: len(errs) == 0, Platform: platform.GOOS()}
	if total == 0 {
		res.Message = "No app temp files found."
	} else {
		res.Message = fmt.Sprintf("Removed %d temporary app file(s).", total)
	}
	if len(errs) > 0 {
		res.OK = total > 0
		res.Message = fmt.Sprintf("Removed %d file(s) with %d error(s).", total, len(errs))
	}
	return res
}

// ClearAllSyncCaches clears Apple Music artwork cache and app temp files.
func ClearAllSyncCaches() CacheClearResult {
	apple := ClearAppleMusicArtworkCache()
	app := ClearAppTempCache()
	cleared := append([]string{}, apple.Cleared...)
	cleared = append(cleared, app.Cleared...)
	errs := append([]string{}, apple.Errors...)
	errs = append(errs, app.Errors...)
	ok := apple.OK && app.OK
	msg := strings.TrimSpace(apple.Message + " " + app.Message)
	if msg == "" {
		msg = "Caches cleared."
	}
	return CacheClearResult{OK: ok, Message: msg, Cleared: cleared, Errors: errs, Platform: platform.GOOS()}
}
