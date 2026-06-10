package spotify

import (
	"net/url"
	"regexp"
	"strings"
)

// LinkKind is the Spotify resource type encoded in a share URL.
type LinkKind string

const (
	LinkTrack    LinkKind = "track"
	LinkAlbum    LinkKind = "album"
	LinkPlaylist LinkKind = "playlist"
)

var spotifyURLRe = regexp.MustCompile(`(?i)(?:https?://)?(?:open\.)?spotify\.com/(track|album|playlist)/([a-zA-Z0-9]+)`)

// ParseLink extracts the resource kind and ID from a Spotify share URL or URI.
func ParseLink(raw string) (LinkKind, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	if m := spotifyURLRe.FindStringSubmatch(raw); len(m) == 3 {
		return LinkKind(strings.ToLower(m[1])), m[2], true
	}
	if strings.HasPrefix(raw, "spotify:") {
		parts := strings.Split(raw, ":")
		if len(parts) >= 3 {
			switch parts[1] {
			case "track", "album", "playlist":
				return LinkKind(parts[1]), parts[2], true
			}
		}
	}
	if u, err := url.Parse(raw); err == nil && strings.Contains(strings.ToLower(u.Host), "spotify.com") {
		if m := spotifyURLRe.FindStringSubmatch(u.String()); len(m) == 3 {
			return LinkKind(strings.ToLower(m[1])), m[2], true
		}
	}
	return "", "", false
}

// IsSpotifyURL reports whether raw looks like a Spotify link.
func IsSpotifyURL(raw string) bool {
	_, _, ok := ParseLink(raw)
	return ok
}
