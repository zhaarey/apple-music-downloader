package engine

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"main/utils/alacfix"
	"main/utils/task"
)

func flattenAndEmbedCover(track *task.Track, trackPath string) error {
	// Flatten only — artwork and metadata are applied in writeMP4Tags (iOS-safe JPEG covr).
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

func finalizeTrackFile(track *task.Track, trackPath, lrc string) error {
	if err := flattenAndEmbedCover(track, trackPath); err != nil {
		return err
	}
	track.SavePath = trackPath
	if Config.ALACFix {
		if err := alacfix.Run(track.SavePath, false); err != nil {
			return fmt.Errorf("ALAC fix failed: %w", err)
		}
	}
	return writeMP4Tags(track, lrc)
}
