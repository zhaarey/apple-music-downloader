package widevinerip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grafov/m3u8"

	"amdl/internal/download"
)

// DefaultKeyFormat selects the EXT-X-KEY that the m3u8 parser reports for
// the playlist (the last one declared), which is the Widevine key on Apple
// Music playlists.
const DefaultKeyFormat = ""

// MediaPlaylist is the DRM-relevant part of an encrypted HLS media playlist.
type MediaPlaylist struct {
	// KeyURIPrefix is the part of the EXT-X-KEY URI before the first comma,
	// e.g. "data:text/plain;base64".
	KeyURIPrefix string
	// KeyPayload is the part after the first comma: the base64 key ID for
	// Widevine, a PSSH or key ID for PlayReady.
	KeyPayload string
	// InitSegmentURL is the EXT-X-MAP URI. For songs it is the whole file.
	InitSegmentURL string
	// SegmentURLs are the media segment URIs in playback order.
	SegmentURLs []string
}

// KeyURI returns the full EXT-X-KEY URI, as license servers expect it.
func (p *MediaPlaylist) KeyURI() string {
	return p.KeyURIPrefix + "," + p.KeyPayload
}

// StreamURLs returns the init segment followed by every media segment.
func (p *MediaPlaylist) StreamURLs() []string {
	urls := make([]string, 0, 1+len(p.SegmentURLs))
	urls = append(urls, p.InitSegmentURL)
	return append(urls, p.SegmentURLs...)
}

// FetchMediaPlaylist downloads and parses the media playlist at playlistURL.
// keyFormat selects the EXT-X-KEY by KEYFORMAT; see DefaultKeyFormat.
func FetchMediaPlaylist(ctx context.Context, playlistURL string, keyFormat string) (*MediaPlaylist, error) {
	body, err := fetch(ctx, playlistURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch media playlist: %w", err)
	}
	return ParseMediaPlaylist(playlistURL, body, keyFormat)
}

// ParseMediaPlaylist parses an HLS media playlist. Relative URIs are
// resolved against playlistURL.
func ParseMediaPlaylist(playlistURL string, body []byte, keyFormat string) (*MediaPlaylist, error) {
	decoded, listType, err := m3u8.DecodeFrom(strings.NewReader(string(body)), true)
	if err != nil {
		return nil, fmt.Errorf("parse media playlist: %w", err)
	}
	if listType != m3u8.MEDIA {
		return nil, errors.New("not a media playlist")
	}
	mediaPlaylist := decoded.(*m3u8.MediaPlaylist)

	keyURI := ""
	if keyFormat != DefaultKeyFormat {
		keyURI = findKeyURIByFormat(string(body), keyFormat)
	} else if mediaPlaylist.Key != nil {
		keyURI = mediaPlaylist.Key.URI
	}
	if keyURI == "" {
		return nil, errors.New("no matching key information found")
	}
	keyURIPrefix, keyPayload, err := splitKeyURI(keyURI)
	if err != nil {
		return nil, err
	}

	if mediaPlaylist.Map == nil || mediaPlaylist.Map.URI == "" {
		return nil, errors.New("no initialization segment found")
	}
	initSegmentURL, err := resolveRelativeURL(playlistURL, mediaPlaylist.Map.URI)
	if err != nil {
		return nil, err
	}

	var segmentURLs []string
	for _, segment := range mediaPlaylist.Segments {
		if segment == nil {
			continue
		}
		segmentURL, err := resolveRelativeURL(playlistURL, segment.URI)
		if err != nil {
			return nil, err
		}
		segmentURLs = append(segmentURLs, segmentURL)
	}

	return &MediaPlaylist{
		KeyURIPrefix:   keyURIPrefix,
		KeyPayload:     keyPayload,
		InitSegmentURL: initSegmentURL,
		SegmentURLs:    segmentURLs,
	}, nil
}

// findKeyURIByFormat returns the URI of the first EXT-X-KEY whose KEYFORMAT
// equals keyFormat, or "" when none matches.
func findKeyURIByFormat(playlist string, keyFormat string) string {
	const keyTag = "#EXT-X-KEY:"
	for _, rawLine := range strings.Split(playlist, "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, keyTag) {
			continue
		}
		attrs := m3u8.DecodeAttributeList(strings.TrimPrefix(line, keyTag))
		if attrs["KEYFORMAT"] == keyFormat {
			return attrs["URI"]
		}
	}
	return ""
}

// splitKeyURI splits an EXT-X-KEY data URI into its prefix and payload.
func splitKeyURI(keyURI string) (prefix string, payload string, err error) {
	prefix, payload, ok := strings.Cut(keyURI, ",")
	if !ok || prefix == "" || payload == "" {
		return "", "", errors.New("invalid DRM key URI")
	}
	return prefix, payload, nil
}

// resolveRelativeURL resolves ref against the directory of baseURL. Absolute
// refs are returned unchanged.
func resolveRelativeURL(baseURL string, ref string) (string, error) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, nil
	}
	lastSlash := strings.LastIndex(baseURL, "/")
	if lastSlash == -1 {
		return "", fmt.Errorf("invalid playlist URL %q", baseURL)
	}
	return baseURL[:lastSlash+1] + ref, nil
}

// fetch GETs url with optional headers and returns the body of a 200 reply.
func fetch(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := download.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
