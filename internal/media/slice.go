package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

// ExportSlice extracts [startMs, endMs) from masterPath into dstPath as AAC M4A.
func ExportSlice(ffmpegConfigured, masterPath, dstPath string, startMs, endMs, sampleRate, channels int) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", err
	}
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	enc := DetectAACEncoder(ffmpegConfigured)
	start := formatFFmpegTime(startMs)
	end := formatFFmpegTime(endMs)
	args := []string{
		"-y",
		"-i", masterPath,
		"-ss", start,
		"-to", end,
		"-vn", "-sn", "-dn",
		"-map", "0:a:0?",
	}
	args = append(args, enc.Parameters...)
	if sampleRate > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", sampleRate))
	}
	if channels > 0 {
		args = append(args, "-ac", fmt.Sprintf("%d", channels))
	}
	args = append(args, "-movflags", "+faststart", dstPath)

	out, err := exec.Command(ffmpeg, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("slice export failed: %s", msg)
	}
	return enc.Label, nil
}

func formatFFmpegTime(ms int) string {
	if ms < 0 {
		ms = 0
	}
	sec := ms / 1000
	rem := ms % 1000
	return fmt.Sprintf("%d.%03d", sec, rem)
}
