package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

// PreflightCheck is one readiness check before starting a download.
type PreflightCheck struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail"`
	Blocking bool   `json:"blocking"`
}

// PreflightResult summarizes whether a download can start.
type PreflightResult struct {
	Ready   bool             `json:"ready"`
	Summary string           `json:"summary"`
	Checks  []PreflightCheck `json:"checks"`
}

func (e *Engine) PreflightDownload(opts RunOptions) PreflightResult {
	checks := []PreflightCheck{}
	add := func(id, label string, ok bool, detail string, blocking bool) {
		checks = append(checks, PreflightCheck{ID: id, Label: label, OK: ok, Detail: detail, Blocking: blocking})
	}

	if err := e.validateDownload(opts); err != nil {
		add("validation", "Download request", false, err.Error(), true)
	} else {
		add("validation", "Download request", true, "URL and output folder look good", false)
	}

	if useYouTubePipeline(opts) {
		yt := appconfig.YtDlpPath(Config.YtDlpPath)
		if _, lerr := exec.LookPath(yt); lerr != nil {
			if _, serr := os.Stat(yt); serr != nil {
				add("yt-dlp", "yt-dlp", false, "Install yt-dlp or add to tools folder", true)
			} else {
				add("yt-dlp", "yt-dlp", true, yt, false)
			}
		} else {
			add("yt-dlp", "yt-dlp", true, "Found on PATH", false)
		}
		if ferr := appconfig.ValidateFFmpegForYouTube(Config.FFmpegPath); ferr != nil {
			add("ffmpeg", "ffmpeg + ffprobe", false, ferr.Error(), true)
		} else {
			add("ffmpeg", "ffmpeg + ffprobe", true, "Ready for YouTube", false)
		}
	} else {
		mp4box := appconfig.MP4BoxPath()
		if _, lerr := exec.LookPath(mp4box); lerr != nil {
			if _, serr := os.Stat(mp4box); serr != nil {
				add("mp4box", "MP4Box", false, "Required for tagging — install GPAC or add to tools folder", true)
			} else {
				add("mp4box", "MP4Box", true, mp4box, false)
			}
		} else {
			add("mp4box", "MP4Box", true, "Found on PATH", false)
		}
		if opts.Quality == "aac" && len(strings.TrimSpace(Config.MediaUserToken)) <= 50 {
			add("token", "media-user-token", false, "Required for AAC downloads — add in Settings", true)
		} else if opts.Quality == "aac" {
			add("token", "media-user-token", true, "Configured", false)
		}
	}

	ready := true
	blocking := 0
	for _, c := range checks {
		if c.OK || !c.Blocking {
			continue
		}
		blocking++
		ready = false
	}
	summary := "Ready to download"
	if !ready {
		summary = fmt.Sprintf("%d issue(s) to fix before downloading", blocking)
	}
	return PreflightResult{Ready: ready, Summary: summary, Checks: checks}
}

func lastJobOutputPath(masterPath string) string {
	if masterPath != "" {
		return masterPath
	}
	if lastYouTubeOutput != "" {
		return lastYouTubeOutput
	}
	if len(AddedTracks) > 0 {
		return AddedTracks[len(AddedTracks)-1].Path
	}
	return ""
}

func noteDownloadOutput(path string) {
	if path != "" {
		lastYouTubeOutput = path
	}
}

// DefaultOutputFolder returns the best folder to open in the file manager when none is specified.
func (e *Engine) DefaultOutputFolder() string {
	if p := lastJobOutputPath(""); p != "" {
		if st, err := os.Stat(p); err == nil {
			if st.IsDir() {
				return p
			}
			return filepath.Dir(p)
		}
		if filepath.Ext(p) != "" {
			return filepath.Dir(p)
		}
		return p
	}
	cfg := e.GetConfig()
	if strings.TrimSpace(cfg.YouTubeOutputLocation) == "apple-music" {
		return strings.TrimSpace(cfg.AacSaveFolder)
	}
	if yt := strings.TrimSpace(cfg.YouTubeSaveFolder); yt != "" {
		return yt
	}
	return strings.TrimSpace(cfg.AacSaveFolder)
}
