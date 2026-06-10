import { useEffect, useMemo, useState } from 'react'
import { DetectURLType, PreviewURL, StartDownloadJob, OpenLogFile, OpenFolder, RevealInFolder, PreflightDownloadJob, PickFolder, CheckTrackDuplicates } from '../wailsjs/go/main/App'
import {
  parseJobResult,
  parseTrackRows,
  parseYouTubeProgress,
  jobStatusMeta,
  deriveFlowPhase,
} from '../lib/downloadStatus'
import { FetchPreviewSkeleton, FetchStatusBanner } from './YouTubeUI'
import YouTubeMetadataEditor, { buildMetaFromPreview, metaPayload } from './YouTubeMetadataEditor'
import YouTubeDeliveryModeSwitch from './YouTubeDeliveryModeSwitch'
import { youtubeDownloadButtonLabel } from '../lib/youtubeDelivery'
import SearchTab from './SearchTab'
import OutputFolderField from './OutputFolderField'
import BulkDownloadQueue from './BulkDownloadQueue'
import DownloadFlowLayout from './download/DownloadFlowLayout'
import TrackStatusPanel from './download/TrackStatusPanel'
import AppleDownloadModeSwitch from './AppleDownloadModeSwitch'
import PageShell from './PageShell'
import { outputFolderKey, outputFolderLabel, outputFolderPath } from '../lib/settings'
import { reportFrontendError } from '../lib/errorReporting'
import { formatActionError } from '../lib/formatActionError'

const QUALITIES = [
  { id: 'aac', label: 'AAC', desc: 'Works immediately', needsWrapper: false },
  { id: 'alac', label: 'Lossless', desc: 'Requires wrapper', needsWrapper: true },
  { id: 'atmos', label: 'Dolby Atmos', desc: 'Requires wrapper', needsWrapper: true },
]

function isYouTubeURL(raw) {
  return /(?:youtube\.com|youtu\.be)/i.test(String(raw || '').trim())
}

function isAppleMusicURL(raw) {
  return /music\.apple\.com/i.test(String(raw || '').trim())
}

function urlTabMismatch(trimmed, youtubeMode) {
  if (!trimmed) return ''
  if (youtubeMode && isAppleMusicURL(trimmed)) {
    return 'This is an Apple Music link — switch to the Apple Music tab to fetch and download it.'
  }
  if (!youtubeMode && isYouTubeURL(trimmed)) {
    return 'This is a YouTube link — switch to the YouTube tab to fetch and download it.'
  }
  return ''
}

const APPLE_SUBSCRIPTION_MSG =
  'Apple Music downloads require an active subscription. Add your media-user-token in Settings (music.apple.com → DevTools → Application → Cookies).'

function formatAppleAuthError(message) {
  const msg = String(message || '')
  if (/media-user-token|active apple music subscription|authorization-token/i.test(msg)) {
    return APPLE_SUBSCRIPTION_MSG
  }
  return msg
}

function outputFolderForQuality(settings, quality, youtubeMode) {
  return outputFolderPath(settings, quality, youtubeMode)
}

async function saveOutputFolder(settings, onSettingsChange, quality, youtubeMode, path) {
  const trimmed = String(path || '').trim()
  if (!trimmed || !onSettingsChange) return
  const key = outputFolderKey(quality, youtubeMode)
  await onSettingsChange({ [key]: trimmed })
}

