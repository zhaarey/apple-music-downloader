package widevinerip

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grafov/m3u8"

	"amdl/internal/download"
	"amdl/internal/widevine-rip/key"
)

const (
	// AppleLicenseURL is Apple's Widevine license endpoint for web playback.
	AppleLicenseURL     = "https://play.itunes.apple.com/WebObjects/MZPlay.woa/wa/acquireWebPlaybackLicense"
	appleWebPlaybackURL = "https://play.music.apple.com/WebObjects/MZPlay.woa/wa/webPlayback"

	appleOrigin     = "https://music.apple.com"
	appleReferer    = "https://music.apple.com/"
	appleStoreFront = "143441-1,25"
	browserUA       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	widevineKeySystem = "com.widevine.alpha"
	// songAssetFlavor is the web playback asset flavor of 256 kbps AAC.
	songAssetFlavor = "28:ctrp256"
	// stationVariantHint marks the preferred (256 kbps) station variant URI.
	stationVariantHint = "256"
)

// ErrUnavailable reports that Apple offers no Widevine stream for an asset.
var ErrUnavailable = errors.New("Unavailable")

// Credentials are the Apple Music tokens sent with web playback requests.
type Credentials struct {
	AuthToken      string // developer token, sent as "Bearer ..."
	MediaUserToken string // media-user-token cookie of the signed-in user
}

// mediaHeaders returns the browser-like headers Apple expects for media requests.
func (c Credentials) mediaHeaders() map[string]string {
	headers := map[string]string{
		"User-Agent":          browserUA,
		"Origin":              appleOrigin,
		"Referer":             appleReferer,
		"Accept":              "application/vnd.apple.mpegurl,application/x-mpegURL,text/plain;q=0.8,*/*;q=0.5",
		"X-Apple-Store-Front": appleStoreFront,
	}
	if c.MediaUserToken != "" {
		headers["x-apple-music-user-token"] = c.MediaUserToken
		headers["Media-User-Token"] = c.MediaUserToken
	}
	return headers
}

// apiHeaders returns the headers for authenticated Apple API POSTs.
func (c Credentials) apiHeaders() map[string]string {
	return map[string]string{
		"Content-Type":             "application/json",
		"Origin":                   appleOrigin,
		"Referer":                  appleReferer,
		"User-Agent":               browserUA,
		"Authorization":            "Bearer " + c.AuthToken,
		"x-apple-music-user-token": c.MediaUserToken,
	}
}

type webPlaybackResponse struct {
	SongList []struct {
		HLSKeyCertURL  string `json:"hls-key-cert-url"`
		HLSPlaylistURL string `json:"hls-playlist-url"`
		Assets         []struct {
			Flavor string `json:"flavor"`
			URL    string `json:"URL"`
		} `json:"assets"`
	} `json:"songList"`
	Status int `json:"status"`
}

type appleLicenseResponse struct {
	ErrorCode  int    `json:"errorCode"`
	License    string `json:"license"`
	RenewAfter int    `json:"renew-after"`
	Status     int    `json:"status"`
}

// FetchSongPlaylistURL asks Apple's web playback API for the 256 kbps AAC
// media playlist of a song. It returns ErrUnavailable when there is none.
func FetchSongPlaylistURL(ctx context.Context, adamID string, creds Credentials) (string, error) {
	var playback webPlaybackResponse
	if err := postJSON(ctx, appleWebPlaybackURL, creds.apiHeaders(), map[string]string{"salableAdamId": adamID}, &playback); err != nil {
		return "", fmt.Errorf("web playback: %w", err)
	}
	if len(playback.SongList) == 0 {
		return "", ErrUnavailable
	}
	for _, asset := range playback.SongList[0].Assets {
		if asset.Flavor == songAssetFlavor {
			return asset.URL, nil
		}
	}
	return "", ErrUnavailable
}

// ResolveStationVariantPlaylist resolves a station master playlist to the
// media playlist of its preferred (256 kbps) variant, falling back to the
// first variant. A URL that is already a media playlist is returned as is.
func ResolveStationVariantPlaylist(masterURL string, creds Credentials) (string, error) {
	body, err := fetch(context.Background(), masterURL, creds.mediaHeaders())
	if err != nil {
		return "", err
	}
	decoded, listType, err := m3u8.DecodeFrom(bytes.NewReader(body), true)
	if err != nil {
		return "", err
	}
	if listType != m3u8.MASTER {
		return masterURL, nil
	}
	variantURI := pickStationVariant(decoded.(*m3u8.MasterPlaylist).Variants)
	if variantURI == "" {
		return masterURL, nil
	}
	return resolveRelativeURL(masterURL, variantURI)
}

func pickStationVariant(variants []*m3u8.Variant) string {
	fallback := ""
	for _, variant := range variants {
		if variant == nil {
			continue
		}
		if strings.Contains(variant.URI, stationVariantHint) {
			return variant.URI
		}
		if fallback == "" {
			fallback = variant.URI
		}
	}
	return fallback
}

// AppleLicenseExchange returns a key.ExchangeFunc that posts challenges for
// adamID to an Apple Widevine license endpoint (AppleLicenseURL when
// licenseURL is empty).
func AppleLicenseExchange(creds Credentials, licenseURL string, adamID string, keyURI string) key.ExchangeFunc {
	if licenseURL == "" {
		licenseURL = AppleLicenseURL
	}
	return func(ctx context.Context, challenge []byte) ([]byte, error) {
		request := map[string]any{
			"challenge":      base64.StdEncoding.EncodeToString(challenge),
			"key-system":     widevineKeySystem,
			"uri":            keyURI,
			"adamId":         adamID,
			"isLibrary":      false,
			"user-initiated": true,
		}
		var response appleLicenseResponse
		if err := postJSON(ctx, licenseURL, creds.apiHeaders(), request, &response); err != nil {
			return nil, fmt.Errorf("license request: %w", err)
		}
		if response.ErrorCode != 0 || response.Status != 0 {
			return nil, fmt.Errorf("license rejected: errorCode=%d status=%d", response.ErrorCode, response.Status)
		}
		license, err := base64.StdEncoding.DecodeString(response.License)
		if err != nil {
			return nil, fmt.Errorf("decode license: %w", err)
		}
		return license, nil
	}
}

// postJSON POSTs payload as JSON and decodes the JSON reply into result.
func postJSON(ctx context.Context, url string, headers map[string]string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := download.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("parse response (%s): %w", resp.Status, err)
	}
	return nil
}
