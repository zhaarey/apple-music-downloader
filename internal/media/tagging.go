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
	MediaKind      string `json:"media_kind,omitempty"` // audio | video
	VideoCodec     string `json:"video_codec,omitempty"`
	AudioCodec     string `json:"audio_codec,omitempty"`
	VideoWidth     int    `json:"video_width,omitempty"`
	VideoHeight    int    `json:"video_height,omitempty"`
	DurationLabel  string `json:"duration_label,omitempty"`
	AppleVideoReady bool  `json:"apple_video_ready,omitempty"`
	AppleVideoDetail string `json:"apple_video_detail,omitempty"`
	TagsPartial     bool   `json:"tags_partial,omitempty"`
	TagsWarning     string `json:"tags_warning,omitempty"`
}

// ReadAudioTags reads Apple Music–friendly tags from an M4A file.
// Damaged/malformed iTunes atoms fall back to ffprobe + filename instead of failing hard.
func ReadAudioTags(path string) (AudioTagInfo, error) {
	return ReadAudioTagsWithProbe("", path)
}

// ReadAudioTagsWithProbe is like ReadAudioTags but uses ffmpegConfigured for ffprobe fallback.
func ReadAudioTagsWithProbe(ffmpegConfigured, path string) (AudioTagInfo, error) {
	out := AudioTagInfo{Path: path}
	tags, err := readMP4TagsSafe(path)
	if err != nil {
		if isRecoverableMP4TagReadErr(err) {
			return readAudioTagsFallback(ffmpegConfigured, path, err), nil
		}
		return out, err
	}
	if tags == nil {
		return readAudioTagsFallback(ffmpegConfigured, path, nil), nil
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

	if len(tags.Pictures) > 0 {
		idx := 0
		if len(tags.Pictures) > 1 {
			idx = len(tags.Pictures) - 1
		}
		pic := tags.Pictures[idx]
		// Skip absurdly large covers that would freeze the UI / OOM the webview.
		const maxArtPreview = 12 << 20 // 12 MiB
		if pic != nil && len(pic.Data) > 0 && len(pic.Data) <= maxArtPreview {
			out.ArtworkCount = len(tags.Pictures)
			out.HasArtwork = true
			out.ArtworkMime = DetectImageMIME(pic.Data, pic.Format)
			out.ArtworkB64 = base64.StdEncoding.EncodeToString(pic.Data)
		} else if pic != nil && len(pic.Data) > maxArtPreview {
			out.ArtworkCount = len(tags.Pictures)
			out.HasArtwork = true
			out.TagsWarning = "Embedded artwork is very large — preview skipped. Re-save with optimized JPEG for Apple Music."
		}
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
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4":
		out.MediaKind = "video"
	default:
		out.MediaKind = "audio"
	}
	return out, nil
}

// EnrichVideoTagInfo attaches ffprobe stream details for MP4 music videos.
func EnrichVideoTagInfo(ffmpegConfigured, path string, info AudioTagInfo) AudioTagInfo {
	if !strings.EqualFold(filepath.Ext(path), ".mp4") {
		return info
	}
	info.MediaKind = "video"
	probe, err := ProbeVideoFile(ffmpegConfigured, path)
	if err != nil {
		if info.Summary != "" {
			info.Summary += " · video probe failed"
		}
		return info
	}
	info.VideoCodec = probe.VideoCodec
	info.AudioCodec = probe.AudioCodec
	info.VideoWidth = probe.Width
	info.VideoHeight = probe.Height
	info.DurationLabel = probe.DurationLabel
	info.AppleVideoReady = probe.AppleReady
	info.AppleVideoDetail = probe.AppleDetail
	if info.DurationLabel != "" {
		info.Summary += " · " + info.DurationLabel
	}
	if info.VideoWidth > 0 && info.VideoHeight > 0 {
		info.Summary += fmt.Sprintf(" · %dx%d", info.VideoWidth, info.VideoHeight)
	}
	return info
}

// AudioTagInfoFromPath builds minimal tag info from the file path when mp4tag cannot read the file.
func AudioTagInfoFromPath(path string) AudioTagInfo {
	trackNum, title := ParseTrackFilename(path)
	out := AudioTagInfo{
		Path:        path,
		Title:       title,
		TrackNumber: trackNum,
		Summary:     title + " · tags could not be read from file",
	}
	return out
}

func pictureMIME(format mp4tag.ImageType) string {
	return DetectImageMIME(nil, format)
}

// DetectImageMIME sniffs image bytes so data URLs render with the correct colors in the UI.
func DetectImageMIME(data []byte, format mp4tag.ImageType) string {
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return "image/png"
	}
	if len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a") {
		return "image/gif"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
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
	OutputPath       string `json:"output_path,omitempty"`
}

