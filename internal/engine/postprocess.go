package engine

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	appconfig "main/internal/config"
	"main/internal/media"
	"main/utils/alacfix"
	"main/utils/task"
)

func isMissingMP4MetaBox(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "box not present") {
		return false
	}
	return strings.Contains(msg, "udta") ||
		strings.Contains(msg, "meta") ||
		strings.Contains(msg, "ilst")
}

func mp4boxITagValue(value string) string {
	return strings.NewReplacer(":", "\\:").Replace(strings.TrimSpace(value))
}

func mp4boxBootstrapTags(track *task.Track, lrc string) error {
	tags := trackToMediaTags(track, lrc)
	coverPath := strings.TrimSpace(tags.CoverPath)
	if coverPath == "" && Config.EmbedCover {
		if err := prepareDownloadArtwork(track); err != nil {
			return err
		}
		coverPath = track.CoverPath
	}

	parts := []string{
		"tool=",
		fmt.Sprintf("title=%s", mp4boxITagValue(tags.Title)),
		fmt.Sprintf("artist=%s", mp4boxITagValue(tags.Artist)),
		fmt.Sprintf("album=%s", mp4boxITagValue(tags.Album)),
		fmt.Sprintf("album_artist=%s", mp4boxITagValue(tags.AlbumArtist)),
		fmt.Sprintf("genre=%s", mp4boxITagValue(tags.Genre)),
		fmt.Sprintf("date=%s", mp4boxITagValue(tags.Year)),
		fmt.Sprintf("track=%d", tags.TrackNumber),
		fmt.Sprintf("tracknum=%d/%d", max16(tags.TrackNumber, 1), max16(tags.TrackTotal, max16(tags.TrackNumber, 1))),
		fmt.Sprintf("disk=%d/%d", max16(tags.DiscNumber, 1), max16(tags.DiscTotal, 1)),
		fmt.Sprintf("performer=%s", mp4boxITagValue(tags.Artist)),
	}
	if Config.EmbedCover && coverPath != "" {
		parts = append(parts, fmt.Sprintf("cover=%s", coverPath))
	}
	cmd := exec.Command(mp4boxPath(), "-itags", strings.Join(parts, ":"), track.SavePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("MP4Box -itags failed (%s): %w", mp4boxPath(), err)
	}
	return nil
}

func max16(v, fallback int16) int16 {
	if v > 0 {
		return v
	}
	return fallback
}

func writeMP4TagsWithBootstrap(track *task.Track, lrc string) error {
	if Config.EmbedCover {
		if err := prepareDownloadArtwork(track); err != nil {
			return fmt.Errorf("artwork: %w", err)
		}
	}
	if !media.HasWritableAppleTags(track.SavePath) {
		if err := mp4boxBootstrapTags(track, lrc); err != nil {
			return err
		}
	}
	err := writeMP4TagsForDownload(track, lrc)
	if err != nil && isMissingMP4MetaBox(err) {
		if bootErr := mp4boxBootstrapTags(track, lrc); bootErr != nil {
			return fmt.Errorf("%w (after missing meta box: %v)", bootErr, err)
		}
		err = writeMP4TagsForDownload(track, lrc)
	}
	return err
}

func flattenAndEmbedCover(track *task.Track, trackPath string) error {
	cmd := exec.Command(mp4boxPath(), "-flat", trackPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("MP4Box failed (%s): %w", mp4boxPath(), err)
	}
	if (strings.Contains(track.PreID, "pl.") || strings.Contains(track.PreID, "ra.")) &&
		Config.DlAlbumcoverForPlaylist && track.CoverPath != "" {
		if err := os.Remove(track.CoverPath); err != nil {
			return fmt.Errorf("cover cleanup failed: %w", err)
		}
	}
	return nil
}

func verifyIPhoneSyncAfterFinalize(track *task.Track, lrc string) error {
	ffmpeg := appconfig.FFmpegPath(Config.FFmpegPath)
	res, err := media.ValidateIPhoneSync(ffmpeg, track.SavePath)
	if err != nil {
		return nil
	}
	if res.Ready {
		return nil
	}
	needsArtRetag := false
	for _, c := range res.Checks {
		if c.Pass {
			continue
		}
		switch c.ID {
		case "embedded_art", "art_format", "art_count", "sidecar_only":
			needsArtRetag = true
		}
	}
	if needsArtRetag {
		if err := writeMP4TagsForDownload(track, lrc); err != nil {
			return fmt.Errorf("re-embed artwork after flatten: %w", err)
		}
		res, err = media.ValidateIPhoneSync(ffmpeg, track.SavePath)
		if err != nil {
			return nil
		}
	}
	if !res.Ready {
		fmt.Printf("iPhone sync note — %q: %s\n", track.Resp.Attributes.Name, res.Summary)
	}
	return nil
}

func finalizeTrackFile(track *task.Track, trackPath, lrc string) error {
	track.SavePath = trackPath
	// AAC-LC downloads from runv3 are fragmented MP4; flatten before go-mp4tag touches the file.
	if err := flattenAndEmbedCover(track, trackPath); err != nil {
		return err
	}
	if Config.ALACFix {
		if err := alacfix.Run(track.SavePath, false); err != nil {
			return fmt.Errorf("ALAC fix failed: %w", err)
		}
	}
	if err := writeMP4TagsWithBootstrap(track, lrc); err != nil {
		return err
	}
	return verifyIPhoneSyncAfterFinalize(track, lrc)
}
