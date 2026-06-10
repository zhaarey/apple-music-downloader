package media

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var forbiddenNames = regexp.MustCompile(`[/\\<>:"|?*]`)

// SanitizePathPart makes a string safe for folder/file names.
func SanitizePathPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Unknown"
	}
	return forbiddenNames.ReplaceAllString(s, "_")
}

// TrackFilename builds a numbered track filename with extension.
func TrackFilename(trackNumber int, title, ext string) string {
	title = SanitizePathPart(title)
	if trackNumber > 0 {
		return fmt.Sprintf("%02d. %s%s", trackNumber, title, ext)
	}
	return title + ext
}

var numberedTrackFile = regexp.MustCompile(`^(\d{1,3})\.\s*(.+)$`)

// ParseTrackFilename extracts track number and title from names like "01. Song Title.m4a".
func ParseTrackFilename(path string) (trackNum int16, title string) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if m := numberedTrackFile.FindStringSubmatch(base); len(m) == 3 {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 && n <= 32767 {
			trackNum = int16(n)
		}
		title = strings.TrimSpace(m[2])
	}
	if title == "" {
		title = base
	}
	return trackNum, title
}
