package media

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhaarey/go-mp4tag"
)

// AudioTagInfo is metadata read from or written to an audio file (GUI/API).
type AudioTagInfo struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Genre       string `json:"genre"`
	Year        string `json:"year"`
	TrackNumber int16  `json:"track_number"`
	TrackTotal  int16  `json:"track_total"`
	DiscNumber  int16  `json:"disc_number"`
	DiscTotal   int16  `json:"disc_total"`
	HasArtwork   bool   `json:"has_artwork"`
	ArtworkCount int    `json:"artwork_count"`
	ArtworkMime  string `json:"artwork_mime,omitempty"`
	ArtworkB64  string `json:"artwork_b64,omitempty"`
	Summary     string `json:"summary"`
}

// ReadAudioTags reads Apple Music–friendly tags from an M4A file.
func ReadAudioTags(path string) (AudioTagInfo, error) {
	out := AudioTagInfo{Path: path}
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return out, err
	}
	defer mp4.Close()

	tags, err := mp4.Read()
	if err != nil {
		return out, err
	}
	if tags == nil {
		return out, nil
	}

	out.Title = strings.TrimSpace(tags.Title)
	out.Artist = strings.TrimSpace(tags.Artist)
	if out.Artist == "" {
		out.Artist = strings.TrimSpace(tags.Custom["PERFORMER"])
	}
	out.Album = strings.TrimSpace(tags.Album)
	out.AlbumArtist = strings.TrimSpace(tags.AlbumArtist)
	out.Genre = strings.TrimSpace(tags.CustomGenre)
	out.Year = strings.TrimSpace(tags.Date)
	out.TrackNumber = tags.TrackNumber
	out.TrackTotal = tags.TrackTotal
	out.DiscNumber = tags.DiscNumber
	out.DiscTotal = tags.DiscTotal

	if len(tags.Pictures) > 0 && len(tags.Pictures[0].Data) > 0 {
		out.ArtworkCount = len(tags.Pictures)
		idx := 0
		if len(tags.Pictures) > 1 {
			idx = len(tags.Pictures) - 1
		}
		out.HasArtwork = true
		out.ArtworkMime = pictureMIME(tags.Pictures[idx].Format)
		out.ArtworkB64 = base64.StdEncoding.EncodeToString(tags.Pictures[idx].Data)
	}

	title := out.Title
	if title == "" {
		title = "Unknown Title"
	}
	artist := out.Artist
	if artist == "" {
		artist = "Unknown Artist"
	}
	out.Summary = title + " · " + artist
	if out.Album != "" {
		out.Summary += " · " + out.Album
	}
	return out, nil
}

func pictureMIME(format mp4tag.ImageType) string {
	if format == mp4tag.ImageTypePNG {
		return "image/png"
	}
	return "image/jpeg"
}

// WriteAudioTagsInput is the payload from the tag editor UI.
type WriteAudioTagsInput struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Genre       string `json:"genre"`
	Year        string `json:"year"`
	TrackNumber int16  `json:"track_number"`
	TrackTotal  int16  `json:"track_total"`
	DiscNumber  int16  `json:"disc_number"`
	DiscTotal   int16  `json:"disc_total"`
	CoverPath        string `json:"cover_path"`
	ClearArtwork     bool   `json:"clear_artwork"`
	SortTags         bool   `json:"sort_tags"`
	OptimizeArtwork  *bool  `json:"optimize_artwork"`
	WriteCoverSidecar *bool `json:"write_cover_sidecar"`
	Mp4boxReembed    bool   `json:"mp4box_reembed"`
}

func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

// WriteAudioTags applies metadata to an existing M4A file.
func WriteAudioTags(input WriteAudioTagsInput) error {
	optimize := boolOrDefault(input.OptimizeArtwork, true)
	writeSidecar := boolOrDefault(input.WriteCoverSidecar, true)
	tags := TrackTags{
		Title:           input.Title,
		Artist:          input.Artist,
		Album:           input.Album,
		AlbumArtist:     input.AlbumArtist,
		Genre:           input.Genre,
		Year:            input.Year,
		TrackNumber:     input.TrackNumber,
		TrackTotal:      input.TrackTotal,
		DiscNumber:      input.DiscNumber,
		DiscTotal:       input.DiscTotal,
		CoverPath:       strings.TrimSpace(input.CoverPath),
		SortTags:        input.SortTags,
		CoverOptimize:   &optimize,
		Mp4boxReembed:   input.Mp4boxReembed,
		WriteCoverSidecar: writeSidecar,
	}
	if input.ClearArtwork {
		tags.CoverPath = ""
		tags.CoverData = nil
		return writeTrackTagsClearArt(input.Path, tags)
	}
	newCover := tags.CoverPath != ""
	if err := WriteTrackTags(input.Path, tags); err != nil {
		return err
	}
	if newCover && writeSidecar {
		if cover, err := PrepareCoverBytes(tags); err == nil && len(cover) > 0 {
			_, _ = WriteCoverSidecarForDir(filepath.Dir(input.Path), cover)
		}
	}
	return nil
}

