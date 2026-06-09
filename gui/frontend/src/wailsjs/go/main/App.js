// Stub bindings — regenerate with `wails generate module` from the gui directory.

export const GetSettings = () => window.go?.main?.App?.GetSettings?.() ?? Promise.resolve({})
export const SaveSettings = (cfg) => window.go?.main?.App?.SaveSettings?.(cfg) ?? Promise.resolve()
export const CheckDependencies = () => window.go?.main?.App?.CheckDependencies?.() ?? Promise.resolve([])
export const Search = (type, query, offset) => window.go?.main?.App?.Search?.(type, query, offset) ?? Promise.resolve({ hits: [], hasNext: false })
export const DetectURLType = (url) => window.go?.main?.App?.DetectURLType?.(url) ?? 'Unknown'
export const PreviewURL = (url) => window.go?.main?.App?.PreviewURL?.(url) ?? Promise.resolve({ error: 'Preview not available' })
export const StartDownloadJob = (url, quality, selectedTrackNums, childURLs, youtubeSaveVideo, youtubeMeta) =>
  window.go?.main?.App?.StartDownloadJob?.(url, quality, selectedTrackNums, childURLs, youtubeSaveVideo, youtubeMeta) ?? Promise.resolve()
export const StartDownload = (urls, quality, singleSong, selectTracks, allArtistAlbums) =>
  window.go?.main?.App?.StartDownload?.(urls, quality, singleSong, selectTracks, allArtistAlbums) ?? Promise.resolve()
export const CancelDownload = () => window.go?.main?.App?.CancelDownload?.() ?? Promise.resolve()
export const IsDownloading = () => window.go?.main?.App?.IsDownloading?.() ?? Promise.resolve(false)
export const PickFolder = () => window.go?.main?.App?.PickFolder?.() ?? Promise.resolve('')
export const OpenFolder = (path) => window.go?.main?.App?.OpenFolder?.(path) ?? Promise.resolve()
export const RevealInFolder = (path) => window.go?.main?.App?.RevealInFolder?.(path) ?? Promise.resolve()
export const OpenConfigFolder = () => window.go?.main?.App?.OpenConfigFolder?.() ?? Promise.resolve()
export const PreflightDownloadJob = (url, quality, youtubeSaveVideo, sourceMode) =>
  window.go?.main?.App?.PreflightDownloadJob?.(url, quality, youtubeSaveVideo, sourceMode) ?? Promise.resolve({ ready: true, summary: '', checks: [] })
export const GetRecentFiles = () => window.go?.main?.App?.GetRecentFiles?.() ?? Promise.resolve([])
export const GetSetupComplete = () => window.go?.main?.App?.GetSetupComplete?.() ?? Promise.resolve(false)
export const SetSetupComplete = (v) => window.go?.main?.App?.SetSetupComplete?.(v) ?? Promise.resolve()
export const GetWizardComplete = () => window.go?.main?.App?.GetWizardComplete?.() ?? Promise.resolve(false)
export const SetWizardComplete = (v) => window.go?.main?.App?.SetWizardComplete?.(v) ?? Promise.resolve()
export const GetConfigPath = () => window.go?.main?.App?.GetConfigPath?.() ?? Promise.resolve('')
export const GetPlatform = () => window.go?.main?.App?.GetPlatform?.() ?? Promise.resolve('windows')
export const GetAppDataDir = () => window.go?.main?.App?.GetAppDataDir?.() ?? Promise.resolve('')
export const GetLogPath = () => window.go?.main?.App?.GetLogPath?.() ?? Promise.resolve('')
export const OpenLogFile = () => window.go?.main?.App?.OpenLogFile?.() ?? Promise.resolve()
export const LogFrontendError = (source, message, detail) =>
  window.go?.main?.App?.LogFrontendError?.(source, message, detail) ?? Promise.resolve()

export const SpliceProbeMaster = (path) => window.go?.main?.App?.SpliceProbeMaster?.(path) ?? Promise.resolve({})
export const SpliceLoadProject = (path) => window.go?.main?.App?.SpliceLoadProject?.(path) ?? Promise.resolve({})
export const SpliceSaveProject = (project) => window.go?.main?.App?.SpliceSaveProject?.(project) ?? Promise.resolve('')
export const SpliceComputeTimings = (project) => window.go?.main?.App?.SpliceComputeTimings?.(project) ?? Promise.resolve([])
export const SpliceSetBoundary = (project, boundaryIndex, positionMs) =>
  window.go?.main?.App?.SpliceSetBoundary?.(project, boundaryIndex, positionMs) ?? Promise.resolve(project)
export const SpliceSetTrackStart = (project, row, startMs) =>
  window.go?.main?.App?.SpliceSetTrackStart?.(project, row, startMs) ?? Promise.resolve(project)
export const SpliceSetTrackDuration = (project, row, durationMs) =>
  window.go?.main?.App?.SpliceSetTrackDuration?.(project, row, durationMs) ?? Promise.resolve(project)
