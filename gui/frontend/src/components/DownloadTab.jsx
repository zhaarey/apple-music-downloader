import { useEffect, useMemo, useState } from 'react'
import { DetectURLType, PreviewURL, StartDownloadJob, OpenLogFile, OpenFolder } from '../wailsjs/go/main/App'
import { parseJobResult, parseTrackRows, jobStatusMeta, trackStatusIcon, trackStatusClass } from '../lib/downloadStatus'

const QUALITIES = [
  { id: 'aac', label: 'AAC', desc: 'Works immediately', needsWrapper: false },
  { id: 'alac', label: 'Lossless', desc: 'Requires wrapper', needsWrapper: true },
  { id: 'atmos', label: 'Dolby Atmos', desc: 'Requires wrapper', needsWrapper: true },
]

function outputFolderForQuality(settings, quality) {
  if (quality === 'alac') return settings?.['alac-save-folder'] || settings?.['aac-save-folder'] || ''
  if (quality === 'atmos') return settings?.['atmos-save-folder'] || settings?.['aac-save-folder'] || ''
  return settings?.['aac-save-folder'] || ''
}

function JobStatusBanner({ jobResult, onOpenFolder, onOpenLog }) {
  if (!jobResult) return null
  const meta = jobStatusMeta(jobResult.phase)
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
        <div className="flex gap-2">
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
}) {
  const [url, setUrl] = useState('')
  const [urlType, setUrlType] = useState('')
  const [quality, setQuality] = useState('aac')
  const [preview, setPreview] = useState(null)
  const [selected, setSelected] = useState(new Set())
  const [fetching, setFetching] = useState(false)
  const [fetchError, setFetchError] = useState('')
  const [jobStarted, setJobStarted] = useState(false)

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

  useEffect(() => {
    if (prefillUrl) {
      setUrl(prefillUrl)
      onPrefillConsumed()
    }
  }, [prefillUrl, onPrefillConsumed])

  useEffect(() => {
    if (url.length > 20) {
      DetectURLType(url).then(setUrlType)
    } else {
      setUrlType('')
    }
  }, [url])

  useEffect(() => {
    if (!downloading && jobStarted) {
      onDownloadEnd?.(parseJobResult(engineEvents))
    }
  }, [downloading, jobStarted, engineEvents, onDownloadEnd])

  const resetPreview = () => {
    setPreview(null)
    setSelected(new Set())
    setFetchError('')
    setJobStarted(false)
    onClearJobSession?.()
  }

  const fetchPreview = async () => {
    const trimmed = url.trim()
    if (!trimmed) return
    setFetching(true)
    setFetchError('')
    setPreview(null)
    setSelected(new Set())
    setJobStarted(false)
    onClearJobSession?.()
    try {
      const res = await PreviewURL(trimmed)
      if (res.error) {
        setFetchError(res.error)
        return
      }
      setPreview(res)
      setSelected(new Set(res.tracks?.map((t) => t.num) || []))
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
    if (quality === 'aac' && !hasToken) {
      setFetchError('AAC downloads require media-user-token in Settings')
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
      await StartDownloadJob(preview.url, quality, selectedTrackNums, childURLs)
    } catch (err) {
      setJobStarted(false)
      const msg = typeof err === 'string' ? err : err?.message || String(err)
      setFetchError(msg)
      onDownloadEnd?.(null)
    }
  }

  const selectedCount = selected.size
  const showProgress = jobStarted && trackRows.length > 0

  return (
    <div className="mx-auto flex h-full max-w-3xl flex-col gap-4 overflow-y-auto pb-4">
      {!preview && (
        <>
          <div>
            <h2 className="text-xl font-semibold">Download from Apple Music</h2>
            <p className="mt-1 text-sm text-white/50">
              Paste a link, fetch to preview, then download — success and errors show here when finished
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
                placeholder="https://music.apple.com/us/playlist/..."
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
              disabled={!url.trim() || fetching || urlType === 'Unknown'}
              className="rounded-xl bg-accent px-5 py-3 font-medium hover:bg-accent-muted disabled:opacity-40"
            >
              {fetching ? 'Fetching…' : 'Fetch'}
            </button>
          </div>

          {fetchError && <p className="text-sm text-red-400">{fetchError}</p>}
          {!hasToken && (
            <p className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-200">
              Add your Apple Music <code className="text-accent">media-user-token</code> in Settings before downloading AAC.
            </p>
          )}
        </>
      )}

      {preview && (
        <>
          <div className="flex items-start justify-between gap-4">
            <button
              type="button"
              onClick={resetPreview}
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
          />

          <div className="flex gap-4 rounded-xl border border-white/10 bg-surface-raised p-4">
            {preview.art_url ? (
              <img src={preview.art_url} alt="" className="h-24 w-24 shrink-0 rounded-lg object-cover" />
            ) : (
              <div className="flex h-24 w-24 shrink-0 items-center justify-center rounded-lg bg-surface text-3xl">♫</div>
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
              <p className="mt-2 text-sm text-yellow-400">Wrapper not running — check Requirements tab before using Lossless or Atmos.</p>
            )}
          </div>

          {preview.can_select_tracks && preview.tracks?.length > 0 && (
            <div className="rounded-xl border border-white/10 bg-surface-raised">
              <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
                <span className="text-sm font-medium">
                  {preview.type === 'Artist' ? 'Albums & videos' : 'Tracks'}
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
                    <span className="w-6 shrink-0 text-right text-xs text-white/40">{t.num}</span>
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
                        {r.detail && r.status !== 'done' && (
                          <p className={`mt-0.5 ${trackStatusClass(r.status)}`}>{r.detail}</p>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <p className="truncate text-xs text-white/40" title={outputFolderForQuality(settings, quality)}>
            Output: {outputFolderForQuality(settings, quality) || preview.output_folder || 'Default downloads folder'}
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
            {downloading ? 'Downloading…' : jobResult ? 'Download again' : `Download ${selectedCount} selected`}
          </button>
        </>
      )}
    </div>
  )
}
