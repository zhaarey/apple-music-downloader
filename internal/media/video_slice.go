package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

// ExportVideoSlice extracts [startMs, endMs) from src into dst as Apple-friendly MP4.
func ExportVideoSlice(ffmpegConfigured, src, dst string, startMs, endMs int) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	enc := VideoAACEncoder(ffmpegConfigured)
	start := formatFFmpegTime(startMs)
	end := formatFFmpegTime(endMs)
	audioTail := append(append([]string{}, enc.Parameters...), "-ac", "2", "-movflags", "+faststart")

	attempts := [][]string{
		append(buildFFmpegMP4Attempt(true, audioTail), "-avoid_negative_ts", "make_zero"),
		append(buildFFmpegMP4Attempt(false, audioTail), "-avoid_negative_ts", "make_zero"),
		{
			"-map", "0:v:0", "-c:v", "copy",
			"-sn", "-dn",
			"-map", "0:a:0?", "-c:a", "aac", "-b:a", IPhoneAACBitrate, "-profile:a", "aac_low", "-ac", "2",
			"-movflags", "+faststart", "-avoid_negative_ts", "make_zero",
		},
	}

	var lastErr error
	for _, mid := range attempts {
		args := []string{
			"-y", "-i", src,
			"-ss", start, "-to", end,
			"-loglevel", "error",
		}
		args = append(args, mid...)
		args = append(args, dst)
		out, err := exec.Command(ffmpeg, args...).CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			lastErr = fmt.Errorf("video trim failed: %s", msg)
			_ = os.Remove(dst)
			continue
		}
		if valErr := validateOutput(ffmpegConfigured, dst); valErr != nil {
			lastErr = valErr
			_ = os.Remove(dst)
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("video trim failed")
	}
	return lastErr
}
