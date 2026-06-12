package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client calls the Spotify Web API using client-credentials auth.
type Client struct {
	ClientID     string
	ClientSecret string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
	http        *http.Client
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		http:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) configured() bool {
	return c != nil && c.ClientID != "" && c.ClientSecret != ""
}

func (c *Client) token() (string, error) {
	if !c.configured() {
		return "", fmt.Errorf("spotify API credentials are not configured")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-30*time.Second)) {
		return c.accessToken, nil
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequest(http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.ClientID+":"+c.ClientSecret)))
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("spotify auth failed: %s (%s)", resp.Status, strings.TrimSpace(string(body)))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("spotify auth returned empty token")
	}
	c.accessToken = tok.AccessToken
	if tok.ExpiresIn <= 0 {
		tok.ExpiresIn = 3600
	}
	c.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

func (c *Client) get(path string) ([]byte, error) {
	tok, err := c.token()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		kind := "resource"
		if strings.Contains(path, "/playlists/") {
			kind = "playlist"
		} else if strings.Contains(path, "/albums/") {
			kind = "album"
		} else if strings.Contains(path, "/tracks/") {
			kind = "track"
		}
		return nil, friendlySpotifyAPIError(resp.StatusCode, strings.TrimSpace(string(body)), kind)
	}
	return body, nil
}

type apiArtist struct {
	Name string `json:"name"`
}

type apiTrack struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Artists []apiArtist `json:"artists"`
	Album struct {
		Name string `json:"name"`
	} `json:"album"`
	ExternalIDs struct {
		ISRC string `json:"isrc"`
	} `json:"external_ids"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

type apiPlaylist struct {
	Name string `json:"name"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

type apiPlaylistTracks struct {
	Items []struct {
		Track apiTrack `json:"track"`
	} `json:"items"`
	Next string `json:"next"`
}

type apiAlbumTracks struct {
	Items []apiTrack `json:"items"`
	Next  string     `json:"next"`
}

func trackFromAPI(t apiTrack) Track {
	artist := ""
	if len(t.Artists) > 0 {
		artist = t.Artists[0].Name
	}
	return Track{
		ID:     t.ID,
		Title:  t.Name,
		Artist: artist,
		Album:  t.Album.Name,
		ISRC:   strings.TrimSpace(t.ExternalIDs.ISRC),
		URL:    t.ExternalURLs.Spotify,
	}
}

// ResolveLink fetches tracks for a Spotify share link using the Web API.
func (c *Client) ResolveLink(kind LinkKind, id, rawURL string) (ResolveResult, error) {
	if !c.configured() {
		return ResolveResult{}, fmt.Errorf("spotify API credentials are not configured")
	}
	switch kind {
	case LinkTrack:
		return c.resolveTrack(id, rawURL)
	case LinkAlbum:
		return c.resolveAlbum(id, rawURL)
	case LinkPlaylist:
		return c.resolvePlaylist(id, rawURL)
	default:
		return ResolveResult{}, fmt.Errorf("unsupported spotify link type")
	}
}

func (c *Client) resolveTrack(id, rawURL string) (ResolveResult, error) {
	body, err := c.get("/tracks/" + url.PathEscape(id))
	if err != nil {
		return ResolveResult{}, err
	}
	var t apiTrack
	if err := json.Unmarshal(body, &t); err != nil {
		return ResolveResult{}, err
	}
	tr := trackFromAPI(t)
	if tr.URL == "" {
		tr.URL = rawURL
	}
	title := tr.Title
	if tr.Artist != "" {
		title = tr.Title + " · " + tr.Artist
	}
	return ResolveResult{
		Kind:   LinkTrack,
		Title:  title,
		URL:    rawURL,
		Tracks: []Track{tr},
	}, nil
}

func (c *Client) resolveAlbum(id, rawURL string) (ResolveResult, error) {
	body, err := c.get("/albums/" + url.PathEscape(id))
	if err != nil {
		return ResolveResult{}, err
	}
	var album struct {
		Name string `json:"name"`
		Artists []apiArtist `json:"artists"`
		ExternalURLs struct {
			Spotify string `json:"spotify"`
		} `json:"external_urls"`
	}
	if err := json.Unmarshal(body, &album); err != nil {
		return ResolveResult{}, err
	}
	tracks, err := c.fetchAlbumTracks(id)
	if err != nil {
		return ResolveResult{}, err
	}
	title := album.Name
	if len(album.Artists) > 0 {
		title = album.Name + " · " + album.Artists[0].Name
	}
	link := album.ExternalURLs.Spotify
	if link == "" {
		link = rawURL
	}
	return ResolveResult{
		Kind:   LinkAlbum,
		Title:  title,
		URL:    link,
		Tracks: tracks,
	}, nil
}

