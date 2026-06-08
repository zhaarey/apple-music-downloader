package main

import (
	"os"
	"path/filepath"

	appconfig "main/internal/config"
	"main/internal/splice"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SpliceProbeMaster(path string) (splice.MasterProbe, error) {
	cfg := a.eng.GetConfig()
	return splice.ProbeMaster(cfg.FFmpegPath, path)
}

func (a *App) SpliceLoadProject(path string) (splice.Project, error) {
	return splice.LoadProject(path)
}

func (a *App) SpliceSaveProject(project splice.Project) (string, error) {
	dir := appconfig.SpliceProjectsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := "project.json"
	if project.Album.Album != "" {
		name = splice.SanitizeFilename(project.Album.Album) + ".json"
	}
	out := filepath.Join(dir, name)
	return out, splice.SaveProject(project, out)
}

func (a *App) SpliceComputeTimings(project splice.Project) [][3]int {
	return a.spliceService().ComputeTimings(project)
}

func (a *App) SpliceSetBoundary(project splice.Project, boundaryIndex, positionMs int) splice.Project {
	return a.spliceService().SetBoundary(project, boundaryIndex, positionMs)
}

func (a *App) SpliceSetTrackStart(project splice.Project, row, startMs int) splice.Project {
	return a.spliceService().SetTrackStartProject(project, row, startMs)
}

func (a *App) SpliceSetTrackDuration(project splice.Project, row, durationMs int) splice.Project {
	return a.spliceService().SetTrackDurationProject(project, row, durationMs)
}

func (a *App) SpliceDistributeDrift(project splice.Project) splice.Project {
	return a.spliceService().DistributeDriftProject(project)
}

func (a *App) SpliceGetPeaks(path string, binCount int) (splice.WaveformPeaks, error) {
	cfg := a.eng.GetConfig()
	return splice.ExtractPeaks(cfg.FFmpegPath, path, binCount)
}

func (a *App) SpliceStartExport(project splice.Project) error {
	return a.spliceService().StartExport(project)
}

func (a *App) SpliceCancelExport() {
	a.spliceService().CancelExport()
}

func (a *App) SpliceIsExporting() bool {
	return a.spliceService().IsExporting()
}

func (a *App) SplicePickMasterFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select master audio file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Audio", Pattern: "*.m4a;*.mp3;*.wav;*.flac;*.aac"},
		},
	})
}

func (a *App) SplicePickArtwork() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select artwork",
		Filters: []runtime.FileFilter{
			{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png"},
		},
	})
}

func (a *App) SplicePickOutputDir() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select output folder"})
}

// SpliceMasterAudioURL returns an asset-server URL for local master audio playback.
func (a *App) SpliceMasterAudioURL(path string) string {
	return spliceMediaURL(path)
}

func (a *App) spliceService() *splice.Service {
	cfg := a.eng.GetConfig()
	if a.splice == nil {
		a.splice = splice.NewService(cfg, a.emitSpliceEvent)
	} else {
		a.splice.SetConfig(cfg)
	}
	return a.splice
}
