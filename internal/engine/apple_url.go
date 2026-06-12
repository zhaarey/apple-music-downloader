package engine

import (
	"net/url"
	"strings"
)

// catalogSongURL builds a canonical Apple Music song link for a catalog song ID.
func catalogSongURL(storefront, songID string) string {
	songID = strings.TrimSpace(songID)
	if songID == "" {
		return ""
	}
	sf := strings.TrimSpace(storefront)
	if sf == "" {
		sf = Config.Storefront
	}
	if sf == "" {
		sf = "us"
	}
	return "https://music.apple.com/" + sf + "/song/" + songID
}

// normalizeAppleCatalogURL converts album links with ?i=songId into canonical song URLs.
func normalizeAppleCatalogURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	songID := strings.TrimSpace(u.Query().Get("i"))
	if songID == "" || !strings.Contains(strings.ToLower(u.Path), "/album/") {
		return raw
	}
	storefront, _ := checkUrl(raw)
	if storefront == "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) > 0 && len(parts[0]) == 2 {
			storefront = parts[0]
		}
	}
	if canon := catalogSongURL(storefront, songID); canon != "" {
		return canon
	}
	return raw
}
