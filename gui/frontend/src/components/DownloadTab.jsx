import { useEffect, useMemo, useState } from 'react'
import { DetectURLType, PreviewURL, StartDownloadJob, OpenLogFile, OpenFolder } from '../wailsjs/go/main/App'
import { parseJobResult, parseTrackRows, parseYouTubeProgress, jobStatusMeta, trackStatusIcon, trackStatusClass } from '../lib/downloadStatus'
import { YouTubeFetchSkeleton, YouTubeProgressPanel } from './YouTubeUI'
import YouTubeMetadataEditor, { buildMetaFromPreview, metaPayload } from './YouTubeMetadataEditor'
import SearchTab from './SearchTab'

const QUALITIES = [
  { id: 'aac', label: 'AAC', desc: 'Works immediately', needsWrapper: false },
  { id: 'alac', label: 'Lossless', desc: 'Requires wrapper', needsWrapper: true },
  { id: 'atmos', label: 'Dolby Atmos', desc: 'Requires wrapper', needsWrapper: true },
]

function outputFolderForQuality(settings, quality, youtubeMode) {
  if (youtubeMode) {
    return settings?.['youtube-save-folder'] || settings?.['aac-save-folder'] || ''
  }
  if (quality === 'alac') return settings?.['alac-save-folder'] || settings?.['aac-save-folder'] || ''
  if (quality === 'atmos') return settings?.['atmos-save-folder'] || settings?.['aac-save-folder'] || ''
  return settings?.['aac-save-folder'] || ''
}

