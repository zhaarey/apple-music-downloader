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
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

type ffprobeResult struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

// ProbeSource reads duration and audio stream info via ffprobe.
func ProbeSource(ffmpegConfigured, path string) (SourceInfo, error) {
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
		return SourceInfo{}, fmt.Errorf("ffprobe failed: %s", strings.TrimSpace(string(out)))
	}

	var result ffprobeResult
	if err := json.Unmarshal(out, &result); err != nil {
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