export const SpliceDistributeDrift = (project) => window.go?.main?.App?.SpliceDistributeDrift?.(project) ?? Promise.resolve(project)
export const SpliceGetPeaks = (path, binCount) => window.go?.main?.App?.SpliceGetPeaks?.(path, binCount) ?? Promise.resolve({ bins: [] })
export const SpliceStartExport = (project) => window.go?.main?.App?.SpliceStartExport?.(project) ?? Promise.resolve()
export const SpliceCancelExport = () => window.go?.main?.App?.SpliceCancelExport?.() ?? Promise.resolve()
export const SpliceIsExporting = () => window.go?.main?.App?.SpliceIsExporting?.() ?? Promise.resolve(false)
export const SplicePickMasterFile = () => window.go?.main?.App?.SplicePickMasterFile?.() ?? Promise.resolve('')
export const SplicePickArtwork = () => window.go?.main?.App?.SplicePickArtwork?.() ?? Promise.resolve('')
export const SplicePickOutputDir = () => window.go?.main?.App?.SplicePickOutputDir?.() ?? Promise.resolve('')
export const SpliceMasterAudioURL = (path) =>
  window.go?.main?.App?.SpliceMasterAudioURL?.(path) ?? Promise.resolve('')

export const TagPickAudioFile = () => window.go?.main?.App?.TagPickAudioFile?.() ?? Promise.resolve('')
export const TagPickArtworkFile = () => window.go?.main?.App?.TagPickArtworkFile?.() ?? Promise.resolve('')
export const TagResolveDrop = (paths) =>
  window.go?.main?.App?.TagResolveDrop?.(paths) ?? Promise.reject(new Error('TagResolveDrop unavailable'))
export const TagReadFile = (path) => window.go?.main?.App?.TagReadFile?.(path) ?? Promise.resolve({})
export const TagWriteFile = (input) => window.go?.main?.App?.TagWriteFile?.(input) ?? Promise.resolve({})
export const TagReadAlbumFolder = (folder) =>
  window.go?.main?.App?.TagReadAlbumFolder?.(folder) ?? Promise.resolve([])
export const TagWriteAlbumBatch = (input) =>
  window.go?.main?.App?.TagWriteAlbumBatch?.(input) ??
  Promise.resolve({ saved: 0, errors: [], summary: 'Unavailable' })
export const TagLocalFileURL = (path) => window.go?.main?.App?.TagLocalFileURL?.(path) ?? Promise.resolve('')

export const ValidateIPhoneSync = (path) => window.go?.main?.App?.ValidateIPhoneSync?.(path) ?? Promise.resolve({ ready: false, summary: 'Unavailable', checks: [] })
export const ValidateIPhoneSyncFolder = (folder) =>
  window.go?.main?.App?.ValidateIPhoneSyncFolder?.(folder) ?? Promise.resolve({ ready: false, summary: 'Unavailable', files: [] })
export const GetAppleMusicCacheInfo = () => window.go?.main?.App?.GetAppleMusicCacheInfo?.() ?? Promise.resolve({ paths: [], note: '' })
export const ClearAppleMusicArtworkCache = () =>
  window.go?.main?.App?.ClearAppleMusicArtworkCache?.() ?? Promise.resolve({ ok: false, message: 'Unavailable' })
export const ClearAppTempCache = () => window.go?.main?.App?.ClearAppTempCache?.() ?? Promise.resolve({ ok: false, message: 'Unavailable' })
export const ClearAllSyncCaches = () => window.go?.main?.App?.ClearAllSyncCaches?.() ?? Promise.resolve({ ok: false, message: 'Unavailable' })

export const PrepareAlbumForSync = (folder) =>
  window.go?.main?.App?.PrepareAlbumForSync?.(folder) ?? Promise.resolve({ folder, prepared: 0, skipped: 0, errors: [], summary: 'Unavailable' })
export const PreviewPrepareAlbumForSync = (folder) =>
  window.go?.main?.App?.PreviewPrepareAlbumForSync?.(folder) ??
  Promise.resolve({ folder, track_count: 0, cover_source: '', recursive: false, warning: '' })
export const GetSyncRepairPreparePreview = () =>
  window.go?.main?.App?.GetSyncRepairPreparePreview?.() ??
  Promise.resolve({ folders: [], track_count: 0, warning: '' })
export const RunSyncRepair = (opts) =>
  window.go?.main?.App?.RunSyncRepair?.(opts) ?? Promise.resolve({ ok: false, summary: 'Unavailable', steps: [], manual_steps: [] })
export const RunSyncRepairElevated = () =>
  window.go?.main?.App?.RunSyncRepairElevated?.() ?? Promise.resolve({ ok: false, summary: 'Unavailable', steps: [], manual_steps: [] })
export const IsAppleMusicRunning = () => window.go?.main?.App?.IsAppleMusicRunning?.() ?? Promise.resolve(false)
export const OpenSyncRepairLog = () => window.go?.main?.App?.OpenSyncRepairLog?.() ?? Promise.resolve()

export const EventsOn = (eventName, callback) => {
  if (window.runtime?.EventsOn) {
    return window.runtime.EventsOn(eventName, callback)
  }
  return () => {}
}

export const EventsOff = (eventName, ...args) => {
  window.runtime?.EventsOff?.(eventName, ...args)
}
