package main

import (
	"path/filepath"
	"strings"

	"main/internal/splice"
	"main/internal/trim"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) TrimPickSourceFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select audio or video to trim",
		Filters: []runtime.FileFilter{
			{DisplayName: "Media", Pattern: "*.m4a;*.mp4"},
		},
	})
}

func (a *App) TrimProbeFile(path string) (trim.ProbeResult, error) {
	cfg := a.eng.GetConfig()
	return trim.Probe(cfg.FFmpegPath, path)
}

func (a *App) TrimGetPeaks(path string, binCount int) (splice.WaveformPeaks, error) {
	cfg := a.eng.GetConfig()
	return splice.ExtractPeaks(cfg.FFmpegPath, path, binCount)
}

func (a *App) TrimMediaURL(path string) string {
	return localMediaURL(path)
}

func (a *App) TrimDefaultOutputPath(sourcePath string) string {
	return trim.DefaultOutputPath(sourcePath)
}

func (a *App) TrimStartExport(input trim.ExportInput) error {
	return a.trimService().StartExport(input)
}

func (a *App) TrimCancelExport() {
	a.trimService().CancelExport()
}

func (a *App) TrimIsExporting() bool {
	return a.trimService().IsExporting()
}

func (a *App) trimService() *trim.Service {
	cfg := a.eng.GetConfig()
	if a.trim == nil {
		a.trim = trim.NewService(cfg, a.emitTrimEvent)
	} else {
		a.trim.SetConfig(cfg)
	}
	return a.trim
}

// TrimSuggestPath validates a handoff path for the trim tab.
func (a *App) TrimSuggestPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".m4a" && ext != ".mp4" {
		return ""
	}
	return path
}
