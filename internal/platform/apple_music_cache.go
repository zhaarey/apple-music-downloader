package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// AppleMusicArtworkCachePaths returns known Apple Music artwork cache directories on this OS.
func AppleMusicArtworkCachePaths() []string {
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return nil
		}
		root := filepath.Join(localAppData, "Apple Computer")
		return []string{
			filepath.Join(root, "iTunes", "Artwork"),
			filepath.Join(root, "Media", "Artwork"),
			filepath.Join(root, "Apple Music", "Artwork"),
		}
	case "darwin":
		home := homeDir()
		paths := []string{
			filepath.Join(home, "Library", "Caches", "com.apple.Music"),
			filepath.Join(home, "Library", "Caches", "com.apple.iTunes"),
		}
		groupRoot := filepath.Join(home, "Library", "Group Containers")
		if entries, err := os.ReadDir(groupRoot); err == nil {
			for _, ent := range entries {
				if !ent.IsDir() {
					continue
				}
				name := ent.Name()
				if len(name) < 8 {
					continue
				}
				// Apple Music group containers often contain "apple.Music" or "AppleMusic".
				if strings.Contains(name, "apple.Music") || strings.Contains(name, "AppleMusic") || name == "group.com.apple.iTunes" {
					paths = append(paths, filepath.Join(groupRoot, name, "Library", "Caches"))
				}
			}
		}
		return paths
	default:
		return nil
	}
}

// AppleMusicCacheNote returns platform-specific guidance for sync troubleshooting.
func AppleMusicCacheNote(existingCount int) string {
	switch runtime.GOOS {
	case "darwin":
		if existingCount == 0 {
			return "No Apple Music artwork cache folders found yet — they appear after Music has cached art. Quit Music before clearing."
		}
		return "Clear artwork cache after re-tagging files, then quit Music, re-import albums, delete on iPhone, and sync via Finder."
	case "windows":
		if existingCount == 0 {
			return "No Apple Music artwork cache folders found yet — they appear after Apple Music has cached art."
		}
		return "Clear artwork cache after re-tagging files, then quit Apple Music, re-import albums, and sync again in Apple Devices."
	default:
		return "Apple Music artwork cache clearing is supported on Windows and macOS."
	}
}
