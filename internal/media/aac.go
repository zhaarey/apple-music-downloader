package media

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	appconfig "main/internal/config"
)

const IPhoneAACBitrate = "256k"

// AACEncoder describes an ffmpeg AAC encoder configuration.
type AACEncoder struct {
	Name       string
	Label      string
	Parameters []string
}

var (
	encoderOnce sync.Once
	encoder     AACEncoder
)

var encoderPriority = []string{"libfdk_aac", "aac_mf", "aac"}

// DetectAACEncoder picks the best available AAC encoder (cached).
func DetectAACEncoder(ffmpegConfigured string) AACEncoder {
	encoderOnce.Do(func() {
		available := ffmpegEncoders(ffmpegConfigured)
		for _, name := range encoderPriority {
			if _, ok := available[name]; ok {
				encoder = encoderConfig(name)
				return
			}
		}
		encoder = encoderConfig("aac")
	})
	return encoder
}

func encoderConfig(name string) AACEncoder {
	switch name {
	case "libfdk_aac":
		return AACEncoder{
			Name:  name,
			Label: "AAC (libfdk_aac VBR5 — highest quality)",
			Parameters: []string{
				"-c:a", "libfdk_aac", "-vbr", "5",
			},
		}
	case "aac_mf":
		return AACEncoder{
			Name:  name,
			Label: "AAC (MediaFoundation — 256 kbps, Windows)",
			Parameters: []string{
				"-c:a", "aac_mf", "-b:a", IPhoneAACBitrate,
			},
		}
	default:
		return AACEncoder{
			Name:  "aac",
			Label: "AAC-LC (256 kbps — Apple Music tier)",
			Parameters: []string{
				"-c:a", "aac", "-b:a", IPhoneAACBitrate, "-profile:a", "aac_low",
			},
		}
	}
}

func ffmpegEncoders(ffmpegConfigured string) map[string]struct{} {
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	out, err := exec.Command(ffmpeg, "-hide_banner", "-encoders").CombinedOutput()
	if err != nil {
		return map[string]struct{}{"aac": {}}
	}
	set := make(map[string]struct{})
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasPrefix(parts[0], "A") {
			set[parts[1]] = struct{}{}
		}
	}
	if len(set) == 0 {
		set["aac"] = struct{}{}
	}
	return set
}

// ConvertToAppleAAC re-encodes a file to AAC 256k M4A for Apple Music import.
func ConvertToAppleAAC(ffmpegConfigured, src, dst string) error {
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	enc := DetectAACEncoder(ffmpegConfigured)
	attempts := [][]string{
		append([]string{"-vn", "-sn", "-dn", "-map", "0:a:0?"}, append(enc.Parameters, "-movflags", "+faststart")...),
		append([]string{"-vn", "-sn", "-dn", "-map", "0:a:0"}, append(enc.Parameters, "-movflags", "+faststart")...),
		{"-vn", "-sn", "-dn", "-map", "0:a", "-c:a", "aac", "-b:a", IPhoneAACBitrate, "-movflags", "+faststart"},
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
		lastErr = fmt.Errorf("AAC conversion failed: %s", msg)
	}
	return lastErr
}
