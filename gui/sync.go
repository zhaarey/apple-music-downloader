package main

import (
	"os"
	"path/filepath"
	"strings"

	"main/internal/media"
	"main/internal/osutil"
	"main/internal/platform"
	"main/utils/structs"
)

func (a *App) ValidateIPhoneSync(path string) (media.SyncValidationResult, error) {
	return media.ValidateIPhoneSync(a.eng.GetConfig().FFmpegPath, path)
}

func (a *App) ValidateIPhoneSyncFolder(folder string) (media.FolderSyncValidationResult, error) {
	return media.ValidateIPhoneSyncFolder(a.eng.GetConfig().FFmpegPath, folder)
}

func (a *App) PreviewPrepareAlbumForSync(folder string) (media.AlbumPreparePreview, error) {
	return media.PreviewPrepareAlbum(folder, false)
}

func (a *App) PrepareAlbumForSync(folder string) (media.AlbumPrepareResult, error) {
	return media.PrepareAlbumForSync(a.eng.GetConfig().FFmpegPath, folder, false)
}

func (a *App) GetSyncRepairPreparePreview() (media.SyncRepairPreparePreview, error) {
	folders := librarySyncFolders(a.eng.GetConfig())
	out := media.SyncRepairPreparePreview{
		Folders: folders,
		Warning: "Updates embedded artwork only — titles and track numbers are preserved. Skips tracks that already match. Then clears PC artwork caches (not iPhone).",
	}
	for _, folder := range folders {
		paths, err := media.CollectAlbumTracks(folder)
		if err != nil {
			return out, err
		}
		out.TrackCount += len(paths)
	}
	return out, nil
}

func (a *App) PrepareLibraryForSync() ([]media.AlbumPrepareResult, error) {
	cfg := a.eng.GetConfig()
	folders := librarySyncFolders(cfg)
	out := make([]media.AlbumPrepareResult, 0, len(folders))
	for _, folder := range folders {
		prep, err := media.PrepareAlbumForSync(cfg.FFmpegPath, folder, true)
		if err != nil {
			return out, err
		}
		out = append(out, prep)
	}
	return out, nil
}

func librarySyncFolders(cfg structs.ConfigSet) []string {
	seen := map[string]struct{}{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if _, ok := seen[p]; ok {
			return
		}
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			return
		}
		seen[p] = struct{}{}
	}
	add(cfg.AacSaveFolder)
	add(cfg.AlacSaveFolder)
	add(cfg.AtmosSaveFolder)
	add(cfg.YouTubeSaveFolder)
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	return out
}

func (a *App) RunSyncRepair(opts media.SyncRepairOptions) media.SyncRepairResult {
	if len(opts.PrepareFolders) == 0 && !opts.SkipPrepare && !opts.CacheOnly {
		opts.PrepareFolders = librarySyncFolders(a.eng.GetConfig())
	}
	return media.RunSyncRepair(opts)
}

func (a *App) RunSyncRepairElevated() (media.SyncRepairResult, error) {
	res := media.SyncRepairResult{
		ManualSteps: media.RunSyncRepair(media.SyncRepairOptions{SkipPrepare: true, CacheOnly: true}).ManualSteps,
		LogPath:     platform.SyncRepairLogPath(),
	}
	ok, message, err := osutil.RunElevatedCacheClear()
	step := media.SyncRepairStep{
		ID:    "elevated_cache",
		Label: "Clear Apple Music artwork cache (administrator)",
		OK:    ok && err == nil,
		Detail: message,
	}
	if err != nil {
		step.Detail = err.Error() + ": " + message
	}
	res.Steps = append(res.Steps, step)
	res.OK = step.OK
	if res.OK {
		res.Summary = "Administrator cache clear completed — re-import albums and re-sync your iPhone."
	} else {
		res.Summary = "Administrator cache clear failed or was cancelled."
	}
	return res, nil
}

func (a *App) IsAppleMusicRunning() bool {
	return platform.IsAppleMusicRunning()
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

func (a *App) OpenSyncRepairLog() error {
	return osutil.RevealInFileManager(platform.SyncRepairLogPath())
}

func (a *App) RunAppleMusicDeepPurge(elevated bool) media.ApplePurgeResult {
	return media.RunAppleMusicDeepPurge(elevated)
}

func (a *App) OpenApplePurgeLog() error {
	return osutil.RevealInFileManager(platform.ApplePurgeLogPath())
}
