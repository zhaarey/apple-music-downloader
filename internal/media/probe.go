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
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Disposition struct {
		AttachedPic int `json:"attached_pic"`
	} `json:"disposition"`
}

// VideoFileInfo describes an MP4 music video for the Tag Editor.
type VideoFileInfo struct {
	VideoCodec    string `json:"video_codec"`
	AudioCodec    string `json:"audio_codec"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	DurationMs    int    `json:"duration_ms"`
	DurationLabel string `json:"duration_label"`
	AudioChannels int    `json:"audio_channels"`
	AppleReady    bool   `json:"apple_ready"`
	AppleDetail   string `json:"apple_detail,omitempty"`
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

func primaryVideoStream(streams []ffprobeStream) (ffprobeStream, bool) {
	for _, stream := range streams {
		if stream.CodecType != "video" || stream.Disposition.AttachedPic == 1 {
			continue
		}
		if isH264Codec(stream.CodecName) {
			return stream, true
		}
	}
	for _, stream := range streams {
		if stream.CodecType == "video" && stream.Disposition.AttachedPic != 1 {
			return stream, true
		}
	}
	return ffprobeStream{}, false
}

// ProbeVideoFile reads MP4 stream details for the Tag Editor and Apple Music checks.
func ProbeVideoFile(ffmpegConfigured, path string) (VideoFileInfo, error) {
	result, err := probeFile(ffmpegConfigured, path)
	if err != nil {
		return VideoFileInfo{}, err
	}
	info := VideoFileInfo{}
	if result.Format.Duration != "" {
		if sec, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			info.DurationMs = int(sec * 1000)
			info.DurationLabel = FormatDuration(info.DurationMs, info.DurationMs >= 3600000)
		}
	}
	streamInfo, err := validateAppleMusicStreams(result.Streams)
	if err != nil {
		info.AppleDetail = err.Error()
	} else {
		info.AppleReady = true
		info.AppleDetail = "H.264 video + AAC stereo"
	}
	info.VideoCodec = streamInfo.VideoCodec
	info.AudioCodec = streamInfo.AudioCodec
	info.AudioChannels = streamInfo.AudioChannels
	if video, ok := primaryVideoStream(result.Streams); ok {
		info.Width = video.Width
		info.Height = video.Height
		if info.VideoCodec == "" {
			info.VideoCodec = video.CodecName
		}
	}
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
