package main

import (
	"main/internal/media"
)

func (a *App) ValidateIPhoneSync(path string) (media.SyncValidationResult, error) {
	return media.ValidateIPhoneSync(a.eng.GetConfig().FFmpegPath, path)
}

func (a *App) ValidateIPhoneSyncFolder(folder string) (media.FolderSyncValidationResult, error) {
	return media.ValidateIPhoneSyncFolder(a.eng.GetConfig().FFmpegPath, folder)
}

func (a *App) GetAppleMusicCacheInfo() media.AppleMusicCacheInfo {
	return media.GetAppleMusicCacheInfo()
}

func (a *App) ClearAppleMusicArtworkCache() media.CacheClearResult {
	return media.ClearAppleMusicArtworkCache()
}

func (a *App) ClearAppTempCache() media.CacheClearResult {
	return media.ClearAppTempCache()
}

func (a *App) ClearAllSyncCaches() media.CacheClearResult {
	return media.ClearAllSyncCaches()
}
