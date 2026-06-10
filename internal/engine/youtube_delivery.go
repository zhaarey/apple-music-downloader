package engine

import "strings"

type YouTubeDeliveryMode string

const (
	YouTubeDeliveryAudio YouTubeDeliveryMode = "audio"
	YouTubeDeliveryVideo YouTubeDeliveryMode = "video"
	YouTubeDeliveryBoth  YouTubeDeliveryMode = "both"
)

func NormalizeYouTubeDelivery(mode string) YouTubeDeliveryMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "video", "mp4":
		return YouTubeDeliveryVideo
	case "both", "audio+video", "audio_video", "all":
		return YouTubeDeliveryBoth
	default:
		return YouTubeDeliveryAudio
	}
}

func (m YouTubeDeliveryMode) SaveAudio() bool {
	return m != YouTubeDeliveryVideo
}

func (m YouTubeDeliveryMode) SaveVideo() bool {
	return m == YouTubeDeliveryVideo || m == YouTubeDeliveryBoth
}

func (m YouTubeDeliveryMode) ExistingMediaIsVideo() bool {
	return m == YouTubeDeliveryVideo
}

func (m YouTubeDeliveryMode) JobStartMessage() string {
	switch m {
	case YouTubeDeliveryVideo:
		return "Starting YouTube download (MP4 video for Apple Music)"
	case YouTubeDeliveryBoth:
		return "Starting YouTube download (AAC 256 kbps + MP4 video)"
	default:
		return "Starting YouTube download (AAC 256 kbps for Apple Music)"
	}
}

func (m YouTubeDeliveryMode) CompleteMessage() string {
	switch m {
	case YouTubeDeliveryVideo:
		return "Saved MP4 video ready for Apple Music import"
	case YouTubeDeliveryBoth:
		return "Saved AAC and MP4 files ready for Apple Music import"
	default:
		return "Saved AAC files ready for Apple Music import"
	}
}
