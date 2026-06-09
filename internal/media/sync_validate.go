package media

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhaarey/go-mp4tag"
)

func check(id, label string, pass bool, detail, severity string) SyncCheck {
	if severity == "" {
		if pass {
			severity = "pass"
		} else {
			severity = "fail"
		}
	}
	return SyncCheck{ID: id, Label: label, Pass: pass, Detail: detail, Severity: severity}
}

// ValidateIPhoneSync inspects a local M4A for artwork and metadata iPhone sync expects.
func ValidateIPhoneSync(ffmpegConfigured, path string) (SyncValidationResult, error) {
	out := SyncValidationResult{Path: path, Checks: []SyncCheck{}}
	if path == "" {
		return out, fmt.Errorf("no file path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return out, err
	}
	if info.IsDir() {
		return out, fmt.Errorf("path is a folder, not a file")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".m4a" && ext != ".m4b" {
		out.Checks = append(out.Checks, check("format", "Audio format", false, "Use .m4a AAC files for music sync", "fail"))
		out.Summary = "Unsupported file type for iPhone music sync."
		return out, nil
	}

	tags, err := ReadAudioTags(path)
	if err != nil {
		return out, err
	}

	out.Checks = append(out.Checks, check("album_artist", "Album artist", strings.TrimSpace(tags.AlbumArtist) != "",
		albumArtistDetail(tags), "fail"))

	out.Checks = append(out.Checks, artworkChecks(path, tags)...)

	if probe, err := ProbeSource(ffmpegConfigured, path); err == nil {
		detail := fmt.Sprintf("%dkHz · %d channel(s)", probe.SampleRate/1000, probe.Channels)
		out.Checks = append(out.Checks, check("audio_stream", "Readable audio stream", true, detail, "pass"))
	} else {
		out.Checks = append(out.Checks, check("audio_stream", "Readable audio stream", false, err.Error(), "fail"))
	}

	if err := validateAACForIPhone(ffmpegConfigured, path); err != nil {
		out.Checks = append(out.Checks, check("aac_codec", "AAC-LC codec", false, err.Error(), "warn"))
	} else {
		out.Checks = append(out.Checks, check("aac_codec", "AAC-LC codec", true, "AAC stereo suitable for Apple Music", "pass"))
	}

	if tags.TrackNumber <= 0 {
		out.Checks = append(out.Checks, check("track_number", "Track number", false, "Set track number for album ordering", "warn"))
	}
	if tags.TrackTotal <= 0 {
		out.Checks = append(out.Checks, check("track_total", "Track total", false, "Set track total to match album size", "warn"))
	}

	out.Ready = true
	failures := 0
	warnings := 0
	for _, c := range out.Checks {
		if c.Pass {
			continue
		}
		if c.Severity == "warn" {
			warnings++
			continue
		}
		failures++
		out.Ready = false
	}
	if out.Ready && warnings > 0 {
		out.Summary = fmt.Sprintf("Ready with %d warning(s) — iPhone sync should work after re-import.", warnings)
	} else if out.Ready {
		out.Summary = "Embedded JPEG artwork and metadata look good for iPhone sync."
	} else {
		out.Summary = fmt.Sprintf("%d issue(s) found — fix in Tag Editor, save tags, then re-import in Apple Music.", failures)
	}
	return out, nil
}

func albumArtistDetail(tags AudioTagInfo) string {
	if strings.TrimSpace(tags.AlbumArtist) != "" {
		return tags.AlbumArtist
	}
	return "Missing — set album artist so album art groups on iPhone"
}

func warnIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "warn"
	}
	return "fail"
}

func artworkChecks(path string, tags AudioTagInfo) []SyncCheck {
	checks := []SyncCheck{}
	dir := filepath.Dir(path)
	if sidecar := FindAlbumCoverFile(dir); sidecar != "" && !tags.HasArtwork {
		checks = append(checks, check("sidecar_only", "Embedded vs folder art", false,
			"Folder has "+filepath.Base(sidecar)+" but file has no embedded cover — Apple Music sync needs covr in each track", "fail"))
	}
	if !tags.HasArtwork {
		checks = append(checks, check("embedded_art", "Embedded artwork", false,
			"No cover in file — PC Apple Music may still show folder art", "fail"))
		return checks
	}
	checks = append(checks, check("embedded_art", "Embedded artwork", true, "Cover found in file tags", "pass"))

	mime := strings.ToLower(tags.ArtworkMime)
	if mime == "image/jpeg" || mime == "image/jpg" {
		checks = append(checks, check("art_format", "Artwork format", true, "JPEG (recommended for iOS)", "pass"))
	} else if mime == "image/png" {
		checks = append(checks, check("art_format", "Artwork format", false, "PNG — save tags to convert to JPEG", "fail"))
	} else if mime != "" {
		checks = append(checks, check("art_format", "Artwork format", false, mime+" — save tags to normalize", "warn"))
	} else {
		checks = append(checks, check("art_format", "Artwork format", false, "Unknown format — save tags to normalize", "warn"))
	}

	if tags.ArtworkCount > 1 {
		checks = append(checks, check("art_count", "Single artwork", false,
			fmt.Sprintf("%d embedded covers — save once to keep one", tags.ArtworkCount), "warn"))
	} else {
		checks = append(checks, check("art_count", "Single artwork", true, "One embedded cover", "pass"))
	}

	checks = append(checks, artworkDimensionDetail(tags.Path))
	return checks
}

func artworkDimensionDetail(path string) SyncCheck {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return check("art_size", "Artwork size", false, "Could not read cover dimensions", "warn")
	}
	defer mp4.Close()
	existing, err := mp4.Read()
	if err != nil || existing == nil || len(existing.Pictures) == 0 {
		return check("art_size", "Artwork size", false, "No cover data to measure", "warn")
	}
	pic := existing.Pictures[0]
	if len(existing.Pictures) > 1 {
		pic = existing.Pictures[len(existing.Pictures)-1]
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(pic.Data))
	if err != nil {
		return check("art_size", "Artwork size", false, "Could not decode cover image", "warn")
	}
	maxEdge := cfg.Width
	if cfg.Height > maxEdge {
		maxEdge = cfg.Height
	}
	detail := fmt.Sprintf("%d×%d px", cfg.Width, cfg.Height)
	if maxEdge > MaxCoverEdgePx {
		return check("art_size", "Artwork size", false, detail+" — save tags to resize to ≤3000px", "warn")
	}
	return check("art_size", "Artwork size", true, detail+" (within iOS limit)", "pass")
}

func validateAACForIPhone(ffmpegConfigured, path string) error {
	result, err := probeFile(ffmpegConfigured, path)
	if err != nil {
		return err
	}
	for _, stream := range result.Streams {
		if stream.CodecType != "audio" {
			continue
		}
		if !isAACCodec(stream.CodecName) {
			return fmt.Errorf("codec is %s, expected AAC", stream.CodecName)
		}
		if stream.Channels > 2 {
			return fmt.Errorf("%d channels — iPhone sync expects stereo", stream.Channels)
		}
		return nil
	}
	return fmt.Errorf("no audio stream found")
}

// ValidateIPhoneSyncFolder validates .m4a files directly in a folder (same scope as Prepare from Tag Editor).
func ValidateIPhoneSyncFolder(ffmpegConfigured, folder string) (FolderSyncValidationResult, error) {
	return ValidateAlbumSync(ffmpegConfigured, folder, false)
}
