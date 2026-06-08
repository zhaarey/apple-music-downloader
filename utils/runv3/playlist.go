package runv3

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"main/internal/logging"

	"github.com/grafov/m3u8"
)

// FMP4Part is one byte range (or whole file) concatenated before Widevine decrypt.
type FMP4Part struct {
	URL    string
	Offset int64
	Limit  int64
}

func (p FMP4Part) fetch(client *http.Client) ([]byte, error) {
	return fetchBytes(client, p.URL, p.Offset, p.Limit)
}

func resolveMediaURL(base, uri string) string {
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return uri
	}
	return base + uri
}

func parsePlaybackPlaylist(playlistURL string) (WebPlayback, error) {
	body, err := fetchPart(playlistURL)
	if err != nil {
		return WebPlayback{}, wrapStage(StageDownload, fmt.Errorf("playlist fetch: %w", err))
	}
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(string(body)), true)
	if err != nil {
		return WebPlayback{}, wrapStage(StageDownload, err)
	}
	if listType != m3u8.MEDIA {
		return WebPlayback{}, wrapStage(StageDownload, errors.New("m3u8 is not a media playlist"))
	}
	mediaPlaylist := from.(*m3u8.MediaPlaylist)
	if mediaPlaylist.Key == nil {
		return WebPlayback{}, wrapStage(StageDownload, errors.New("no encryption key in playlist"))
	}
	split := strings.Split(mediaPlaylist.Key.URI, ",")
	if len(split) < 2 {
		return WebPlayback{}, wrapStage(StageDownload, errors.New("invalid KEY URI in playlist"))
	}
	lastSlash := strings.LastIndex(playlistURL, "/")
	if lastSlash < 0 {
		return WebPlayback{}, wrapStage(StageDownload, errors.New("invalid playlist URL"))
	}
	base := playlistURL[:lastSlash+1]

	var parts []FMP4Part
	byteRange := false
	if mediaPlaylist.Map != nil && mediaPlaylist.Map.URI != "" {
		part := FMP4Part{
			URL:    resolveMediaURL(base, mediaPlaylist.Map.URI),
			Offset: mediaPlaylist.Map.Offset,
			Limit:  mediaPlaylist.Map.Limit,
		}
		parts = append(parts, part)
		byteRange = byteRange || part.Limit > 0
	}
	for _, segment := range mediaPlaylist.Segments {
		if segment == nil || segment.URI == "" {
			continue
		}
		part := FMP4Part{
			URL:    resolveMediaURL(base, segment.URI),
			Offset: segment.Offset,
			Limit:  segment.Limit,
		}
		parts = append(parts, part)
		byteRange = byteRange || part.Limit > 0
	}
	if len(parts) == 0 {
		return WebPlayback{}, wrapStage(StageDownload, errors.New("no init or media segments in playlist"))
	}
	logging.Info("runv3 playlist parsed: %d parts, byte-range=%v", len(parts), byteRange)
	return WebPlayback{
		KidBase64: split[1],
		UriPrefix: split[0],
		Parts:     parts,
	}, nil
}

// EncodeParts serializes parts for the MV license hand-off (ExtMvData).
func EncodeParts(parts []FMP4Part) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString(";")
		}
		b.WriteString(p.URL)
		if p.Limit > 0 {
			b.WriteString(fmt.Sprintf("#%d:%d", p.Offset, p.Limit))
		}
	}
	return b.String()
}

func decodePartToken(raw string) FMP4Part {
	part := FMP4Part{URL: raw}
	idx := strings.LastIndex(raw, "#")
	if idx <= 0 || !strings.Contains(raw[idx:], ":") {
		return part
	}
	part.URL = raw[:idx]
	var off, lim int64
	if _, err := fmt.Sscanf(raw[idx+1:], "%d:%d", &off, &lim); err == nil && lim > 0 {
		part.Offset = off
		part.Limit = lim
	}
	return part
}
