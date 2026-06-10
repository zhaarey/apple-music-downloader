package main

import (
	"main/internal/appstate"
	"main/internal/engine"
	"main/internal/osutil"

	appconfig "main/internal/config"
)

func (a *App) OpenConfigFolder() error {
	return osutil.RevealInFileManager(appconfig.AppDataDir())
}

func (a *App) PreflightDownloadJob(url string, quality string, youtubeDeliveryMode string, sourceMode string) engine.PreflightResult {
	opts := engine.RunOptions{
		URLs:                []string{url},
		Quality:             quality,
		YouTubeDeliveryMode: youtubeDeliveryMode,
	}
	if sourceMode == "youtube" || quality == "youtube" {
		opts.Quality = "youtube"
	}
	return a.eng.PreflightDownload(opts)
}

func (a *App) CheckTrackDuplicates(
	url string,
	quality string,
	selectedTrackNums []int,
	youtubeDeliveryMode string,
	sourceMode string,
	youtubeMeta []engine.YouTubeDownloadMeta,
	preview engine.PreviewResult,
) engine.DuplicateCheckResult {
	opts := engine.RunOptions{
		URLs:                []string{url},
		Quality:             quality,
		SelectedTrackNums:   selectedTrackNums,
		YouTubeDeliveryMode: youtubeDeliveryMode,
		YouTubeMeta:         youtubeMeta,
	}
	if sourceMode == "youtube" || quality == "youtube" {
		opts.Quality = "youtube"
	}
	return a.eng.CheckTrackDuplicates(opts, preview, youtubeMeta)
}

func (a *App) GetRecentFiles() []string {
	return appstate.GetRecentFiles()
}

func (a *App) GetSetupComplete() bool {
	return appstate.GetSetupComplete()
}

func (a *App) SetSetupComplete(complete bool) error {
	return appstate.SetSetupComplete(complete)
}