func writeTrackTagsClearArt(path string, tags TrackTags) error {
	t, err := buildMP4Tags(tags)
	if err != nil {
		return err
	}
	t.Pictures = nil
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return err
	}
	defer mp4.Close()
	return mp4.Write(t, []string{"allpictures"})
}

func clonePicture(p *mp4tag.MP4Picture) *mp4tag.MP4Picture {
	if p == nil {
		return nil
	}
	data := append([]byte(nil), p.Data...)
	return &mp4tag.MP4Picture{Format: p.Format, Data: data}
}

func readExistingPictures(path string) ([]*mp4tag.MP4Picture, error) {
	existing, err := readMP4TagsSafe(path)
	if err != nil {
		if isMissingMetaBoxErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	out := make([]*mp4tag.MP4Picture, 0, len(existing.Pictures))
	for _, p := range existing.Pictures {
		if p != nil && len(p.Data) > 0 {
			out = append(out, clonePicture(p))
		}
	}
	return out, nil
}

func resolveWritePictures(tags TrackTags, existing []*mp4tag.MP4Picture) ([]*mp4tag.MP4Picture, error) {
	wantNew := tags.CoverPath != "" || len(tags.CoverData) > 0 || tags.CoverURL != ""
	if wantNew {
		coverData, err := PrepareCoverBytes(tags)
		if err != nil || len(coverData) == 0 {
			if err == nil {
				err = os.ErrNotExist
			}
			return nil, fmt.Errorf("artwork: %w", err)
		}
		return []*mp4tag.MP4Picture{{Format: mp4tag.ImageTypeJPEG, Data: coverData}}, nil
	}
	if len(existing) == 0 {
		return nil, nil
	}
	pic := existing[0]
	if len(existing) > 1 {
		pic = existing[len(existing)-1]
	}
	opts := DefaultCoverNormalizeOptions()
	if tags.CoverOptimize != nil && !*tags.CoverOptimize {
		opts = LegacyCoverNormalizeOptions()
	}
	if normalized, err := NormalizeCoverWithOptions(pic.Data, opts); err == nil && len(normalized) > 0 {
		return []*mp4tag.MP4Picture{{Format: mp4tag.ImageTypeJPEG, Data: normalized}}, nil
	}
	return []*mp4tag.MP4Picture{clonePicture(pic)}, nil
}

func buildMP4Tags(tags TrackTags) (*mp4tag.MP4Tags, error) {
	return BuildAppleMusicTags(tags)
}

func writeMP4Tags(path string, t *mp4tag.MP4Tags, pictures []*mp4tag.MP4Picture) (err error) {
	t.Pictures = pictures
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("mp4tag write failed: %v", r)
		}
	}()
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return err
	}
	defer mp4.Close()
	delStrings := []string{}
	if len(pictures) > 0 {
		delStrings = []string{"allpictures"}
	}
	return mp4.Write(t, delStrings)
}

// TrackTags holds Apple Music–friendly metadata for an M4A file.
type TrackTags struct {
	Title          string
	Artist         string
	Album          string
	AlbumArtist    string
	Genre          string
	Year           string
	TrackNumber    int16
	TrackTotal     int16
	DiscNumber     int16
	DiscTotal      int16
	SortTags       bool
	IsCompilation  bool
	Lyrics         string
	ContentRating  string
	Copyright      string
	Publisher      string
	Composer       string
	ItunesAlbumID  int32
	ItunesArtistID int32
	CustomMeta     map[string]string
	CoverPath      string
	CoverURL       string
	CoverData      []byte
	CoverMIME         string
	RequireCover      bool
	CoverOptimize     *bool
	Mp4boxReembed     bool
	WriteCoverSidecar bool
}

// WriteTrackTags applies metadata to an existing audio or video file.
func WriteTrackTags(path string, tags TrackTags) error {
	if strings.EqualFold(filepath.Ext(path), ".mp4") {
		return WriteVideoTrackTags("", path, tags)
	}
	existing, err := readExistingPictures(path)
	if err != nil {
		return err
	}
	pictures, err := resolveWritePictures(tags, existing)
	if err != nil {
		return err
	}
	if tags.RequireCover && len(pictures) == 0 {
		return fmt.Errorf("artwork required but no cover could be embedded")
	}
	t, err := buildMP4Tags(tags)
	if err != nil {
		return err
	}
	if err := writeMP4Tags(path, t, pictures); err != nil {
		return err
	}
	if tags.Mp4boxReembed && len(pictures) > 0 && len(pictures[0].Data) > 0 {
		if err := ReembedCoverMP4Box(path, pictures[0].Data); err != nil {
			return err
		}
	}
	return nil
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
		mime := "image/jpeg"
		ext := strings.ToLower(filepath.Ext(tags.CoverPath))
		if ext == ".png" {
			mime = "image/png"
		}
		return data, mime, nil
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
