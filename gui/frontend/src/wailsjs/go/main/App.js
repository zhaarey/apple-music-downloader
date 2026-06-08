// Stub bindings — regenerate with `wails generate module` from the gui directory.

export const GetSettings = () => window.go?.main?.App?.GetSettings?.() ?? Promise.resolve({})
export const SaveSettings = (cfg) => window.go?.main?.App?.SaveSettings?.(cfg) ?? Promise.resolve()
export const CheckDependencies = () => window.go?.main?.App?.CheckDependencies?.() ?? Promise.resolve([])
export const Search = (type, query, offset) => window.go?.main?.App?.Search?.(type, query, offset) ?? Promise.resolve({ hits: [], hasNext: false })
export const DetectURLType = (url) => window.go?.main?.App?.DetectURLType?.(url) ?? 'Unknown'
export const PreviewURL = (url) => window.go?.main?.App?.PreviewURL?.(url) ?? Promise.resolve({ error: 'Preview not available' })
export const StartDownloadJob = (url, quality, selectedTrackNums, childURLs) =>
  window.go?.main?.App?.StartDownloadJob?.(url, quality, selectedTrackNums, childURLs) ?? Promise.resolve()
export const StartDownload = (urls, quality, singleSong, selectTracks, allArtistAlbums) =>
  window.go?.main?.App?.StartDownload?.(urls, quality, singleSong, selectTracks, allArtistAlbums) ?? Promise.resolve()
export const CancelDownload = () => window.go?.main?.App?.CancelDownload?.() ?? Promise.resolve()
export const IsDownloading = () => window.go?.main?.App?.IsDownloading?.() ?? Promise.resolve(false)
export const PickFolder = () => window.go?.main?.App?.PickFolder?.() ?? Promise.resolve('')
export const OpenFolder = (path) => window.go?.main?.App?.OpenFolder?.(path) ?? Promise.resolve()
export const GetWizardComplete = () => window.go?.main?.App?.GetWizardComplete?.() ?? Promise.resolve(false)
export const SetWizardComplete = (v) => window.go?.main?.App?.SetWizardComplete?.(v) ?? Promise.resolve()
export const GetConfigPath = () => window.go?.main?.App?.GetConfigPath?.() ?? Promise.resolve('')
export const GetLogPath = () => window.go?.main?.App?.GetLogPath?.() ?? Promise.resolve('')
export const OpenLogFile = () => window.go?.main?.App?.OpenLogFile?.() ?? Promise.resolve()

export const EventsOn = (eventName, callback) => {
  if (window.runtime?.EventsOn) {
    return window.runtime.EventsOn(eventName, callback)
  }
  return () => {}
}

export const EventsOff = (eventName, ...args) => {
  window.runtime?.EventsOff?.(eventName, ...args)
}
