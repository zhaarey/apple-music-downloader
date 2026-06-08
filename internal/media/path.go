package media

import (
	"fmt"
	"regexp"
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
