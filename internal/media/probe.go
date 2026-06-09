package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	appconfig "main/internal/config"
)

// SourceInfo describes a local audio file.
type SourceInfo struct {
	DurationMs  int    `json:"duration_ms"`
	SampleRate  int    `json:"sample_rate"`
	Channels    int    `json:"channels"`
	Summary     string `json:"summary"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

type ffprobeStream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	Profile    string `json:"profile"`
	PixFmt     string `json:"pix_fmt"`
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

// VideoProbeInfo describes streams in a video file for Apple Music validation.
type VideoProbeInfo struct {
	HasH264Video bool
	HasAACAudio  bool
	AudioChannels int
	VideoCodec   string
	AudioCodec   string
}

type ffprobeResult struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

// ProbeSource reads duration and audio stream info via ffprobe.
func ProbeSource(ffmpegConfigured, path string) (SourceInfo, error) {
	result, err := probeFile(ffmpegConfigured, path)
	if err != nil {
		return SourceInfo{}, err
	}

	info := SourceInfo{}
	if result.Format.Duration != "" {
		if sec, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			info.DurationMs = int(sec * 1000)
		}
	}
	for _, stream := range result.Streams {
		if stream.CodecType != "audio" {
			continue
		}
		if stream.SampleRate != "" {
			if sr, err := strconv.Atoi(stream.SampleRate); err == nil {
				info.SampleRate = sr
			}
		}
		if stream.Channels > 0 {
			info.Channels = stream.Channels
		}
		break
	}
	info.Summary = formatSummary(info)
	return info, nil
}

func probeFile(ffmpegConfigured, path string) (ffprobeResult, error) {
	ffprobe := appconfig.FFprobePath(ffmpegConfigured)
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	}
	out, err := exec.Command(ffprobe, args...).CombinedOutput()
	if err != nil {
		return ffprobeResult{}, fmt.Errorf("ffprobe failed: %s", strings.TrimSpace(string(out)))
	}
	var result ffprobeResult
	if err := json.Unmarshal(out, &result); err != nil {
		return ffprobeResult{}, err
	}
	return result, nil
}

func isH264Codec(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "h264" || n == "avc1" || strings.HasPrefix(n, "avc")
}

func isAACCodec(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "aac")
}

func validateAppleMusicStreams(streams []ffprobeStream) (VideoProbeInfo, error) {
	info := VideoProbeInfo{}
	for _, stream := range streams {
		switch stream.CodecType {
		case "video":
			if info.VideoCodec == "" {
				info.VideoCodec = stream.CodecName
			}
			if isH264Codec(stream.CodecName) {
				info.HasH264Video = true
			}
		case "audio":
			if info.AudioCodec == "" {
				info.AudioCodec = stream.CodecName
			}
			if isAACCodec(stream.CodecName) && stream.Channels > 0 && stream.Channels <= 2 {
				info.HasAACAudio = true
				info.AudioChannels = stream.Channels
			}
		}
	}
	if !info.HasH264Video {
		if info.VideoCodec == "" {
			return info, fmt.Errorf("MP4 has no video track")
		}
		return info, fmt.Errorf("MP4 video is %s, not H.264 — Apple Music requires H.264", info.VideoCodec)
	}
	if !info.HasAACAudio {
		if info.AudioCodec == "" {
			return info, fmt.Errorf("MP4 has no AAC stereo audio track — Apple Music will play silently")
		}
		return info, fmt.Errorf("MP4 audio is %s, not AAC stereo — Apple Music will play silently", info.AudioCodec)
	}
	return info, nil
}

// ValidateAppleMusicMP4 checks that a file has H.264 video and AAC-LC stereo audio.
func ValidateAppleMusicMP4(ffmpegConfigured, path string) error {
	result, err := probeFile(ffmpegConfigured, path)
	if err != nil {
		return err
	}
	_, err = validateAppleMusicStreams(result.Streams)
	return err
}

func formatSummary(info SourceInfo) string {
	if info.DurationMs <= 0 {
		return "—"
	}
	ch := "stereo"
	if info.Channels == 1 {
		ch = "mono"
	} else if info.Channels > 2 {
		ch = fmt.Sprintf("%dch", info.Channels)
	}
	sr := info.SampleRate
	if sr >= 1000 {
		sr = sr / 1000
	}
	return fmt.Sprintf("%s · %dkHz %s", FormatDuration(info.DurationMs, info.DurationMs >= 3600000), sr, ch)
}
