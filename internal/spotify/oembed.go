package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var oembedTitleRe = regexp.MustCompile(`(?i)^(.+?)\s*(?:[-–—·]\s*|\s+by\s+)(.+)$`)

type oembedResp struct {
	Title string `json:"title"`
	Type  string `json:"type"`
}

// ResolveViaOEmbed fetches a single track title/artist without Spotify API credentials.
func ResolveViaOEmbed(raw string) (Track, error) {
	reqURL := "https://open.spotify.com/oembed?url=" + url.QueryEscape(strings.TrimSpace(raw))
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return Track{}, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Track{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Track{}, fmt.Errorf("spotify oembed: %s (%s)", resp.Status, strings.TrimSpace(string(body)))
	}
	var obj oembedResp
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return Track{}, err
	}
	title, artist := parseOEmbedTitle(obj.Title)
	if title == "" {
		return Track{}, fmt.Errorf("could not parse track info from Spotify preview")
	}
	return Track{Title: title, Artist: artist, URL: strings.TrimSpace(raw)}, nil
}

func parseOEmbedTitle(raw string) (title, artist string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if m := oembedTitleRe.FindStringSubmatch(raw); len(m) == 3 {
		title = strings.TrimSpace(m[1])
		artist = strings.TrimSpace(m[2])
		artist = strings.TrimPrefix(artist, "song by ")
		artist = strings.TrimPrefix(artist, "Song by ")
		return title, artist
	}
	return raw, ""
}
