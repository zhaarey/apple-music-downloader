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

func (a *App) PreflightDownloadJob(url string, quality string, youtubeSaveVideo bool, sourceMode string) engine.PreflightResult {
	opts := engine.RunOptions{
		URLs:             []string{url},
		Quality:          quality,
		YouTubeSaveVideo: youtubeSaveVideo,
	}
	if sourceMode == "youtube" || quality == "youtube" {
		opts.Quality = "youtube"
	}
	return a.eng.PreflightDownload(opts)
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
