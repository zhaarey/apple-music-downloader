// Package crossdevice defines the album export layout shared between the iOS IPA
// and the desktop Tag Editor / Trim workflows. See docs/ios/cross-device-export.md.
package crossdevice

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// ContractVersion is bumped when folder layout or encoding rules change.
	ContractVersion = 1

	// Encoding (mirror internal/media/aac.go and video.go).
	IPhoneAACBitrate = "256k"
	AACProfile       = "aac_low"
	AudioChannels    = 2

	// Artwork (mirror internal/media/artwork.go).
	MaxCoverEdgePx    = 3000
	TargetCoverEdgePx = 1200
	CoverJPEGQuality  = 90
	CoverSidecarName  = "cover.jpg"

	// Filename patterns.
	VideoSuffix     = " [video]"
	AudioExtension  = ".m4a"
	VideoExtension  = ".mp4"
	ManifestName    = ".aura-album.json"
	IOSDownloadsDir = "Downloads"

	// Import states for IPA post-download flow (see docs/ios/music-import-ux.md).
	StateDownloaded      = "downloaded"
	StateValidated       = "validated"
	StateReadyForImport  = "ready_for_import"
	StateImportGuided    = "import_guided"
	StateRetained        = "retained"
)

var forbiddenNames = regexp.MustCompile(`[/\\<>:"|?*]`)

// AlbumManifest is written beside tracks on iOS for PC pickup and import tracking.
type AlbumManifest struct {
	ContractVersion int    `json:"contract_version"`
	Album           string `json:"album"`
	AlbumArtist     string `json:"album_artist,omitempty"`
	Source          string `json:"source"` // apple | youtube
	ImportState     string `json:"import_state"`
	TrackCount      int    `json:"track_count"`
	HasVideo        bool   `json:"has_video,omitempty"`
	ExportedAt      string `json:"exported_at,omitempty"` // RFC3339
}

// SanitizePathPart matches internal/media.SanitizePathPart.
func SanitizePathPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Unknown"
	}
	return forbiddenNames.ReplaceAllString(s, "_")
}

// TrackFilename builds "01. Title.m4a" style names.
func TrackFilename(trackNumber int, title, ext string) string {
	title = SanitizePathPart(title)
	if trackNumber > 0 {
		return fmt.Sprintf("%02d. %s%s", trackNumber, title, ext)
	}
	return title + ext
}

// VideoFilename builds "01. Title [video].mp4".
func VideoFilename(trackNumber int, title string) string {
	base := TrackFilename(trackNumber, title, "")
	return base + VideoSuffix + VideoExtension
}

// AlbumRelativePath returns the album folder name under the downloads root.
func AlbumRelativePath(albumTitle string) string {
	dir := SanitizePathPart(albumTitle)
	if dir == "" || dir == "Unknown" {
		return "YouTube Downloads"
	}
	return dir
}

// IsVideoTrackFilename reports whether a basename is a music-video export.
func IsVideoTrackFilename(name string) bool {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return strings.HasSuffix(base, VideoSuffix)
}

// AudioStemFromVideo returns the .m4a basename stem for a [video].mp4 file.
func AudioStemFromVideo(videoBasename string) string {
	base := strings.TrimSuffix(videoBasename, VideoExtension)
	base = strings.TrimSuffix(base, VideoSuffix)
	return base + AudioExtension
}
