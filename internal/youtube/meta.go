package youtube

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	appconfig "main/internal/config"
	"main/internal/media"
	"main/utils/structs"
)

// MetaFromDownload builds tagging metadata for a YouTube export row.
func MetaFromDownload(meta DownloadMeta, cfg structs.ConfigSet) media.TrackTags {
	title := meta.Title
	artist := meta.Artist
	album := meta.Album
	if title == "" {
		title = "Unknown Title"
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = title
	}
	albumArtist := meta.AlbumArtist
	if albumArtist == "" {
		albumArtist = artist
	}
	trackNum := int16(meta.TrackNumber)
	if trackNum <= 0 {
		trackNum = int16(meta.Num)
	}
	discNum := int16(meta.DiscNumber)
	if discNum <= 0 {
		discNum = 1
	}
	trackTotal := int16(meta.TrackTotal)
	if trackTotal <= 0 {
		trackTotal = trackNum
	}
	return media.TrackTags{
		Title:       title,
		Artist:      artist,
		Album:       album,
		AlbumArtist: albumArtist,
		Genre:       meta.Genre,
		Year:        meta.Year,
		TrackNumber: trackNum,
		TrackTotal:  trackTotal,
		DiscNumber:  discNum,
		DiscTotal:   1,
		SortTags:    cfg.TagSortOrder,
		CoverURL:    meta.ArtURL,
	}
}

func DefaultYear() string {
	return strconv.Itoa(time.Now().Year())
}

func OutputPath(saveDir string, meta DownloadMeta, multiTrack, video bool) string {
	albumDir := media.SanitizePathPart(meta.Album)
	if albumDir == "" {
		albumDir = "YouTube Downloads"
	}
	dir := filepath.Join(saveDir, albumDir)
	name := audioFilename(meta, multiTrack)
	if video {
		name = stringsTrimSuffix(name, ".m4a") + " [video].mp4"
	}
	return filepath.Join(dir, name)
}

func audioFilename(meta DownloadMeta, multiTrack bool) string {
	title := meta.Title
	if title == "" {
		title = fmt.Sprintf("Track %d", meta.Num)
	}
	n := meta.TrackNumber
	if n <= 0 {
		n = meta.Num
	}
	if multiTrack || n > 0 {
		return media.TrackFilename(n, title, ".m4a")
	}
	return media.SanitizePathPart(title) + ".m4a"
}

func stringsTrimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// FinalizeVideo converts a downloaded MP4 to H.264 + AAC stereo and tags it for Apple Music import.
func FinalizeVideo(cfg structs.ConfigSet, srcPath string, meta DownloadMeta, multiTrack bool) (string, error) {
	dst := OutputPath(OutputDir(cfg), meta, multiTrack, true)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "", err
	}
	if err := media.ConvertVideoToAppleMP4(cfg.FFmpegPath, srcPath, dst); err != nil {
		return "", err
	}
	tags := MetaFromDownload(meta, cfg)
	if err := media.WriteTrackTags(dst, tags); err != nil {
		return dst, fmt.Errorf("tagging failed: %w", err)
	}
	return dst, nil
}

// FinalizeAudio converts and tags a downloaded file for Apple Music import.
func FinalizeAudio(cfg structs.ConfigSet, saveDir string, srcPath string, meta DownloadMeta, multiTrack bool) (string, error) {
	dst := OutputPath(saveDir, meta, multiTrack, false)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "", err
	}
	if err := media.ConvertToAppleAAC(cfg.FFmpegPath, srcPath, dst); err != nil {
		return "", err
	}
	tags := MetaFromDownload(meta, cfg)
	if err := media.WriteTrackTags(dst, tags); err != nil {
		return dst, fmt.Errorf("tagging failed: %w", err)
	}
	return dst, nil
}

func MetaByNum(metas []DownloadMeta) map[int]DownloadMeta {
	out := make(map[int]DownloadMeta, len(metas))
	for _, m := range metas {
		out[m.Num] = m
	}
	return out
}

func OutputDir(cfg structs.ConfigSet) string {
	dir := cfg.YouTubeSaveFolder
	if dir == "" {
		dir = cfg.AacSaveFolder
	}
	return dir
}

func FFmpegArgs(cfg structs.ConfigSet) []string {
	return []string{"--ffmpeg-location", appconfig.FFmpegLocation(cfg.FFmpegPath)}
}