function JobStatusBanner({ jobResult, onOpenFolder, onOpenLog, onSplitIntoTracks }) {
  if (!jobResult) return null
  const meta = jobStatusMeta(jobResult.phase)
  const canSplit = Boolean(jobResult.handoff?.master_path || jobResult.masterPath)
  return (
    <div className={`rounded-xl border px-4 py-3 ${meta.className}`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="font-semibold">{meta.label}</p>
          <p className="mt-1 text-sm opacity-90">{jobResult.message}</p>
          {jobResult.total > 0 && (
            <p className="mt-2 text-xs opacity-80">
              {jobResult.success} succeeded · {jobResult.failed} failed/unavailable · {jobResult.total} attempted
            </p>
          )}
          {jobResult.recentErrors?.length > 0 && (
            <ul className="mt-2 space-y-1 text-xs opacity-90">
              {jobResult.recentErrors.map((msg, i) => (
                <li key={i}>• {msg}</li>
              ))}
            </ul>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {canSplit && onSplitIntoTracks && (
            <button type="button" onClick={onSplitIntoTracks} className="rounded-lg bg-accent/90 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent">
              Split into tracks
            </button>
          )}
          <button type="button" onClick={onOpenFolder} className="rounded-lg bg-black/20 px-3 py-1.5 text-xs hover:bg-black/30">
            Open folder
          </button>
          {(jobResult.phase === 'failed' || jobResult.phase === 'partial') && (
            <button type="button" onClick={onOpenLog} className="rounded-lg bg-black/20 px-3 py-1.5 text-xs hover:bg-black/30">
              View log
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

export default function DownloadTab({
  settings,
  deps,
  prefillUrl,
  onPrefillConsumed,
  downloading,
  onDownloadStart,
  onDownloadEnd,
  engineEvents,
  jobSession,
  onClearJobSession,
  onResetPipeline,
  onSettingsChange,
  onSplitIntoTracks,
  sourceMode,
}) {
  const [url, setUrl] = useState('')
  const [urlType, setUrlType] = useState('')
  const [quality, setQuality] = useState('aac')
  const [preview, setPreview] = useState(null)
  const [selected, setSelected] = useState(new Set())
  const [fetching, setFetching] = useState(false)
  const [fetchError, setFetchError] = useState('')
  const [jobStarted, setJobStarted] = useState(false)
  const [saveVideo, setSaveVideo] = useState(false)
  const [fetchStep, setFetchStep] = useState(0)
  const [metaByTrack, setMetaByTrack] = useState({})

  const youtubeMode = sourceMode ? sourceMode === 'youtube' : Boolean(settings?.['youtube-mode'])
  const showSourceToggle = !sourceMode
  const embedSearch = sourceMode === 'apple'
  const ytDlpOk = deps?.some((d) => d.name === 'yt-dlp' && d.ok)
  const ffmpegOk = deps?.some((d) => d.name === 'ffmpeg' && d.ok)
  const wrapperOk = deps?.some((d) => d.name?.includes('wrapper') && d.ok)
  const hasToken = (settings?.['media-user-token'] || '').length > 50

  const previewTracks = useMemo(() => {
    if (!preview?.tracks?.length) return []
    return preview.tracks.filter((t) => selected.has(t.num))
  }, [preview, selected])

  const trackRows = useMemo(
    () => parseTrackRows(previewTracks, engineEvents, jobStarted),
    [previewTracks, engineEvents, jobStarted],
  )

  const jobResult = useMemo(() => {
    if (downloading || !jobStarted) return null
    return parseJobResult(engineEvents)
  }, [engineEvents, downloading, jobStarted])

  const youtubeProgress = useMemo(
    () => (youtubeMode ? parseYouTubeProgress(engineEvents) : null),
    [engineEvents, youtubeMode],
  )

  useEffect(() => {
    if (prefillUrl && !youtubeMode) {
      setUrl(prefillUrl)
      onPrefillConsumed()
    }
  }, [prefillUrl, onPrefillConsumed, youtubeMode])

  useEffect(() => {
    if (url.length > 12) {
      DetectURLType(url).then(setUrlType)
    } else {
      setUrlType('')
    }
  }, [url, youtubeMode])

  useEffect(() => {
    if (!fetching || !youtubeMode) {
      setFetchStep(0)
      return undefined
    }
    let step = 0
    const timer = setInterval(() => {
      step += 1
      setFetchStep(step)
    }, 1400)
    return () => clearInterval(timer)
  }, [fetching, youtubeMode])

  useEffect(() => {
    if (!downloading && jobStarted) {
      onDownloadEnd?.(parseJobResult(engineEvents))
    }
  }, [downloading, jobStarted, engineEvents, onDownloadEnd])

  const resetPreview = ({ clearPipeline = false } = {}) => {
    setPreview(null)
    setSelected(new Set())
    setFetchError('')
    setJobStarted(false)
    setMetaByTrack({})
    if (clearPipeline) {
      onResetPipeline?.()
    } else {
      onClearJobSession?.()
    }
  }

  const setMode = async (nextYouTube) => {
    resetPreview({ clearPipeline: true })
    setUrl('')
    setUrlType('')
    await onSettingsChange?.({ 'youtube-mode': nextYouTube })
  }

  const fetchPreview = async (forcedUrl) => {
    const trimmed = (forcedUrl ?? url).trim()
    if (!trimmed) return
    if (forcedUrl) setUrl(trimmed)
    setFetching(true)
    setFetchError('')
    setPreview(null)
    setSelected(new Set())
    setJobStarted(false)
    onResetPipeline?.()
    try {
      const res = await PreviewURL(trimmed)
      if (res.error) {
        setFetchError(res.error)
        return
      }
      setPreview(res)
      setSelected(new Set(res.tracks?.map((t) => t.num) || [1]))
      setMetaByTrack(buildMetaFromPreview(res))
    } finally {
      setFetching(false)
    }
  }

  const toggleTrack = (num) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(num)) next.delete(num)
      else next.add(num)
      return next
    })
  }

  const startDownload = async () => {
    if (!preview || downloading) return
    if (!youtubeMode && quality === 'aac' && !hasToken) {
      setFetchError('AAC downloads require media-user-token in Settings')
      return
    }
    if (youtubeMode && (!ytDlpOk || !ffmpegOk || deps?.some((d) => d.name === 'ffprobe (YouTube)' && !d.ok))) {
      setFetchError('YouTube mode requires yt-dlp, ffmpeg, and ffprobe in the same folder — check Requirements tab')
      return
    }

    const isArtist = preview.type === 'Artist'
    let selectedTrackNums = []
    let childURLs = []

    if (isArtist) {
      childURLs = (preview.tracks || []).filter((t) => selected.has(t.num) && t.url).map((t) => t.url)
      if (childURLs.length === 0) {
        setFetchError('Select at least one album or music video')
        return
      }
    } else if (preview.can_select_tracks) {
      selectedTrackNums = [...selected].sort((a, b) => a - b)
      if (selectedTrackNums.length === 0) {
        setFetchError('Select at least one track')
        return
      }
    }

    setFetchError('')
    setJobStarted(true)
    onDownloadStart?.()
    try {
      const youtubeMeta = youtubeMode ? metaPayload(metaByTrack, selected) : []
      await StartDownloadJob(preview.url, youtubeMode ? 'youtube' : quality, selectedTrackNums, childURLs, saveVideo, youtubeMeta)
    } catch (err) {
      setJobStarted(false)
      const msg = typeof err === 'string' ? err : err?.message || String(err)
      setFetchError(msg)
      onDownloadEnd?.(null)
    }
  }

  const selectedCount = selected.size
  const showProgress = jobStarted && (trackRows.length > 0 || (youtubeMode && downloading))
  const urlUnknown = urlType === 'Unknown' && url.trim().length > 12 && !youtubeMode

  const handleSearchSelect = (searchUrl) => {
    setUrl(searchUrl)
    resetPreview({ clearPipeline: true })
    fetchPreview(searchUrl)
  }

  return (
    <div className="mx-auto flex h-full max-w-3xl flex-col gap-4 overflow-y-auto pb-4">
      {embedSearch && !preview && (
        <SearchTab embedded onPreview={handleSearchSelect} />
      )}

      {showSourceToggle && (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-white/10 bg-surface-raised p-3">
        <div>
          <p className="text-sm font-medium">Source</p>
          <p className="text-xs text-white/50">
            {youtubeMode ? 'Download audio from YouTube links (DJ sets, mixes)' : 'Download from Apple Music'}
          </p>
        </div>
        <div className="flex rounded-lg bg-surface p-1">
          <button
            type="button"
            disabled={downloading}
            onClick={() => setMode(false)}
            className={`rounded-md px-4 py-2 text-sm font-medium transition ${
              !youtubeMode ? 'bg-accent text-white' : 'text-white/60 hover:text-white'
            }`}
          >
            Apple Music
          </button>
          <button
            type="button"
            disabled={downloading}
            onClick={() => setMode(true)}
            className={`rounded-md px-4 py-2 text-sm font-medium transition ${
              youtubeMode ? 'bg-accent text-white' : 'text-white/60 hover:text-white'
            }`}
          >
            YouTube Audio
          </button>
        </div>
      </div>
      )}

      {!preview && (
        <>
          <div>
            <h2 className="text-xl font-semibold">
              {youtubeMode ? 'Download from YouTube' : 'Download from Apple Music'}
            </h2>
            <p className="mt-1 text-sm text-white/50">
              {youtubeMode
                ? 'Paste a video or playlist URL — audio is saved in the best available quality'
                : embedSearch
                  ? 'Paste a link below or pick from search results — fetch to preview, then download'
                  : 'Paste a link, fetch to preview, then download — success and errors show here when finished'}
            </p>
          </div>

          <div className="flex gap-2">
            <div className="relative flex-1">
              <input
                value={url}
                onChange={(e) => {
                  setUrl(e.target.value)
                  resetPreview()
                }}
                onKeyDown={(e) => e.key === 'Enter' && fetchPreview()}
                placeholder={
                  youtubeMode
                    ? 'https://www.youtube.com/watch?v=... or playlist link'
                    : 'https://music.apple.com/us/playlist/...'
                }
                className="w-full rounded-xl border border-white/10 bg-surface-raised px-4 py-3 pr-24 text-sm focus:border-accent focus:outline-none"
              />
              {urlType && (
                <span className="absolute right-3 top-1/2 -translate-y-1/2 rounded-full bg-accent/20 px-2 py-0.5 text-xs text-accent">
                  {urlType}
                </span>
              )}
            </div>
            <button
              type="button"
              onClick={fetchPreview}
              disabled={!url.trim() || fetching || urlUnknown}
              className="rounded-xl bg-accent px-5 py-3 font-medium hover:bg-accent-muted disabled:opacity-40"
            >
              {fetching ? 'Fetching…' : 'Fetch'}
            </button>
          </div>

          {fetchError && <p className="text-sm text-red-400">{fetchError}</p>}

          {fetching && youtubeMode && <YouTubeFetchSkeleton step={fetchStep} />}

          {!fetching && youtubeMode ? (
            <div className="space-y-2">
              {(!ytDlpOk || !ffmpegOk || deps?.some((d) => d.name === 'ffprobe (YouTube)' && !d.ok)) && (
                <p className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-200">
                  Install <strong>yt-dlp</strong>, <strong>ffmpeg</strong>, and <strong>ffprobe</strong> (essentials build) on PATH or in <code>dist/tools/</code> — see Requirements tab.
                </p>
              )}
              <p className="rounded-lg border border-white/10 bg-white/[0.02] px-3 py-2 text-sm text-white/60">
                Long DJ sets and live streams are supported. No Apple Music account needed.
              </p>
            </div>
          ) : (
            !hasToken && (
              <p className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-200">
                Add your Apple Music <code className="text-accent">media-user-token</code> in Settings before downloading AAC.
              </p>
            )
          )}
        </>
      )}

      {preview && (
        <>
          <div className="flex items-start justify-between gap-4">
            <button
              type="button"
              onClick={() => resetPreview({ clearPipeline: true })}
              className="text-sm text-white/50 hover:text-white"
              disabled={downloading}
            >
              ← Change URL
            </button>
            {downloading && (
              <span className="rounded-full bg-accent/20 px-3 py-1 text-xs text-accent animate-pulse">Downloading…</span>
            )}
          </div>

          <JobStatusBanner
            jobResult={jobResult || jobSession}
            onOpenFolder={() => OpenFolder('')}
            onOpenLog={() => OpenLogFile()}
            onSplitIntoTracks={() => {
              const h = (jobResult || jobSession)?.handoff
              if (h?.master_path) onSplitIntoTracks?.(h)
            }}
          />

          <div className="flex gap-4 rounded-xl border border-white/10 bg-surface-raised p-4">
            {preview.art_url ? (
              <img src={preview.art_url} alt="" className="h-24 w-24 shrink-0 rounded-lg object-cover" />
            ) : (
              <div className="flex h-24 w-24 shrink-0 items-center justify-center rounded-lg bg-surface text-3xl">▶</div>
            )}
            <div className="min-w-0 flex-1">
              <p className="text-xs uppercase tracking-wide text-accent">{preview.type}</p>
              <h3 className="truncate text-lg font-semibold">{preview.title}</h3>
              <p className="truncate text-sm text-white/60">{preview.subtitle}</p>
              <p className="mt-1 text-xs text-white/40">
                {preview.track_count} item{preview.track_count !== 1 ? 's' : ''}
                {preview.total_duration ? ` · ${preview.total_duration}` : ''}
              </p>
            </div>
          </div>

          {!youtubeMode && (
            <div>
              <p className="mb-2 text-sm text-white/60">Quality</p>
              <div className="grid gap-2 sm:grid-cols-3">
                {QUALITIES.map((q) => {
                  const blocked = q.needsWrapper && !wrapperOk
                  return (
                    <button
                      key={q.id}
                      type="button"
                      disabled={blocked || downloading}
                      onClick={() => setQuality(q.id)}
                      className={`rounded-xl border p-3 text-left transition ${
                        quality === q.id
                          ? 'border-accent bg-accent/10'
                          : blocked
                            ? 'border-white/5 opacity-40'
                            : 'border-white/10 hover:border-white/20'
                      }`}
                    >
                      <div className="font-medium">{q.label}</div>
                      <div className="mt-1 text-xs text-white/50">{q.desc}</div>
                    </button>
                  )
                })}
              </div>
              {!wrapperOk && (quality === 'alac' || quality === 'atmos') && (
                <p className="mt-2 text-sm text-yellow-400">
                  Wrapper not running — check Requirements tab before using Lossless or Atmos.
                </p>
              )}
            </div>
          )}

          {youtubeMode && (
            <div className="space-y-3">
              <p className="rounded-lg border border-accent/20 bg-accent/5 px-3 py-2 text-sm text-white/70">
                Downloads convert to <strong>AAC 256 kbps</strong> with Apple Music tags · organized in Album folders
              </p>
              <YouTubeMetadataEditor
                preview={preview}
                selected={selected}
                metaByTrack={metaByTrack}
                disabled={downloading}
                onChange={(num, patch) =>
                  setMetaByTrack((prev) => ({
                    ...prev,
                    [num]: { ...prev[num], ...patch },
                  }))
                }
                onSharedChange={(patch) =>
                  setMetaByTrack((prev) => {
                    const next = { ...prev }
                    Object.keys(next).forEach((k) => {
                      if (selected.has(Number(k))) {
                        next[k] = { ...next[k], ...patch }
                      }
                    })
                    return next
                  })
                }
              />
              <label className="flex items-start gap-3 rounded-lg border border-white/10 bg-surface-raised px-3 py-3 text-sm">
                <input
                  type="checkbox"
                  checked={saveVideo}
                  onChange={(e) => setSaveVideo(e.target.checked)}
                  disabled={downloading}
                  className="mt-1 shrink-0"
                />
                <span>
                  <span className="font-medium text-white/90">Also save MP4 video copy</span>
                  <span className="mt-1 block text-xs text-white/50">
                    Creates a separate H.264 MP4 with AAC stereo audio (required for sound in the iPhone Music app).
                    Adds extra download and conversion time.
                  </span>
                </span>
              </label>
            </div>
          )}

          {preview.can_select_tracks && preview.tracks?.length > 0 && (
            <div className="rounded-xl border border-white/10 bg-surface-raised">
              <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
                <span className="text-sm font-medium">
                  {preview.type === 'Artist' ? 'Albums & videos' : preview.type === 'YouTube Playlist' ? 'Videos' : 'Tracks'}
                </span>
                <div className="flex gap-3 text-xs">
                  <button
                    type="button"
                    onClick={() => setSelected(new Set(preview.tracks.map((t) => t.num)))}
                    className="text-accent hover:underline"
                    disabled={downloading}
                  >
                    Select all
                  </button>
                  <button
                    type="button"
                    onClick={() => setSelected(new Set())}
                    className="text-white/50 hover:underline"
                    disabled={downloading}
                  >
                    Clear
                  </button>
                </div>
              </div>
              <ul className="max-h-64 divide-y divide-white/5 overflow-y-auto">
                {preview.tracks.map((t) => (
                  <li key={t.num} className="flex items-center gap-3 px-4 py-2.5 text-sm hover:bg-white/[0.02]">
                    <input
                      type="checkbox"
                      checked={selected.has(t.num)}
                      onChange={() => toggleTrack(t.num)}
                      disabled={downloading}
                      className="shrink-0"
                    />
                    {youtubeMode && t.art_url ? (
                      <img src={t.art_url} alt="" className="h-10 w-10 shrink-0 rounded object-cover" />
                    ) : (
                      <span className="w-6 shrink-0 text-right text-xs text-white/40">{t.num}</span>
                    )}
                    <div className="min-w-0 flex-1">
                      <p className="truncate">{t.name}</p>
                      <p className="truncate text-xs text-white/40">{t.artist}</p>
                    </div>
                    <span className="shrink-0 text-xs text-white/40">{t.duration}</span>
                    {t.explicit && <span className="shrink-0 text-xs text-white/50">E</span>}
                    {t.is_mv && <span className="shrink-0 text-xs text-white/50">MV</span>}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {youtubeMode && (downloading || jobStarted) && (
            <YouTubeProgressPanel progress={youtubeProgress} downloading={downloading} trackRows={trackRows} />
          )}

          {showProgress && (
            <div className="rounded-xl border border-white/10 bg-black/20 p-4">
              <div className="mb-2 flex items-center justify-between">
                <p className="text-sm font-medium">Track status</p>
                {!downloading && jobResult && (
                  <span className="text-xs text-white/40">
                    {trackRows.filter((r) => r.status === 'done' || r.status === 'skipped').length}/{trackRows.length} OK
                  </span>
                )}
              </div>
              <ul className="max-h-48 space-y-2 overflow-y-auto text-xs">
                {trackRows.map((r) => (
                  <li key={r.num} className="rounded-lg bg-white/[0.02] px-2 py-1.5">
                    <div className="flex items-start gap-2">
                      <span className={`mt-0.5 w-4 shrink-0 ${trackStatusClass(r.status)}`}>{trackStatusIcon(r.status)}</span>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-white/90">{r.label}</p>
                        {r.sublabel && <p className="truncate text-[11px] text-white/40">{r.sublabel}</p>}
                        {r.detail && (
                          <p className={`mt-0.5 ${trackStatusClass(r.status)}`}>{r.detail}</p>
                        )}
                        {youtubeMode && r.status === 'downloading' && r.percent > 0 && (
                          <div className="mt-1.5 h-1 overflow-hidden rounded-full bg-black/30">
                            <div className="h-full rounded-full bg-accent transition-all" style={{ width: `${r.percent}%` }} />
                          </div>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <p
            className="truncate text-xs text-white/40"
            title={outputFolderForQuality(settings, quality, youtubeMode)}
          >
            Output:{' '}
            {outputFolderForQuality(settings, quality, youtubeMode) || preview.output_folder || 'Default downloads folder'}
          </p>

          {fetchError && (
            <p className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">{fetchError}</p>
          )}

          <button
            type="button"
            onClick={startDownload}
            disabled={downloading || selectedCount === 0}
            className="rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
          >
            {downloading
              ? saveVideo
                ? 'Downloading audio + video…'
                : 'Downloading…'
              : jobResult
                ? 'Download again'
                : youtubeMode
                  ? saveVideo
                    ? `Download audio + MP4 (${selectedCount})`
                    : `Download audio (${selectedCount})`
                  : `Download ${selectedCount} selected`}
          </button>
        </>
      )}
    </div>
  )
}
