package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

// WriteVideoTrackTags embeds metadata into an MP4 video using ffmpeg stream copy.
// go-mp4tag cannot reliably rewrite video containers (moov layout / multiple tracks).
func WriteVideoTrackTags(ffmpegConfigured, path string, tags TrackTags) error {
	return WriteVideoTrackTagsTo(ffmpegConfigured, path, path, tags)
}

// WriteVideoTrackTagsTo writes tagged MP4 to dst (stream copy from src). When src != dst the source file is kept.
func WriteVideoTrackTagsTo(ffmpegConfigured, srcPath, dstPath string, tags TrackTags) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("video file is empty")
	}

	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	dir := filepath.Dir(dstPath)
	tmp, err := os.CreateTemp(dir, ".amd-tag-*.mp4")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	args := []string{"-y", "-loglevel", "error", "-i", srcPath}

	var coverCleanup func()
	coverPath, coverCleanup, coverErr := PrepareCoverFile(tags)
	if coverErr == nil && coverPath != "" {
		defer coverCleanup()
		args = append(args, "-i", coverPath)
	}

	args = append(args, "-map", "0:v:0", "-map", "0:a:0")
	if coverPath != "" {
		args = append(args,
			"-map", "1",
			"-c", "copy",
			"-disposition:v:1", "attached_pic",
			"-metadata:s:v:1", "mimetype=image/jpeg",
		)
	} else {
		args = append(args, "-c", "copy")
	}

	args = appendFFmpegMetadataArgs(args, tags)
	args = append(args, "-movflags", "+faststart", tmpPath)

	out, err := exec.Command(ffmpeg, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ffmpeg metadata embed failed: %s", msg)
	}

	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("replace tagged video: %w", err)
	}
	if err := os.Rename(tmpPath, dstPath); err != nil {
		return fmt.Errorf("replace tagged video: %w", err)
	}
	return nil
}

func appendFFmpegMetadataArgs(args []string, tags TrackTags) []string {
	title := strings.TrimSpace(tags.Title)
	artist := strings.TrimSpace(tags.Artist)
	album := strings.TrimSpace(tags.Album)
	albumArtist := strings.TrimSpace(tags.AlbumArtist)
	if title == "" {
		title = "Unknown Title"
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = title
	}
	if albumArtist == "" {
		albumArtist = artist
	}

	pairs := []struct{ key, val string }{
		{"title", title},
		{"artist", artist},
		{"album", album},
		{"album_artist", albumArtist},
		{"date", strings.TrimSpace(tags.Year)},
		{"genre", strings.TrimSpace(tags.Genre)},
		{"composer", artist},
	}
	for _, p := range pairs {
		if p.val != "" {
			args = append(args, "-metadata", p.key+"="+p.val)
		}
	}
	trackNum := tags.TrackNumber
	if trackNum <= 0 {
		trackNum = 1
	}
	trackTotal := tags.TrackTotal
	if trackTotal <= 0 {
		trackTotal = trackNum
	}
	discNum := tags.DiscNumber
	if discNum <= 0 {
		discNum = 1
	}
	discTotal := tags.DiscTotal
	if discTotal <= 0 {
		discTotal = 1
	}
	args = append(args,
		"-metadata", fmt.Sprintf("track=%d/%d", trackNum, trackTotal),
		"-metadata", fmt.Sprintf("disc=%d/%d", discNum, discTotal),
	)
	return args
}
