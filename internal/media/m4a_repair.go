package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

// RemuxM4ACopy remuxes an M4A/MP4 with stream copy to repair damaged metadata atoms
// without re-encoding audio. Writes to dst (must differ from src).
func RemuxM4ACopy(ffmpegConfigured, src, dst string) error {
	src = strings.TrimSpace(src)
	dst = strings.TrimSpace(dst)
	if src == "" || dst == "" {
		return fmt.Errorf("missing remux paths")
	}
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		return fmt.Errorf("remux destination must differ from source")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	args := []string{
		"-y", "-i", src,
		"-map", "0:a:0?",
		"-c", "copy",
		"-map_metadata", "0",
		"-movflags", "+faststart",
		"-loglevel", "error",
		dst,
	}
	out, err := exec.Command(ffmpeg, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("remux failed: %s", msg)
	}
	return nil
}

// RepairAndWriteTrackTags remuxes a damaged M4A then writes Apple Music tags.
// If dst equals src, remuxes to a temp file and replaces src on success.
func RepairAndWriteTrackTags(ffmpegConfigured, path string, tags TrackTags) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("no file path")
	}
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	tmp := filepath.Join(dir, base+".repair-tmp.m4a")
	defer os.Remove(tmp)

	if err := RemuxM4ACopy(ffmpegConfigured, path, tmp); err != nil {
		return err
	}
	if err := WriteTrackTags(tmp, tags); err != nil {
		return fmt.Errorf("write after remux: %w", err)
	}
	bak := path + ".bak"
	_ = os.Remove(bak)
	if err := os.Rename(path, bak); err != nil {
		// Windows may lock; try overwrite via copy
		if copyErr := copyMediaFile(tmp, path); copyErr != nil {
			return fmt.Errorf("replace original after remux: %v / %v", err, copyErr)
		}
		_ = os.Remove(tmp)
		return nil
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Rename(bak, path) // rollback
		return fmt.Errorf("install remuxed file: %w", err)
	}
	_ = os.Remove(bak)
	return nil
}
