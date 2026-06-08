package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"main/utils/ampapi"
)

type PreviewTrack struct {
	Num        int    `json:"num"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	Type       string `json:"type"`
	Duration   string `json:"duration"`
	DurationMs int    `json:"duration_ms"`
	Explicit   bool   `json:"explicit"`
	IsMV       bool   `json:"is_mv"`
	URL        string `json:"url,omitempty"`
}

type PreviewResult struct {
	URL             string         `json:"url"`
	Type            string         `json:"type"`
	Title           string         `json:"title"`
	Subtitle        string         `json:"subtitle"`
	ArtURL          string         `json:"art_url"`
	TrackCount      int            `json:"track_count"`
	TotalDuration   string         `json:"total_duration"`
	Tracks          []PreviewTrack `json:"tracks"`
	CanSelectTracks bool           `json:"can_select_tracks"`
	OutputFolder    string         `json:"output_folder"`
	Error           string         `json:"error,omitempty"`
}

func (e *Engine) PreviewURL(raw string) PreviewResult {
	raw = strings.TrimSpace(raw)
	if Config.YouTubeMode {
		return e.previewYouTube(raw)
	}
	out := PreviewResult{
		URL:  raw,
		Type: e.DetectURLType(raw),
	}
	if raw == "" {
		out.Error = "Enter an Apple Music URL"
		return out
	}
	if out.Type == "Unknown" {
		out.Error = "Unrecognized URL — use a song, album, playlist, artist, or music video link"
		return out
	}

	token, err := e.getToken()
	if err != nil {
		out.Error = err.Error()
		return out
	}

	lang := Config.Language
	if lang == "" {
		lang = "en-US"
	}

	switch out.Type {
	case "Song":
		return e.previewSong(raw, token, lang, out)
	case "Music Video":
		return e.previewMusicVideo(raw, token, lang, out)
	case "Album":
		return e.previewAlbum(raw, token, lang, out)
	case "Playlist":
		return e.previewPlaylist(raw, token, lang, out)
	case "Artist":
		return e.previewArtist(raw, token, lang, out)
	case "Station":
		out.Error = "Station downloads work via CLI; paste a playlist or album link here for preview"
		return out
	default:
		out.Error = "Preview not available for this URL type"
		return out
	}
}

func (e *Engine) previewSong(raw, token, lang string, out PreviewResult) PreviewResult {
	storefront, songID := checkUrlSong(raw)
	if songID == "" {
		out.Error = "Invalid song URL"
		return out
	}
	resp, err := ampapi.GetSongResp(storefront, songID, lang, token)
	if err != nil || len(resp.Data) == 0 {
		out.Error = "Could not load song metadata"
		return out
	}
	s := resp.Data[0]
	out.Title = s.Attributes.Name
	out.Subtitle = s.Attributes.ArtistName
	out.ArtURL = formatArtworkURL(s.Attributes.Artwork.URL)
	out.TrackCount = 1
	out.CanSelectTracks = false
	out.OutputFolder = outputFolderForQuality("aac")
	out.Tracks = []PreviewTrack{trackFromSongData(s, 1)}
	out.TotalDuration = formatDuration(s.Attributes.DurationInMillis)
	return out
}

func (e *Engine) previewMusicVideo(raw, token, lang string, out PreviewResult) PreviewResult {
	storefront, mvID := checkUrlMv(raw)
	if mvID == "" {
		out.Error = "Invalid music video URL"
		return out
	}
	resp, err := ampapi.GetSongResp(storefront, mvID, lang, token)
	if err != nil || len(resp.Data) == 0 {
		out.Error = "Could not load music video metadata"
		return out
	}
	s := resp.Data[0]
	out.Title = s.Attributes.Name
	out.Subtitle = s.Attributes.ArtistName
	out.ArtURL = formatArtworkURL(s.Attributes.Artwork.URL)
	out.TrackCount = 1
	out.CanSelectTracks = false
	out.OutputFolder = Config.MVSaveFolder
	t := trackFromSongData(s, 1)
	t.IsMV = true
	t.Type = "music-videos"
	out.Tracks = []PreviewTrack{t}
	out.TotalDuration = formatDuration(s.Attributes.DurationInMillis)
	return out
}

func (e *Engine) previewAlbum(raw, token, lang string, out PreviewResult) PreviewResult {
	storefront, albumID := checkUrl(raw)
	if albumID == "" {
		out.Error = "Invalid album URL"
		return out
	}
	resp, err := ampapi.GetAlbumResp(storefront, albumID, lang, token)
	if err != nil || len(resp.Data) == 0 {
		out.Error = "Could not load album metadata"
		return out
	}
	meta := resp.Data[0]
	out.Title = meta.Attributes.Name
	out.Subtitle = meta.Attributes.ArtistName
	out.ArtURL = formatArtworkURL(meta.Attributes.Artwork.URL)
	out.CanSelectTracks = true
	out.OutputFolder = outputFolderForQuality("aac")
	tracks, totalMs := tracksFromRelationship(meta.Relationships.Tracks.Data)
	out.Tracks = tracks
	out.TrackCount = len(tracks)
	out.TotalDuration = formatDuration(totalMs)
	return out
}

func (e *Engine) previewPlaylist(raw, token, lang string, out PreviewResult) PreviewResult {
	storefront, playlistID := checkUrlPlaylist(raw)
	if playlistID == "" {
		out.Error = "Invalid playlist URL"
		return out
	}
	resp, err := ampapi.GetPlaylistResp(storefront, playlistID, lang, token)
	if err != nil || len(resp.Data) == 0 {
		out.Error = "Could not load playlist metadata"
		return out
	}
	meta := resp.Data[0]
	out.Title = meta.Attributes.Name
	out.Subtitle = "Apple Music · Playlist"
	out.ArtURL = formatArtworkURL(meta.Attributes.Artwork.URL)
	out.CanSelectTracks = true
	out.OutputFolder = outputFolderForQuality("aac")
	tracks, totalMs := tracksFromRelationship(meta.Relationships.Tracks.Data)
	out.Tracks = tracks
	out.TrackCount = len(tracks)
	out.TotalDuration = formatDuration(totalMs)
	return out
}

func (e *Engine) previewArtist(raw, token, lang string, out PreviewResult) PreviewResult {
	storefront, artistID := checkUrlArtist(raw)
	if artistID == "" {
		out.Error = "Invalid artist URL"
		return out
	}
	name, _, err := getUrlArtistName(raw, token)
	if err != nil {
		out.Error = "Could not load artist metadata"
		return out
	}
	out.Title = name
	out.Subtitle = "Select albums and music videos to download"
	out.CanSelectTracks = true
	out.OutputFolder = outputFolderForQuality("aac")

	albums, err := fetchArtistCatalog(storefront, artistID, "albums", lang, token)
	if err != nil {
		out.Error = "Could not load artist albums"
		return out
	}
	mvs, _ := fetchArtistCatalog(storefront, artistID, "music-videos", lang, token)

	tracks := make([]PreviewTrack, 0, len(albums)+len(mvs))
	num := 1
	for _, a := range albums {
		tracks = append(tracks, PreviewTrack{
			Num:      num,
			ID:       a.ID,
			Name:     a.Name,
			Artist:   name,
			Type:     "album",
			Duration: a.ReleaseDate,
			URL:      a.URL,
		})
		num++
	}
	for _, mv := range mvs {
		tracks = append(tracks, PreviewTrack{
			Num:      num,
			ID:       mv.ID,
			Name:     mv.Name,
			Artist:   name,
			Type:     "music-video",
			IsMV:     true,
			Duration: mv.ReleaseDate,
			URL:      mv.URL,
		})
		num++
	}
	out.Tracks = tracks
	out.TrackCount = len(tracks)
	return out
}

type catalogEntry struct {
	ID          string
	Name        string
	ReleaseDate string
	URL         string
}

func fetchArtistCatalog(storefront, artistID, relationship, lang, token string) ([]catalogEntry, error) {
	offset := 0
	var entries []catalogEntry
	for {
		req, err := http.NewRequest("GET", fmt.Sprintf(
			"https://amp-api.music.apple.com/v1/catalog/%s/artists/%s/%s?limit=100&offset=%d&l=%s",
			storefront, artistID, relationship, offset, lang,
		), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		req.Header.Set("Origin", "https://music.apple.com")
		do, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if do.StatusCode != http.StatusOK {
			do.Body.Close()
			return nil, fmt.Errorf("artist catalog: %s", do.Status)
		}
		obj := new(catalogArtistResp)
		err = json.NewDecoder(do.Body).Decode(obj)
		do.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, item := range obj.Data {
			entries = append(entries, catalogEntry{
				ID:          item.ID,
				Name:        item.Attributes.Name,
				ReleaseDate: item.Attributes.ReleaseDate,
				URL:         item.Attributes.URL,
			})
		}
		if len(obj.Next) == 0 {
			break
		}
		offset += 100
	}
	sort.Slice(entries, func(i, j int) bool {
		di, _ := time.Parse("2006-01-02", entries[i].ReleaseDate)
		dj, _ := time.Parse("2006-01-02", entries[j].ReleaseDate)
		return di.Before(dj)
	})
	return entries, nil
}

type catalogArtistResp struct {
	Next string `json:"next"`
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Name        string `json:"name"`
			ReleaseDate string `json:"releaseDate"`
			URL         string `json:"url"`
		} `json:"attributes"`
	} `json:"data"`
}

func trackFromSongData(s ampapi.SongRespData, num int) PreviewTrack {
	return PreviewTrack{
		Num:        num,
		ID:         s.ID,
		Name:       s.Attributes.Name,
		Artist:     s.Attributes.ArtistName,
		Type:       s.Type,
		Duration:   formatDuration(s.Attributes.DurationInMillis),
		DurationMs: s.Attributes.DurationInMillis,
		Explicit:   s.Attributes.ContentRating == "explicit",
		IsMV:       s.Type == "music-videos",
	}
}

func tracksFromRelationship(data []ampapi.TrackRespData) ([]PreviewTrack, int) {
	tracks := make([]PreviewTrack, 0, len(data))
	totalMs := 0
	for i, t := range data {
		tracks = append(tracks, PreviewTrack{
			Num:        i + 1,
			ID:         t.ID,
			Name:       t.Attributes.Name,
			Artist:     t.Attributes.ArtistName,
			Type:       t.Type,
			Duration:   formatDuration(t.Attributes.DurationInMillis),
			DurationMs: t.Attributes.DurationInMillis,
			Explicit:   t.Attributes.ContentRating == "explicit",
			IsMV:       t.Type == "music-videos",
		})
		totalMs += t.Attributes.DurationInMillis
	}
	return tracks, totalMs
}

func outputFolderForQuality(quality string) string {
	switch quality {
	case "atmos":
		if Config.AtmosSaveFolder != "" {
			return Config.AtmosSaveFolder
		}
	case "alac":
		if Config.AlacSaveFolder != "" {
			return Config.AlacSaveFolder
		}
	default:
		if Config.AacSaveFolder != "" {
			return Config.AacSaveFolder
		}
	}
	return Config.AacSaveFolder
}
