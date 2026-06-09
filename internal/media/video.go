package media

import (
	"fmt"
	"os/exec"
	"strings"

	appconfig "main/internal/config"
)

// ConvertVideoToAppleMP4 remuxes H.264 video and re-encodes audio to AAC-LC stereo for iOS Apple Music.
func ConvertVideoToAppleMP4(ffmpegConfigured, src, dst string) error {
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	enc := DetectAACEncoder(ffmpegConfigured)
	audioTail := append(append([]string{}, enc.Parameters...), "-ac", "2", "-movflags", "+faststart")

	attempts := [][]string{
		buildAppleMP4Attempt(true, audioTail),
		buildAppleMP4Attempt(false, audioTail),
		{
			"-map", "0:v:0", "-c:v", "copy",
			"-sn", "-dn",
			"-map", "0:a:0", "-c:a", "aac", "-b:a", IPhoneAACBitrate, "-profile:a", "aac_low", "-ac", "2",
			"-movflags", "+faststart",
		},
	}

	var lastErr error
	for _, mid := range attempts {
		args := append([]string{"-y", "-i", src, "-loglevel", "error"}, mid...)
		args = append(args, dst)
		out, err := exec.Command(ffmpeg, args...).CombinedOutput()
		if err == nil {
			return nil
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		lastErr = fmt.Errorf("Apple Music MP4 conversion failed: %s", msg)
	}
	return lastErr
}

func buildAppleMP4Attempt(copyVideo bool, audioTail []string) []string {
	videoPart := []string{"-map", "0:v:0?", "-sn", "-dn"}
	if copyVideo {
		videoPart = append(videoPart, "-c:v", "copy")
	} else {
		videoPart = append(videoPart, "-c:v", "libx264", "-preset", "fast", "-crf", "23", "-pix_fmt", "yuv420p")
	}
	audioPart := append([]string{"-map", "0:a:0?"}, audioTail...)
	return append(videoPart, audioPart...)
}
