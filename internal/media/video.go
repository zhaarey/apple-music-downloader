package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

// ConvertVideoToAppleMP4 remuxes H.264 video and re-encodes audio to AAC-LC stereo for iOS Apple Music.
func ConvertVideoToAppleMP4(ffmpegConfigured, src, dst string) error {
	if err := convertViaMP4Box(ffmpegConfigured, src, dst); err == nil {
		return nil
	}
	return convertViaFFmpeg(ffmpegConfigured, src, dst)
}

func convertViaMP4Box(ffmpegConfigured, src, dst string) error {
	mp4box := appconfig.MP4BoxPath()
	if _, err := exec.LookPath(mp4box); err != nil {
		if _, statErr := os.Stat(mp4box); statErr != nil {
			return fmt.Errorf("MP4Box not found")
		}
	}

	dir := filepath.Dir(dst)
	base := strings.TrimSuffix(filepath.Base(dst), filepath.Ext(dst))
	vidTemp := filepath.Join(dir, base+"._vid.tmp.mp4")
	audTemp := filepath.Join(dir, base+"._aud.tmp.m4a")
	defer os.Remove(vidTemp)
	defer os.Remove(audTemp)

	if err := extractVideoTrack(ffmpegConfigured, src, vidTemp); err != nil {
		return err
	}
	if err := extractAudioTrack(ffmpegConfigured, src, audTemp); err != nil {
		return err
	}

	_ = os.Remove(dst)
	cmd := exec.Command(mp4box, "-quiet", "-add", vidTemp, "-add", audTemp, "-new", dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("MP4Box mux failed: %s", msg)
	}
	return validateOutput(ffmpegConfigured, dst)
}

func convertViaFFmpeg(ffmpegConfigured, src, dst string) error {
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	enc := VideoAACEncoder(ffmpegConfigured)
	audioTail := append(append([]string{}, enc.Parameters...), "-ac", "2", "-movflags", "+faststart")

	attempts := [][]string{
		buildFFmpegMP4Attempt(true, audioTail),
		buildFFmpegMP4Attempt(false, audioTail),
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
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			lastErr = fmt.Errorf("Apple Music MP4 conversion failed: %s", msg)
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
		lastErr = fmt.Errorf("Apple Music MP4 conversion failed")
	}
	return lastErr
}

func extractVideoTrack(ffmpegConfigured, src, dst string) error {
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	attempts := [][]string{
		{"-map", "0:v:0", "-an", "-sn", "-dn", "-c:v", "copy"},
		{"-map", "0:v:0", "-an", "-sn", "-dn", "-c:v", "libx264", "-preset", "fast", "-crf", "23", "-pix_fmt", "yuv420p"},
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
		lastErr = fmt.Errorf("video extract failed: %s", msg)
	}
	return lastErr
}

func extractAudioTrack(ffmpegConfigured, src, dst string) error {
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	enc := VideoAACEncoder(ffmpegConfigured)
	args := append(
		[]string{"-y", "-i", src, "-loglevel", "error", "-map", "0:a:0", "-vn", "-sn", "-dn"},
		enc.Parameters...,
	)
	args = append(args, "-ac", "2", "-movflags", "+faststart", dst)
	out, err := exec.Command(ffmpeg, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("audio extract failed: %s", msg)
	}
	return nil
}

func buildFFmpegMP4Attempt(copyVideo bool, audioTail []string) []string {
	videoPart := []string{"-map", "0:v:0", "-sn", "-dn"}
	if copyVideo {
		videoPart = append(videoPart, "-c:v", "copy")
	} else {
		videoPart = append(videoPart, "-c:v", "libx264", "-preset", "fast", "-crf", "23", "-pix_fmt", "yuv420p")
	}
	audioPart := append([]string{"-map", "0:a:0"}, audioTail...)
	return append(videoPart, audioPart...)
}

func validateOutput(ffmpegConfigured, path string) error {
	if err := ValidateAppleMusicMP4(ffmpegConfigured, path); err != nil {
		return fmt.Errorf("MP4 validation failed: %w", err)
	}
	return nil
}
