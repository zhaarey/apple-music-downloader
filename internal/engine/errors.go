package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

func releaseYear(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

func firstGenre(names []string) string {
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

func (e *Engine) validateDownload(opts RunOptions) error {
	if useYouTubePipeline(opts) {
		return e.validateYouTubeDownload(opts)
	}
	if len(opts.URLs) == 0 {
		return fmt.Errorf("no download URL provided")
	}
	for _, raw := range opts.URLs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return fmt.Errorf("download URL is empty")
		}
		if !strings.Contains(raw, "music.apple.com") && !strings.Contains(raw, "classical.music.apple.com") {
			return fmt.Errorf("invalid URL: must be an Apple Music link (got %q)", raw)
		}
		if e.DetectURLType(raw) == "Unknown" {
			return fmt.Errorf("unsupported URL type: %q (expected album, song, playlist, artist, station, or music-video link)", raw)
		}
	}
	for _, raw := range opts.ChildURLs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return fmt.Errorf("selected catalog URL is empty")
		}
		if !strings.Contains(raw, "music.apple.com") {
			return fmt.Errorf("invalid catalog URL: %q", raw)
		}
	}

	mp4box := appconfig.MP4BoxPath()
	if _, err := exec.LookPath(mp4box); err != nil {
		if _, statErr := os.Stat(mp4box); statErr != nil {
			return fmt.Errorf("MP4Box not found at %q: tagging requires MP4Box.exe in the app tools folder or on PATH", mp4box)
		}
	}

	saveFolder := Config.AacSaveFolder
	if opts.Quality == "atmos" {
		saveFolder = Config.AtmosSaveFolder
	} else if opts.Quality == "alac" || opts.Quality == "" {
		saveFolder = Config.AlacSaveFolder
	}
	if saveFolder == "" {
		return fmt.Errorf("output folder is not configured in Settings")
	}
	if err := os.MkdirAll(saveFolder, 0755); err != nil {
		return fmt.Errorf("cannot create output folder %q: %w", saveFolder, err)
	}

	if opts.Quality == "aac" && effectiveAacType() == "aac-lc" {
		if len(strings.TrimSpace(Config.MediaUserToken)) <= 50 {
			return fmt.Errorf("AAC downloads require a valid media-user-token in Settings (copy it from music.apple.com cookies in DevTools)")
		}
	}

	if opts.Quality == "alac" || opts.Quality == "atmos" {
		if !probePort(Config.DecryptM3u8Port) || !probePort(Config.GetM3u8Port) {
			return fmt.Errorf("%s downloads require the wrapper decrypt service on %s and %s (see Requirements tab for setup)", strings.ToUpper(opts.Quality), Config.DecryptM3u8Port, Config.GetM3u8Port)
		}
	}

	for _, raw := range opts.URLs {
		if strings.Contains(raw, "/station/") && len(strings.TrimSpace(Config.MediaUserToken)) <= 50 {
			return fmt.Errorf("station downloads require a valid media-user-token in Settings")
		}
		if strings.Contains(raw, "/music-video/") && len(strings.TrimSpace(Config.MediaUserToken)) <= 50 {
			return fmt.Errorf("music video downloads require a valid media-user-token in Settings")
		}
	}

	_ = filepath.Clean(saveFolder)
	return nil
}

func formatTokenError(err error) error {
	if err == nil {
		return nil
	}
	if Config.AuthorizationToken != "" && Config.AuthorizationToken != "your-authorization-token" {
		return fmt.Errorf("failed to get Apple Music API token: %w (fallback authorization-token also failed)", err)
	}
	return fmt.Errorf("failed to get Apple Music API token: %w. Check internet access or set authorization-token in Settings", err)
}
