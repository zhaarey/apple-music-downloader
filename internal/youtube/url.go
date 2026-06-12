package youtube

import "strings"

// IsURL reports whether raw looks like a YouTube link.
func IsURL(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return false
	}
	return strings.Contains(raw, "youtube.com/") ||
		strings.Contains(raw, "youtu.be/") ||
		strings.Contains(raw, "youtube.com?") ||
		strings.HasPrefix(raw, "youtu.be")
}
