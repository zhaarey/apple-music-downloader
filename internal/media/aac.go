package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

var videoEncoderOnce sync.Once
var videoEncoder AACEncoder

// appleAACEncoderParams returns iOS / Apple Music–safe AAC-LC 256 kbps settings.
func appleAACEncoderParams(encoderName string) []string {
	return []string{"-c:a", encoderName, "-b:a", IPhoneAACBitrate, "-profile:a", "aac_low"}
}

// VideoAACEncoder picks an iOS-safe AAC encoder for MP4 video exports (avoids aac_mf).
func VideoAACEncoder(ffmpegConfigured string) AACEncoder {
	videoEncoderOnce.Do(func() {
		available := ffmpegEncoders(ffmpegConfigured)
		if _, ok := available["aac"]; ok {
			videoEncoder = AACEncoder{
				Name:  "aac",
				Label: "AAC-LC (256 kbps — Apple Music video)",
				Parameters: appleAACEncoderParams("aac"),
			}
			return
		}
		if _, ok := available["aac_mf"]; ok {
			videoEncoder = AACEncoder{
				Name:  "aac_mf",
				Label: "AAC-LC (256 kbps — Apple Music video)",
				Parameters: appleAACEncoderParams("aac_mf"),
			}
			return
		}
		if _, ok := available["libfdk_aac"]; ok {
			videoEncoder = AACEncoder{
				Name:  "libfdk_aac",
				Label: "AAC-LC (256 kbps — Apple Music video)",
				Parameters: appleAACEncoderParams("libfdk_aac"),
			}
			return
		}
		videoEncoder = AACEncoder{
			Name:  "aac",
			Label: "AAC-LC (256 kbps — Apple Music video)",
			Parameters: appleAACEncoderParams("aac"),
		}
	})
	return videoEncoder
}

// DetectAACEncoder picks the best available AAC encoder for Apple Music import (cached).
// Prefers native AAC-LC 256 kbps for consistent quality on iPhone sync.
func DetectAACEncoder(ffmpegConfigured string) AACEncoder {
	encoderOnce.Do(func() {
		available := ffmpegEncoders(ffmpegConfigured)
		for _, name := range []string{"aac", "aac_mf", "libfdk_aac"} {
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
	case "aac_mf":
		return AACEncoder{
			Name:  name,
			Label: "AAC-LC (256 kbps — Apple Music)",
			Parameters: appleAACEncoderParams("aac_mf"),
		}
	case "libfdk_aac":
		return AACEncoder{
			Name:  name,
			Label: "AAC-LC (256 kbps — Apple Music)",
			Parameters: appleAACEncoderParams("libfdk_aac"),
		}
	default:
		return AACEncoder{
			Name:  "aac",
			Label: "AAC-LC (256 kbps — Apple Music)",
			Parameters: appleAACEncoderParams("aac"),
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

// IsConvertibleAudioExt reports whether the file type can be converted to AAC M4A for Apple Music.
func IsConvertibleAudioExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".mp3", ".flac", ".wav", ".aiff", ".aif", ".ogg", ".opus", ".wma", ".m4a", ".aac", ".mp4", ".m4b", ".caf", ".alac":
		return true
	default:
		return false
	}
}

// DefaultAppleAACOutputPath returns a sibling .m4a path that does not overwrite src.
func DefaultAppleAACOutputPath(src string) string {
	dir := filepath.Dir(src)
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	if base == "" {
		base = "audio"
	}
	candidate := filepath.Join(dir, base+".m4a")
	if !sameFilePath(candidate, src) {
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
	candidate = filepath.Join(dir, base+" - AAC.m4a")
	if _, err := os.Stat(candidate); err != nil {
		return candidate
	}
	for i := 2; i < 100; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s - AAC (%d).m4a", base, i))
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
	return filepath.Join(dir, base+" - AAC.m4a")
}

func sameFilePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// ConvertToAACResult is returned after converting a local file to Apple Music AAC.
type ConvertToAACResult struct {
	SourcePath string `json:"source_path"`
	OutputPath string `json:"output_path"`
	Encoder    string `json:"encoder"`
	Summary    string `json:"summary"`
}

// ConvertFileToAppleAAC converts src to AAC-LC 256 kbps M4A suitable for Apple Music import.
// When dst is empty, writes beside the source without overwriting it.
func ConvertFileToAppleAAC(ffmpegConfigured, src, dst string) (ConvertToAACResult, error) {
	src = strings.TrimSpace(src)
	dst = strings.TrimSpace(dst)
	out := ConvertToAACResult{SourcePath: src}
	if src == "" {
		return out, fmt.Errorf("no source file")
	}
	if !IsConvertibleAudioExt(filepath.Ext(src)) {
		return out, fmt.Errorf("unsupported file type %s — use MP3, FLAC, WAV, AIFF, OGG, Opus, WMA, AAC, or M4A", filepath.Ext(src))
	}
	if _, err := os.Stat(src); err != nil {
		return out, err
	}
	if dst == "" {
		dst = DefaultAppleAACOutputPath(src)
	}
	if !strings.EqualFold(filepath.Ext(dst), ".m4a") {
		dst = strings.TrimSuffix(dst, filepath.Ext(dst)) + ".m4a"
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return out, err
	}
	enc := DetectAACEncoder(ffmpegConfigured)
	if err := ConvertToAppleAAC(ffmpegConfigured, src, dst); err != nil {
		return out, err
	}
	out.OutputPath = dst
	out.Encoder = enc.Label
	out.Summary = fmt.Sprintf("Converted to AAC 256 kbps · %s", filepath.Base(dst))
	return out, nil
}

// ConvertToAppleAAC re-encodes a file to AAC-LC 256k M4A for Apple Music import.
func ConvertToAppleAAC(ffmpegConfigured, src, dst string) error {
	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	enc := DetectAACEncoder(ffmpegConfigured)
	metaFlags := []string{"-map_metadata", "0", "-map_chapters", "-1"}
	attempts := [][]string{
		append(append([]string{"-vn", "-sn", "-dn", "-map", "0:a:0?"}, append(enc.Parameters, metaFlags...)...), "-movflags", "+faststart"),
		append(append([]string{"-vn", "-sn", "-dn", "-map", "0:a:0"}, append(enc.Parameters, metaFlags...)...), "-movflags", "+faststart"),
		append([]string{"-vn", "-sn", "-dn", "-map", "0:a", "-c:a", "aac", "-b:a", IPhoneAACBitrate, "-profile:a", "aac_low"}, append(metaFlags, "-movflags", "+faststart")...),
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