func (c *Client) fetchAlbumTracks(albumID string) ([]Track, error) {
	var out []Track
	next := "/albums/" + url.PathEscape(albumID) + "/tracks?limit=50"
	for next != "" {
		path := next
		if strings.HasPrefix(next, "http") {
			path = strings.TrimPrefix(next, "https://api.spotify.com/v1")
		}
		body, err := c.get(path)
		if err != nil {
			return nil, err
		}
		var page apiAlbumTracks
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			if strings.TrimSpace(item.Name) == "" {
				continue
			}
			out = append(out, trackFromAPI(item))
		}
		next = page.Next
	}
	return out, nil
}

func (c *Client) resolvePlaylist(id, rawURL string) (ResolveResult, error) {
	body, err := c.get("/playlists/" + url.PathEscape(id) + "?fields=name,external_urls.spotify")
	if err != nil {
		return ResolveResult{}, err
	}
	var pl apiPlaylist
	if err := json.Unmarshal(body, &pl); err != nil {
		return ResolveResult{}, err
	}
	tracks, err := c.fetchPlaylistTracks(id)
	if err != nil {
		return ResolveResult{}, err
	}
	link := pl.ExternalURLs.Spotify
	if link == "" {
		link = rawURL
	}
	return ResolveResult{
		Kind:   LinkPlaylist,
		Title:  pl.Name,
		URL:    link,
		Tracks: tracks,
	}, nil
}

func (c *Client) fetchPlaylistTracks(playlistID string) ([]Track, error) {
	var out []Track
	next := "/playlists/" + url.PathEscape(playlistID) + "/tracks?limit=100&fields=items(track(id,name,artists(name),album(name),external_ids.isrc,external_urls.spotify)),next"
	for next != "" {
		path := next
		if strings.HasPrefix(next, "http") {
			path = strings.TrimPrefix(next, "https://api.spotify.com/v1")
		}
		body, err := c.get(path)
		if err != nil {
			return nil, err
		}
		var page apiPlaylistTracks
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			if item.Track.ID == "" || strings.TrimSpace(item.Track.Name) == "" {
				continue
			}
			out = append(out, trackFromAPI(item.Track))
		}
		next = page.Next
	}
	return out, nil
}

// Resolve uses Spotify oEmbed for single track links (no API keys required).
func Resolve(raw string) (ResolveResult, error) {
	kind, _, ok := ParseLink(raw)
	if !ok {
		return ResolveResult{}, fmt.Errorf("not a spotify link")
	}
	raw = strings.TrimSpace(raw)
	if kind != LinkTrack {
		return ResolveResult{}, fmt.Errorf("spotify %ss are not supported — paste one track link at a time (open.spotify.com/track/…). For playlists, use an Apple Music playlist link instead", kind)
	}
	tr, err := ResolveViaOEmbed(raw)
	if err != nil {
		return ResolveResult{}, err
	}
	title := tr.Title
	if tr.Artist != "" {
		title = tr.Title + " · " + tr.Artist
	}
	return ResolveResult{
		Kind:   LinkTrack,
		Title:  title,
		URL:    raw,
		Tracks: []Track{tr},
	}, nil
}

func friendlySpotifyAPIError(status int, body, kind string) error {
	switch status {
	case http.StatusNotFound:
		if kind == "playlist" {
			return fmt.Errorf("couldn't read that playlist — on Spotify open ⋯ → Make public, then paste the link again")
		}
		return fmt.Errorf("spotify %s not found — check the link is correct and public", kind)
	case http.StatusUnauthorized:
		return fmt.Errorf("spotify API access was rejected")
	case http.StatusForbidden:
		if strings.Contains(strings.ToLower(body), "premium") {
			return fmt.Errorf("spotify developer access requires Premium on the app owner account — paste one open.spotify.com/track/… link instead")
		}
		return fmt.Errorf("spotify API access denied — paste one open.spotify.com/track/… link instead")
	case http.StatusTooManyRequests:
		return fmt.Errorf("spotify rate limit hit — wait a minute and try again")
	default:
		if body != "" {
			return fmt.Errorf("spotify API error (%d): %s", status, body)
		}
		return fmt.Errorf("spotify API error: %d", status)
	}
}
