package engine

import (
	"fmt"
	"strings"
	"unicode"

	"main/utils/ampapi"

	spotify "main/internal/spotify"
)

// SpotifyMatchItem is one Spotify track matched (or not) on Apple Music.
type SpotifyMatchItem struct {
	SpotifyTitle  string      `json:"spotify_title"`
	SpotifyArtist string      `json:"spotify_artist"`
	SpotifyAlbum  string      `json:"spotify_album,omitempty"`
	SpotifyISRC   string      `json:"spotify_isrc,omitempty"`
	MatchStatus   string      `json:"match_status"` // matched, uncertain, not_found
	MatchMethod   string      `json:"match_method,omitempty"` // isrc, catalog, none
	Score         int         `json:"score"`
	AppleHit      *SearchHit  `json:"apple_hit,omitempty"`
	Alternatives  []SearchHit `json:"alternatives,omitempty"`
}

// SpotifyResolveResult is returned by ResolveSpotifyLink.
type SpotifyResolveResult struct {
	SourceKind  string             `json:"source_kind"`
	SourceTitle string             `json:"source_title"`
	SourceURL   string             `json:"source_url"`
	TrackCount  int                `json:"track_count"`
	Matched     int                `json:"matched"`
	ISRCMatched int                `json:"isrc_matched"`
	Uncertain   int                `json:"uncertain"`
	Missing     int                `json:"missing"`
	Items       []SpotifyMatchItem `json:"items"`
	Error       string             `json:"error,omitempty"`
}

func (e *Engine) ResolveSpotifyLink(raw string) SpotifyResolveResult {
	raw = strings.TrimSpace(raw)
	out := SpotifyResolveResult{}

	res, err := spotify.Resolve(raw)
	if err != nil {
		out.Error = err.Error()
		return out
	}

	token, err := e.getToken()
	if err != nil {
		out.Error = err.Error()
		return out
	}

	out.SourceKind = string(res.Kind)
	out.SourceTitle = res.Title
	out.SourceURL = res.URL
	out.TrackCount = len(res.Tracks)

	for _, tr := range res.Tracks {
		item, status := e.matchSpotifyTrack(tr, token)
		out.Items = append(out.Items, item)
		switch status {
		case "matched":
			out.Matched++
			if item.MatchMethod == "isrc" {
				out.ISRCMatched++
			}
		case "uncertain":
			out.Uncertain++
		default:
			out.Missing++
		}
	}

	if out.TrackCount == 0 {
		out.Error = "No tracks found in that Spotify link"
	}
	return out
}

func (e *Engine) matchSpotifyTrack(tr spotify.Track, token string) (SpotifyMatchItem, string) {
	item := SpotifyMatchItem{
		SpotifyTitle:  tr.Title,
		SpotifyArtist: tr.Artist,
		SpotifyAlbum:  tr.Album,
		SpotifyISRC:   normalizeISRC(tr.ISRC),
		MatchMethod:   "none",
	}

	lang := Config.Language
	if lang == "" {
		lang = "en-US"
	}

	if item.SpotifyISRC != "" {
		if hit, ok := e.matchByISRC(tr, item.SpotifyISRC, token, lang); ok {
			item.MatchMethod = "isrc"
			item.Score = 100
			item.AppleHit = &hit
			item.MatchStatus = "matched"
			return item, item.MatchStatus
		}
	}

	item.MatchMethod = "catalog"
	query := strings.TrimSpace(tr.Title + " " + tr.Artist)
	if query == "" {
		item.MatchStatus = "not_found"
		return item, item.MatchStatus
	}

	resp, err := ampapi.Search(Config.Storefront, query, "songs", lang, token, 8, 0)
	if err != nil || resp == nil || resp.Results.Songs == nil || len(resp.Results.Songs.Data) == 0 {
		item.MatchStatus = "not_found"
		return item, item.MatchStatus
	}

	bestScore := -1
	bestIdx := -1
	scores := make([]int, len(resp.Results.Songs.Data))
	for i, song := range resp.Results.Songs.Data {
		scores[i] = scoreAppleSongMatch(tr.Title, tr.Artist, song.Attributes.Name, song.Attributes.ArtistName)
		if scores[i] > bestScore {
			bestScore = scores[i]
			bestIdx = i
		}
	}
	if bestIdx < 0 || bestScore < 45 {
		item.MatchStatus = "not_found"
		return item, item.MatchStatus
	}

	item.Score = bestScore
	hit := songToSearchHit(resp.Results.Songs.Data[bestIdx])
	item.AppleHit = &hit
	for i, song := range resp.Results.Songs.Data {
		if i == bestIdx {
			continue
		}
		if scores[i] >= 45 {
			item.Alternatives = append(item.Alternatives, songToSearchHit(song))
		}
	}
	if bestScore >= 75 {
		item.MatchStatus = "matched"
	} else {
		item.MatchStatus = "uncertain"
	}
	return item, item.MatchStatus
}

func (e *Engine) matchByISRC(tr spotify.Track, isrc, token, lang string) (SearchHit, bool) {
	songs, err := ampapi.LookupSongsByISRC(Config.Storefront, isrc, lang, token)
	if err != nil || len(songs) == 0 {
		return SearchHit{}, false
	}
	if len(songs) == 1 {
		return songToSearchHit(songs[0]), true
	}
	bestScore := -1
	bestIdx := 0
	for i, song := range songs {
		score := scoreAppleSongMatch(tr.Title, tr.Artist, song.Attributes.Name, song.Attributes.ArtistName)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return songToSearchHit(songs[bestIdx]), true
}

func normalizeISRC(raw string) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func songToSearchHit(item ampapi.SongRespData) SearchHit {
	songURL := catalogSongURL(Config.Storefront, item.ID)
	if songURL == "" {
		songURL = normalizeAppleCatalogURL(item.Attributes.URL)
	}
	return SearchHit{
		Type:   "Song",
		Name:   item.Attributes.Name,
		Detail: fmt.Sprintf("%s — %s", item.Attributes.ArtistName, item.Attributes.AlbumName),
		URL:    songURL,
		ID:     item.ID,
		ArtURL: strings.Replace(item.Attributes.Artwork.URL, "{w}x{h}", "300x300", 1),
	}
}

func scoreAppleSongMatch(spTitle, spArtist, amTitle, amArtist string) int {
	score := 0
	if normMatch(spTitle, amTitle) {
		score += 55
	} else if normContains(amTitle, spTitle) || normContains(spTitle, amTitle) {
		score += 35
	}
	if normMatch(spArtist, amArtist) {
		score += 40
	} else if normContains(amArtist, spArtist) || normContains(spArtist, amArtist) {
		score += 20
	}
	return score
}

func normMatch(a, b string) bool {
	return normalizeMatchText(a) == normalizeMatchText(b)
}

func normContains(haystack, needle string) bool {
	h := normalizeMatchText(haystack)
	n := normalizeMatchText(needle)
	if n == "" {
		return false
	}
	return strings.Contains(h, n)
}

func normalizeMatchText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