function JobStatusBanner({ jobResult, onRevealFile, onOpenFolder, onOpenLog, onSplitIntoTracks }) {
  if (!jobResult) return null
  const meta = jobStatusMeta(jobResult.phase)
  const canSplit = Boolean(jobResult.handoff?.master_path || jobResult.masterPath)
  const revealPath = jobResult.outputPath || jobResult.masterPath || jobResult.handoff?.master_path || ''
  return (
    <div className={`rounded-xl border px-4 py-3 ${meta.className}`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="font-semibold">{meta.label}</p>
          <p className="mt-1 text-sm opacity-90">{jobResult.message}</p>
          {revealPath && (
            <p className="mt-1 truncate text-xs opacity-70" title={revealPath}>
              {revealPath}
            </p>
          )}
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
          {revealPath ? (
            <button type="button" onClick={() => onRevealFile?.(revealPath)} className="rounded-lg bg-black/20 px-3 py-1.5 text-xs hover:bg-black/30">
              Reveal file
            </button>
          ) : (
            <button type="button" onClick={onOpenFolder} className="rounded-lg bg-black/20 px-3 py-1.5 text-xs hover:bg-black/30">
              Open folder
            </button>
          )}
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
  downloadActiveForThisTab = false,
  onDownloadStart,
  onDownloadEnd,
  engineEvents,
  jobSession,
  onClearJobSession,
  onResetPipeline,
  onSettingsChange,
  onSplitIntoTracks,
  sourceMode,
  platform = 'windows',
  showAppleSearch = true,
  showLosslessQualities = true,
}) {
  const [url, setUrl] = useState('')
  const [urlType, setUrlType] = useState('')
  const [quality, setQuality] = useState('aac')
  const [preview, setPreview] = useState(null)
  const [selected, setSelected] = useState(new Set())
  const [fetching, setFetching] = useState(false)
  const [fetchError, setFetchError] = useState('')
  const [jobStarted, setJobStarted] = useState(false)
  const [youtubeDeliveryMode, setYoutubeDeliveryMode] = useState('audio')
  const [fetchStep, setFetchStep] = useState(0)
  const [fetchStatus, setFetchStatus] = useState('')
  const [metaByTrack, setMetaByTrack] = useState({})
  const [preflight, setPreflight] = useState(null)
  const [duplicateInfo, setDuplicateInfo] = useState(null)
  const [checkingDuplicates, setCheckingDuplicates] = useState(false)
  const [appleInputMode, setAppleInputMode] = useState('single')
  const [queueAddRequest, setQueueAddRequest] = useState(null)

  const youtubeMode = sourceMode ? sourceMode === 'youtube' : Boolean(settings?.['youtube-mode'])
  const showSourceToggle = !sourceMode
  const embedSearch = sourceMode === 'apple' && showAppleSearch
  const qualityOptions = showLosslessQualities ? QUALITIES : QUALITIES.filter((q) => !q.needsWrapper)
  const ytDlpOk = deps?.some((d) => d.name === 'yt-dlp' && d.ok)
  const ffmpegOk = deps?.some((d) => d.name === 'ffmpeg' && d.ok)
  const wrapperOk = deps?.some((d) => d.name?.includes('wrapper') && d.ok)
  const hasToken = (settings?.['media-user-token'] || '').length > 50
  const outputFolder = outputFolderForQuality(settings, quality, youtubeMode)
  const outputFolderHint =
    !youtubeMode && quality !== 'aac'
      ? 'Uses the AAC folder if this quality-specific path is empty. Change all folders in Settings.'
      : 'Also editable in Settings → Output folders.'

  const browseOutputFolder = async () => {
    const path = await PickFolder()
    if (!path) return
    await saveOutputFolder(settings, onSettingsChange, quality, youtubeMode, path)
  }

  const saveOutputFolderPath = async (path) => {
    await saveOutputFolder(settings, onSettingsChange, quality, youtubeMode, path)
  }

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

  const onDiskByNum = useMemo(() => {
    const map = {}
    duplicateInfo?.tracks?.forEach((t) => {
      if (t.on_disk) map[t.num] = t
    })
    return map
  }, [duplicateInfo])

  const duplicateFoldersKey = JSON.stringify(settings?.['duplicate-check-folders'] || [])

  useEffect(() => {
    if (!preview?.tracks?.length || downloading) {
      setDuplicateInfo(null)
      return undefined
    }
    let cancelled = false
    const timer = setTimeout(async () => {
      setCheckingDuplicates(true)
      try {
        const selectedTrackNums = preview.can_select_tracks ? [...selected].sort((a, b) => a - b) : []
        const youtubeMeta = youtubeMode ? metaPayload(metaByTrack, selected) : []
        const res = await CheckTrackDuplicates(
          preview.url,
          youtubeMode ? 'youtube' : quality,
          selectedTrackNums,
          youtubeDeliveryMode,
          youtubeMode ? 'youtube' : 'apple',
          youtubeMeta,
          preview,
        )
        if (!cancelled) setDuplicateInfo(res)
      } catch (e) {
        reportFrontendError('CheckTrackDuplicates', e)
        if (!cancelled) setDuplicateInfo(null)
      } finally {
        if (!cancelled) setCheckingDuplicates(false)
      }
    }, 350)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [
    preview,
    quality,
    selected,
    youtubeMode,
    youtubeDeliveryMode,
    metaByTrack,
    outputFolder,
    duplicateFoldersKey,
    downloading,
  ])

  useEffect(() => {
    if (prefillUrl && !youtubeMode) {
      setUrl(prefillUrl)
      onPrefillConsumed()
    }
  }, [prefillUrl, onPrefillConsumed, youtubeMode])

  useEffect(() => {
    const trimmed = url.trim()
    if (trimmed.length <= 12) {
      setUrlType('')
      return
    }
    if (youtubeMode) {
      if (isYouTubeURL(trimmed)) {
        setUrlType(trimmed.includes('list=') ? 'YouTube Playlist' : 'YouTube Video')
      } else if (isAppleMusicURL(trimmed)) {
        setUrlType('Apple Music')
      } else {
        setUrlType('')
      }
      return
    }
    if (isYouTubeURL(trimmed)) {
      setUrlType('YouTube Video')
      return
    }
    DetectURLType(trimmed).then(setUrlType)
  }, [url, youtubeMode])

  useEffect(() => {
    if (!fetching) {
      setFetchStep(0)
      return undefined
    }
    let step = 0
    const timer = setInterval(() => {
      step += 1
      setFetchStep(step)
    }, 1400)
    return () => clearInterval(timer)
  }, [fetching])

  useEffect(() => {
    if (!downloadActiveForThisTab && downloading) {
      setJobStarted(false)
    }
  }, [downloadActiveForThisTab, downloading])

  useEffect(() => {
    if (!downloading && jobStarted && downloadActiveForThisTab) {
      onDownloadEnd?.(parseJobResult(engineEvents))
    }
  }, [downloading, jobStarted, engineEvents, onDownloadEnd, downloadActiveForThisTab])

  const resolvedJobResult = jobResult || jobSession

  const flowPhase = useMemo(
    () => deriveFlowPhase({ preview, jobStarted, downloading, jobResult: resolvedJobResult }),
    [preview, jobStarted, downloading, resolvedJobResult],
  )

  const resetPreview = ({ clearPipeline = false } = {}) => {
    setPreview(null)
    setSelected(new Set())
    setFetchError('')
    setFetchStatus('')
    setJobStarted(false)
    setMetaByTrack({})
    setPreflight(null)
    if (clearPipeline) {
      onResetPipeline?.()
    } else {
      onClearJobSession?.()
    }
  }

  const startAnotherDownload = () => {
    setUrl('')
    setUrlType('')
    setYoutubeDeliveryMode('audio')
    resetPreview({ clearPipeline: true })
  }

  const setMode = async (nextYouTube) => {
    resetPreview({ clearPipeline: true })
    setUrl('')
    setUrlType('')
    await onSettingsChange?.({ 'youtube-mode': nextYouTube })
  }

  const fetchPreview = async (forcedUrl) => {
    const trimmed = (forcedUrl ?? url).trim()
    if (!trimmed) {
      setFetchError(youtubeMode ? 'Paste a YouTube video or playlist link first.' : 'Paste an Apple Music link first.')
      setFetchStatus('')
      return
    }
    const mismatch = urlTabMismatch(trimmed, youtubeMode)
    if (mismatch) {
      setFetchError(mismatch)
      setFetchStatus('')
      return
    }
    if (forcedUrl) setUrl(trimmed)
    const fetchingYouTube = youtubeMode || isYouTubeURL(trimmed)
    setFetching(true)
    setFetchError('')
    setFetchStatus(
      fetchingYouTube
        ? 'Fetching video info from YouTube — this may take a moment…'
        : 'Loading preview from Apple Music…',
    )
    setPreview(null)
    setSelected(new Set())
    setDuplicateInfo(null)
    setJobStarted(false)
    onResetPipeline?.()
    try {
      const res = await PreviewURL(trimmed)
      if (res?.error) {
        setFetchError(formatAppleAuthError(res.error))
        setFetchStatus('')
        return
      }
      if (!res?.title && !(res?.tracks?.length > 0)) {
        setFetchError('Preview returned no metadata. Check the link and try again.')
        setFetchStatus('')
        return
      }
      setPreview(res)
      setSelected(new Set(res.tracks?.map((t) => t.num) || [1]))
      setMetaByTrack(buildMetaFromPreview(res))
      setFetchStatus('')
    } catch (e) {
      reportFrontendError('DownloadTab.fetchPreview', e)
      setFetchError(formatActionError(e, 'Fetch preview'))
      setFetchStatus('')
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
    const mismatch = urlTabMismatch(preview.url || url, youtubeMode)
    if (mismatch) {
      setFetchError(mismatch)
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

    if (youtubeMode) {
      const youtubeMetaDraft = metaPayload(metaByTrack, selected)
      const missingCustomArt = youtubeMetaDraft.find(
        (m) => m?.art_source === 'custom' && !String(m.cover_path || '').trim(),
      )
      if (missingCustomArt) {
        setFetchError('Choose a custom artwork image or switch artwork to YouTube thumbnail.')
        return
      }
    }

    setFetchError('')
    setPreflight(null)

    const sourceMode = youtubeMode ? 'youtube' : 'apple'
    try {
      const check = await PreflightDownloadJob(preview.url, youtubeMode ? 'youtube' : quality, youtubeDeliveryMode, sourceMode)
      if (!check?.ready) {
        setPreflight(check)
        const failed = (check.checks || []).filter((c) => !c.ok && c.blocking).map((c) => c.detail || c.label)
        setFetchError(failed[0] || check.summary || 'Fix the issues below before downloading.')
        return
      }
    } catch (e) {
      reportFrontendError('DownloadTab.preflight', e)
      setFetchError(formatActionError(e, 'Pre-flight check'))
      return
    }

    setJobStarted(true)
    onDownloadStart?.()
    try {
      const youtubeMeta = youtubeMode ? metaPayload(metaByTrack, selected) : []
      await StartDownloadJob(preview.url, youtubeMode ? 'youtube' : quality, selectedTrackNums, childURLs, youtubeDeliveryMode, youtubeMeta)
    } catch (err) {
      setJobStarted(false)
      const msg = typeof err === 'string' ? err : err?.message || String(err)
      setFetchError(formatAppleAuthError(msg))
      onDownloadEnd?.(null)
    }
  }

  const selectedCount = selected.size
  const showTrackPanel =
    jobStarted &&
    (downloading || resolvedJobResult) &&
    (downloadActiveForThisTab || !downloading) &&
    trackRows.length > 0
  const urlUnknown = urlType === 'Unknown' && url.trim().length > 12 && !youtubeMode

  const flowTitle = youtubeMode ? 'Download from YouTube' : 'Download from Apple Music'
  const flowSubtitle =
    flowPhase === 'link'
      ? youtubeMode
        ? 'Paste a video or playlist URL — fetch to preview, then download'
        : embedSearch
          ? 'Paste a link or pick from search — fetch to preview, then download'
          : 'Paste a link, fetch to preview, then download'
      : flowPhase === 'review'
        ? 'Confirm tracks, quality, and output folder'
        : flowPhase === 'downloading'
          ? 'Download in progress — track status updates below'
          : 'Download finished — open files or start another'

  const backAction =
    flowPhase === 'review' ? (
      <button
        type="button"
        onClick={() => resetPreview({ clearPipeline: true })}
        className="text-sm text-white/50 hover:text-white"
        disabled={downloading}
      >
        ← Change link
      </button>
    ) : null

  const flowFooter =
    flowPhase === 'link' ? (
      <button
        type="button"
        onClick={() => fetchPreview()}
        disabled={!url.trim() || fetching || urlUnknown}
        aria-busy={fetching}
        className="w-full rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
      >
        {fetching ? 'Fetching…' : 'Fetch'}
      </button>
    ) : flowPhase === 'review' ? (
      <button
        type="button"
        onClick={startDownload}
        disabled={downloading || selectedCount === 0}
        className="w-full rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
      >
        {youtubeMode
          ? youtubeDownloadButtonLabel(youtubeDeliveryMode, selectedCount)
          : `Download ${selectedCount} selected`}
      </button>
    ) : flowPhase === 'done' ? (
      <div className="flex flex-col gap-2 sm:flex-row">
        <button
          type="button"
          onClick={startDownload}
          disabled={selectedCount === 0}
          className="flex-1 rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
        >
          Download again
        </button>
        <button
          type="button"
          onClick={startAnotherDownload}
          className="flex-1 rounded-xl border border-white/15 bg-white/[0.04] py-3 font-semibold text-white/90 hover:bg-white/[0.08]"
        >
          {youtubeMode ? 'New link' : 'Start new download'}
        </button>
      </div>
    ) : null

  const handleSearchSelect = (searchUrl) => {
    setUrl(searchUrl)
    resetPreview({ clearPipeline: true })
    fetchPreview(searchUrl)
  }

  const addPreviewToQueue = () => {
    if (!preview?.url || downloading) return
    setQueueAddRequest({ url: preview.url, ts: Date.now() })
    setAppleInputMode('bulk')
    resetPreview({ clearPipeline: true })
    setUrl('')
  }

  const showBulkMode = !youtubeMode && appleInputMode === 'bulk'

  return (
    <PageShell wide>
      {!youtubeMode && (
        <AppleDownloadModeSwitch
          mode={appleInputMode}
          disabled={downloading}
          onChange={setAppleInputMode}
        />
      )}

      {showBulkMode ? (
        <BulkDownloadQueue
          quality={quality}
          qualityOptions={qualityOptions}
          settings={settings}
          hasToken={hasToken}
          downloading={downloading}
          jobStarted={jobStarted}
          engineEvents={engineEvents}
          onDownloadStart={() => {
            setJobStarted(true)
            onDownloadStart?.()
          }}
          onDownloadEnd={onDownloadEnd}
          onQualityChange={setQuality}
          addUrlRequest={queueAddRequest}
        />
      ) : (
        <DownloadFlowLayout
          title={flowTitle}
          subtitle={flowSubtitle}
          phase={flowPhase}
          stepperVariant={youtubeMode ? 'youtube' : 'single'}
          backAction={backAction}
          footer={flowFooter}
        >
          {embedSearch && flowPhase === 'link' && (
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

          {flowPhase === 'link' && (
            <>
              {!youtubeMode && appleInputMode === 'single' && (
                <button
                  type="button"
                  disabled={downloading}
                  onClick={() => setAppleInputMode('bulk')}
                  className="text-sm font-medium text-accent hover:underline disabled:opacity-40"
                >
                  Need many albums? Switch to bulk queue →
                </button>
              )}

              <div className="relative">
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

              {fetching && <FetchStatusBanner message={fetchStatus} variant="info" />}
              {fetching && (
                <FetchPreviewSkeleton step={fetchStep} youtubeMode={youtubeMode || isYouTubeURL(url)} />
              )}
              {!fetching && fetchError && <FetchStatusBanner message={fetchError} variant="error" />}

              {!fetching && !fetchError && youtubeMode ? (
                <div className="space-y-2">
                  {(!ytDlpOk || !ffmpegOk || deps?.some((d) => d.name === 'ffprobe (YouTube)' && !d.ok)) && (
                    <p className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-200">
                      Install <strong>yt-dlp</strong>, <strong>ffmpeg</strong>, and <strong>ffprobe</strong> on PATH or in{' '}
                      <code>dist/tools/</code> — see Requirements tab.
                    </p>
                  )}
                  <p className="rounded-lg border border-white/10 bg-white/[0.02] px-3 py-2 text-sm text-white/60">
                    Long DJ sets and live streams are supported. No Apple Music account needed.
                  </p>
                </div>
              ) : (
                !fetching &&
                !fetchError && (
                  <div className="space-y-2">
                    {!hasToken && (
                      <p className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-200">
                        {APPLE_SUBSCRIPTION_MSG}
                      </p>
                    )}
                    <p className="rounded-lg border border-white/10 bg-white/[0.02] px-3 py-2 text-sm text-white/60">
                      Paste a link from <strong className="text-white/80">music.apple.com</strong> — songs, albums,
                      playlists, artists, or music videos.
                    </p>
                  </div>
                )
              )}
            </>
          )}

          {preview && flowPhase !== 'link' && (
            <>
              <div className="flex items-center justify-end gap-3">
                {!youtubeMode && flowPhase === 'review' && (
                  <button
                    type="button"
                    onClick={addPreviewToQueue}
                    disabled={downloading}
                    className="text-sm text-accent hover:underline disabled:opacity-40"
                  >
                    Add to bulk queue
                  </button>
                )}
                {downloading && (
                  <span className="rounded-full bg-accent/20 px-3 py-1 text-xs text-accent animate-pulse">
                    Downloading…
                  </span>
                )}
              </div>

              {flowPhase === 'done' && (
                <JobStatusBanner
                  jobResult={resolvedJobResult}
                  onRevealFile={(path) => RevealInFolder(path)}
                  onOpenFolder={() => OpenFolder('')}
                  onOpenLog={() => OpenLogFile()}
                  onSplitIntoTracks={() => {
                    const h = resolvedJobResult?.handoff
                    if (h?.master_path) onSplitIntoTracks?.(h)
                  }}
                />
              )}

              <div className="flex gap-4 rounded-xl border border-white/10 bg-surface-raised p-4">
                {preview.art_url ? (
                  <img src={preview.art_url} alt="" className="h-24 w-24 shrink-0 rounded-lg object-cover" />
                ) : (
                  <div className="flex h-24 w-24 shrink-0 items-center justify-center rounded-lg bg-surface text-3xl">
                    ▶
                  </div>
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

              {flowPhase === 'review' && !youtubeMode && (
                <div>
                  <p className="mb-2 text-sm text-white/60">Quality</p>
                  <div className="grid gap-2 sm:grid-cols-3">
                    {qualityOptions.map((q) => {
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

              {flowPhase === 'review' && youtubeMode && (
                <div className="space-y-3">
                  <YouTubeDeliveryModeSwitch
                    value={youtubeDeliveryMode}
                    onChange={setYoutubeDeliveryMode}
                    disabled={downloading}
                  />
                  <YouTubeMetadataEditor
                    preview={preview}
                    selected={selected}
                    metaByTrack={metaByTrack}
                    disabled={downloading}
                    deliveryMode={youtubeDeliveryMode}
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
                </div>
              )}

              {flowPhase === 'review' && duplicateInfo?.existing_count > 0 && (
                <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
                  <p className="font-medium">
                    {duplicateInfo.existing_count} of {duplicateInfo.selected_count} selected{' '}
                    {duplicateInfo.existing_count === 1 ? 'track is' : 'tracks are'} already on disk
                  </p>
                  <p className="mt-1 text-xs text-amber-100/80">
                    Existing files are skipped during download.
                  </p>
                </div>
              )}

              {flowPhase === 'review' && checkingDuplicates && !duplicateInfo && preview.can_select_tracks && (
                <p className="text-xs text-white/40">Checking for existing files…</p>
              )}

              {flowPhase === 'review' && preview.can_select_tracks && preview.tracks?.length > 0 && (
                <div className="rounded-xl border border-white/10 bg-surface-raised">
                  <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
                    <span className="text-sm font-medium">
                      {preview.type === 'Artist'
                        ? 'Albums & videos'
                        : preview.type === 'YouTube Playlist'
                          ? 'Videos'
                          : 'Tracks'}
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
                  <ul className="max-h-64 divide-y divide-white/5 overflow-y-auto xl:max-h-[min(70vh,36rem)]">
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
                        {onDiskByNum[t.num] && (
                          <button
                            type="button"
                            title={onDiskByNum[t.num].existing_path || 'Already on disk'}
                            onClick={() =>
                              onDiskByNum[t.num].existing_path && RevealInFolder(onDiskByNum[t.num].existing_path)
                            }
                            className="shrink-0 rounded-full border border-emerald-500/40 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-emerald-300 hover:bg-emerald-500/20"
                          >
                            On disk
                          </button>
                        )}
                        {t.explicit && <span className="shrink-0 text-xs text-white/50">E</span>}
                        {t.is_mv && <span className="shrink-0 text-xs text-white/50">MV</span>}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {flowPhase === 'review' && (
                <OutputFolderField
                  label={outputFolderLabel(quality, youtubeMode)}
                  hint={outputFolderHint}
                  value={outputFolder || preview?.output_folder || ''}
                  disabled={downloading || !onSettingsChange}
                  onBrowse={browseOutputFolder}
                  onOpen={() => outputFolder && OpenFolder(outputFolder)}
                  onSavePath={saveOutputFolderPath}
                />
              )}

              {!youtubeMode && flowPhase === 'review' && quality === 'aac' && !hasToken && (
                <p className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-200">
                  {APPLE_SUBSCRIPTION_MSG}
                </p>
              )}
            </>
          )}

          {showTrackPanel && (
            <TrackStatusPanel
              trackRows={trackRows}
              visible
              downloading={downloading}
              youtubeProgress={youtubeMode ? youtubeProgress : null}
            />
          )}

          {fetchError && flowPhase !== 'link' && (
            <p className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">{fetchError}</p>
          )}

          {preflight && !preflight.ready && flowPhase === 'review' && (
            <ul className="space-y-1 rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-100">
              {(preflight.checks || [])
                .filter((c) => !c.ok)
                .map((c) => (
                  <li key={c.id}>
                    <span className="font-medium">{c.label}:</span> {c.detail || 'Not ready'}
                  </li>
                ))}
            </ul>
          )}
        </DownloadFlowLayout>
      )}
    </PageShell>
  )
}
