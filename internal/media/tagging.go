package media

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/zhaarey/go-mp4tag"
)

// TrackTags holds Apple Music–friendly metadata for an M4A file.
type TrackTags struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Year        string
	TrackNumber int16
	TrackTotal  int16
	DiscNumber  int16
	DiscTotal   int16
	SortTags    bool
	CoverPath   string
	CoverURL    string
	CoverData   []byte
	CoverMIME   string
}

// WriteTrackTags applies metadata to an existing M4A file.
func WriteTrackTags(path string, tags TrackTags) error {
	title := strings.TrimSpace(tags.Title)
	artist := strings.TrimSpace(tags.Artist)
	album := strings.TrimSpace(tags.Album)
	if title == "" {
		title = "Unknown Title"
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = title
	}
	albumArtist := strings.TrimSpace(tags.AlbumArtist)
	if albumArtist == "" {
		albumArtist = artist
	}
	trackNum := tags.TrackNumber
	if trackNum <= 0 {
		trackNum = 1
	}
	discNum := tags.DiscNumber
	if discNum <= 0 {
		discNum = 1
	}
	trackTotal := tags.TrackTotal
	if trackTotal <= 0 {
		trackTotal = trackNum
	}
	discTotal := tags.DiscTotal
	if discTotal <= 0 {
		discTotal = 1
	}

	t := &mp4tag.MP4Tags{
		Title:       title,
		Artist:      artist,
		Album:       album,
		AlbumArtist: albumArtist,
		CustomGenre: strings.TrimSpace(tags.Genre),
		TrackNumber: trackNum,
		TrackTotal:  trackTotal,
		DiscNumber:  discNum,
		DiscTotal:   discTotal,
		Date:        strings.TrimSpace(tags.Year),
		Custom: map[string]string{
			"PERFORMER": artist,
		},
	}
	if tags.SortTags {
		t.TitleSort = title
		t.ArtistSort = artist
		t.AlbumSort = album
		t.AlbumArtistSort = albumArtist
	}
	coverData, coverMIME, err := resolveCover(tags)
	if err == nil && len(coverData) > 0 {
		format := mp4tag.ImageTypeJPEG
		if strings.Contains(coverMIME, "png") || strings.HasSuffix(strings.ToLower(tags.CoverURL), ".png") {
			format = mp4tag.ImageTypePNG
		}
		t.Pictures = []*mp4tag.MP4Picture{{Format: format, Data: coverData}}
	}

	mp4, err := mp4tag.Open(path)
	if err != nil {
		return err
	}
	defer mp4.Close()
	return mp4.Write(t, []string{})
}

func resolveCover(tags TrackTags) ([]byte, string, error) {
	if len(tags.CoverData) > 0 {
		return tags.CoverData, tags.CoverMIME, nil
	}
	if tags.CoverPath != "" {
		data, err := os.ReadFile(tags.CoverPath)
		if err != nil {
			return nil, "", err
		}
		return data, "image/jpeg", nil
	}
	if tags.CoverURL == "" {
		return nil, "", os.ErrNotExist
	}
	resp, err := http.Get(tags.CoverURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", os.ErrInvalid
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return nil, "", os.ErrInvalid
	}
	return data, resp.Header.Get("Content-Type"), nil
}