func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

func resolveWriteTarget(src, output string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", fmt.Errorf("no file selected")
	}
	out := strings.TrimSpace(output)
	if out == "" {
		return src, nil
	}
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(out)) {
		return src, nil
	}
	return out, nil
}

func copyMediaFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// WriteAudioTags applies metadata to an existing M4A or MP4 file.
func WriteAudioTags(input WriteAudioTagsInput) error {
	target, err := resolveWriteTarget(input.Path, input.OutputPath)
	if err != nil {
		return err
	}
	optimize := boolOrDefault(input.OptimizeArtwork, false)
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
	isVideo := strings.EqualFold(filepath.Ext(input.Path), ".mp4")
	if input.ClearArtwork {
		tags.CoverPath = ""
		tags.CoverData = nil
		if err := writeTrackTagsClearArtAt(input.Path, target, tags, isVideo); err != nil {
			return err
		}
		return writeCoverSidecarIfNeeded(target, tags, false, writeSidecar)
	}
	newCover := tags.CoverPath != ""
	if err := writeTrackTagsAt(input.Path, target, tags, isVideo); err != nil {
		return err
	}
	return writeCoverSidecarIfNeeded(target, tags, newCover, writeSidecar)
}

func writeCoverSidecarIfNeeded(target string, tags TrackTags, newCover, writeSidecar bool) error {
	if !newCover || !writeSidecar {
		return nil
	}
	if cover, err := PrepareCoverBytes(tags); err == nil && len(cover) > 0 {
		_, _ = WriteCoverSidecarForDir(filepath.Dir(target), cover)
	}
	return nil
}

func writeTrackTagsAt(src, dst string, tags TrackTags, isVideo bool) error {
	if isVideo {
		return WriteVideoTrackTagsTo("", src, dst, tags)
	}
	if !strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		if err := copyMediaFile(src, dst); err != nil {
			return fmt.Errorf("copy for save as: %w", err)
		}
	}
	err := WriteTrackTags(dst, tags)
	if err == nil {
		return nil
	}
	if !isCorruptMP4TagErr(err) && !isMissingMetaBoxErr(err) && !strings.Contains(strings.ToLower(err.Error()), "box not present") {
		return err
	}
	// Damaged ilst/meta: remux then rewrite.
	return RepairAndWriteTrackTags("", dst, tags)
}

func writeTrackTagsClearArtAt(src, dst string, tags TrackTags, isVideo bool) error {
	if isVideo {
		return WriteVideoTrackTagsTo("", src, dst, tags)
	}
	if !strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		if err := copyMediaFile(src, dst); err != nil {
			return fmt.Errorf("copy for save as: %w", err)
		}
	}
	return writeTrackTagsClearArt(dst, tags)
}

func writeTrackTagsClearArt(path string, tags TrackTags) error {
	if strings.EqualFold(filepath.Ext(path), ".mp4") {
		return WriteVideoTrackTags("", path, tags)
	}
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
		if isRecoverableMP4TagReadErr(err) {
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
	_ = existing
	wantNew := tags.CoverPath != "" || len(tags.CoverData) > 0 || tags.CoverURL != ""
	if !wantNew {
		// Metadata-only save: leave embedded artwork bytes untouched (avoids re-reading corrupt covr atoms).
		return nil, nil
	}
	coverData, format, err := PrepareCoverForEmbed(tags)
	if err != nil || len(coverData) == 0 {
		if err == nil {
			err = os.ErrNotExist
		}
		return nil, fmt.Errorf("artwork: %w", err)
	}
	return []*mp4tag.MP4Picture{{Format: format, Data: coverData}}, nil
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
	pictures, err := resolveWritePictures(tags, nil)
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
