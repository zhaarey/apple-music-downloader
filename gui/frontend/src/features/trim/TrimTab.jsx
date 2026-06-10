import { useCallback, useEffect, useRef, useState } from 'react'
import {
  TrimPickSourceFile,
  TrimProbeFile,
  TrimGetPeaks,
  TrimMediaURL,
  TrimDefaultOutputPath,
  TrimStartExport,
  TrimCancelExport,
  TrimIsExporting,
  TagPickSaveMediaFile,
  RevealInFolder,
  EventsOn,
} from '../../wailsjs/go/main/App'
import PageShell from '../../components/PageShell'
import TrimRangeEditor from './TrimRangeEditor'
import { useTrimRange } from './useTrimRange'
import { useMasterAudio } from '../splice/useMasterAudio'
import { formatMsPrecise, parseTimeInput } from '../splice/spliceTime'
import { resolveMediaURL } from '../../lib/resolveMediaURL'
import { formatActionError } from '../../lib/formatActionError'
import { reportFrontendError } from '../../lib/errorReporting'
import { useConfirm } from '../../hooks/useConfirm'

function TimeField({ label, value, onChange, disabled }) {
  return (
    <label className="block text-xs text-white/50">
      {label}
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 font-mono text-sm disabled:opacity-50"
      />
    </label>
  )
}

function NudgeButton({ children, onClick, disabled }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="rounded-md border border-white/10 bg-black/20 px-2 py-1 text-[11px] text-white/70 hover:bg-black/30 disabled:opacity-40"
    >
      {children}
    </button>
  )
}

export default function TrimTab({ handoff, onHandoffConsumed }) {
  const [sourcePath, setSourcePath] = useState('')
  const [probe, setProbe] = useState(null)
  const [peaks, setPeaks] = useState(null)
  const [mediaURL, setMediaURL] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [exporting, setExporting] = useState(false)
  const [exportMsg, setExportMsg] = useState('')
  const [copyTags, setCopyTags] = useState(true)
  const [startDraft, setStartDraft] = useState('0:00.000')
  const [endDraft, setEndDraft] = useState('0:00.000')

  const videoRef = useRef(null)
  const { requestConfirm, ConfirmDialogSlot } = useConfirm()
  const trimRange = useTrimRange()
  const { audioRef, status: audioStatus, error: audioError, isReady: audioReady } = useMasterAudio(
    probe?.media_kind === 'audio' ? mediaURL : '',
  )

  const isVideo = probe?.media_kind === 'video'
  const durationMs = probe?.duration_ms || 0
  const busy = loading || exporting

  const seekPreview = useCallback(
    (ms) => {
      const sec = Math.max(0, ms) / 1000
      if (isVideo && videoRef.current) {
        videoRef.current.currentTime = sec
      } else if (audioRef.current) {
        audioRef.current.currentTime = sec
      }
    },
    [isVideo, audioRef],
  )

  const syncDrafts = useCallback((start, end) => {
    setStartDraft(formatMsPrecise(start, start >= 3600000))
    setEndDraft(formatMsPrecise(end, end >= 3600000))
  }, [])

  const loadFile = useCallback(
    async (path) => {
      const trimmed = String(path || '').trim()
      if (!trimmed) return
      setLoading(true)
      setError('')
      setNotice('')
      setPeaks(null)
      setProbe(null)
      setMediaURL('')
      try {
        const info = await TrimProbeFile(trimmed)
        const url = await TrimMediaURL(trimmed)
        const waveform = await TrimGetPeaks(trimmed, 2000)
        setSourcePath(trimmed)
        setProbe(info)
        setMediaURL(url)
        setPeaks(waveform)
        trimRange.resetRange(info.duration_ms)
        syncDrafts(0, info.duration_ms)
      } catch (e) {
        reportFrontendError('TrimTab.loadFile', e)
        setError(formatActionError(e, 'Load file'))
      } finally {
        setLoading(false)
      }
    },
    [trimRange, syncDrafts],
  )

  useEffect(() => {
    if (!handoff) return
    loadFile(handoff)
    onHandoffConsumed?.()
  }, [handoff, loadFile, onHandoffConsumed])

  useEffect(() => {
    const off = EventsOn('trim:event', (ev) => {
      if (ev?.type === 'trim_progress') {
        setExporting(true)
        setExportMsg(ev.message || 'Trimming…')
      }
      if (ev?.type === 'trim_complete') {
        setExporting(false)
        setExportMsg('')
        if (ev.phase === 'cancelled') {
          setNotice('Trim cancelled.')
        } else if (ev.phase === 'success') {
          setNotice(ev.message || 'Trim saved.')
          if (ev.output_path) setSourcePath(ev.output_path)
        }
      }
      if (ev?.type === 'trim_error') {
        setExporting(false)
        setExportMsg('')
        setError(ev.message || 'Trim failed.')
      }
    })
    return () => off?.()
  }, [])

  useEffect(() => {
    TrimIsExporting().then(setExporting)
  }, [])

  const pickFile = async () => {
    try {
      const path = await TrimPickSourceFile()
      if (path) await loadFile(path)
    } catch (e) {
      setError(formatActionError(e, 'Open file'))
    }
  }

  const applyStartDraft = () => {
    const ms = parseTimeInput(startDraft)
    if (ms == null) {
      setError('Invalid start time.')
      return
    }
    const next = trimRange.setStartMs(ms, durationMs)
    syncDrafts(next.startMs, next.endMs)
    seekPreview(next.startMs)
  }

  const applyEndDraft = () => {
    const ms = parseTimeInput(endDraft)
    if (ms == null) {
      setError('Invalid end time.')
      return
    }
    const next = trimRange.setEndMs(ms, durationMs)
    syncDrafts(next.startMs, next.endMs)
    seekPreview(next.endMs)
  }

  const runExport = async ({ overwrite = false, outputPath = '' } = {}) => {
    if (!sourcePath || !trimRange.valid) {
      setError('Pick a valid trim range (at least 1 second).')
      return
    }
    setError('')
    setNotice('')
    try {
      await TrimStartExport({
        source_path: sourcePath,
        output_path: outputPath,
        start_ms: trimRange.startMs,
        end_ms: trimRange.endMs,
        copy_tags: copyTags,
        overwrite,
      })
    } catch (e) {
      setError(formatActionError(e, 'Trim export'))
    }
  }

  const saveAsNew = async () => {
    const suggested = await TrimDefaultOutputPath(sourcePath)
    const dest = await TagPickSaveMediaFile(sourcePath || suggested)
    if (!dest) return
    await runExport({ outputPath: dest })
  }

  const saveOverwrite = () => {
    requestConfirm({
      title: 'Replace original file?',
      message:
        'The trimmed version will replace the current file. A one-time backup is saved as filename.bak if no backup exists yet.',
      confirmLabel: 'Replace file',
      onConfirm: () => runExport({ overwrite: true }),
    })
  }

  const handleNudge = (field, deltaMs) => {
    const next = trimRange.nudge(field, deltaMs, durationMs)
    syncDrafts(next.startMs, next.endMs)
    seekPreview(field === 'start' ? next.startMs : next.endMs)
  }

  const previewURL = mediaURL ? resolveMediaURL(mediaURL) : ''

  return (
    <PageShell wide>
      <div className="space-y-4">
        <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
          <h2 className="text-lg font-semibold">Trim media</h2>
          <p className="mt-1 text-sm text-white/50">
            Remove dead space from YouTube downloads or any local .m4a / .mp4 — set start and end, then save.
          </p>

          <div className="mt-4 flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={pickFile}
              disabled={busy}
              className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold hover:bg-accent-muted disabled:opacity-40"
            >
              Open file
            </button>
            {sourcePath && (
              <>
                <button
                  type="button"
                  onClick={() => RevealInFolder(sourcePath)}
                  className="rounded-xl border border-white/15 px-4 py-2 text-sm text-white/80 hover:bg-white/5"
                >
                  Reveal in folder
                </button>
                {exporting && (
                  <button
                    type="button"
                    onClick={() => TrimCancelExport()}
                    className="rounded-xl border border-red-500/30 px-4 py-2 text-sm text-red-200 hover:bg-red-500/10"
                  >
                    Cancel export
                  </button>
                )}
              </>
            )}
          </div>

          {sourcePath && (
            <p className="mt-2 truncate font-mono text-xs text-white/40" title={sourcePath}>
              {sourcePath}
            </p>
          )}
          {probe?.summary && <p className="mt-1 text-xs text-white/55">{probe.summary}</p>}
        </div>

        {loading && (
          <p className="rounded-lg border border-accent/25 bg-accent/10 px-3 py-2 text-sm text-accent">
            Loading file and waveform…
          </p>
        )}

        {error && (
          <p className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">{error}</p>
        )}
        {notice && (
          <p className="rounded-lg border border-emerald-500/25 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-100">
            {notice}
          </p>
        )}
        {exporting && exportMsg && (
          <p className="rounded-lg border border-accent/25 bg-accent/10 px-3 py-2 text-sm text-accent">{exportMsg}</p>
        )}

        {sourcePath && !loading && (
          <>
            <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
              <p className="text-sm font-medium">Preview</p>
              {isVideo ? (
                <video
                  ref={videoRef}
                  src={previewURL}
                  controls
                  className="mt-3 max-h-64 w-full rounded-lg bg-black"
                />
              ) : (
                <>
                  <audio ref={audioRef} controls className="mt-3 w-full" />
                  {audioStatus === 'loading' && (
                    <p className="mt-2 text-xs text-white/45">Loading audio preview…</p>
                  )}
                  {audioError && <p className="mt-2 text-xs text-red-300">{audioError}</p>}
                  {audioReady && (
                    <p className="mt-2 text-xs text-white/45">Scrub preview, then adjust trim handles below.</p>
                  )}
                </>
              )}
            </div>

            <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
              <p className="text-sm font-medium">Trim range</p>
              <p className="mt-1 text-xs text-white/45">
                Drag green (start) and red (end) handles, or edit times below. Selection:{' '}
                {formatMsPrecise(trimRange.selectionMs)}.
              </p>
              <div className="mt-3">
                <TrimRangeEditor
                  peaks={peaks}
                  durationMs={durationMs}
                  startMs={trimRange.startMs}
                  endMs={trimRange.endMs}
                  disabled={busy}
                  onStartChange={(ms) => {
                    const next = trimRange.setStartMs(ms, durationMs)
                    syncDrafts(next.startMs, next.endMs)
                  }}
                  onEndChange={(ms) => {
                    const next = trimRange.setEndMs(ms, durationMs)
                    syncDrafts(next.startMs, next.endMs)
                  }}
                  onSeek={seekPreview}
                />
              </div>

              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <TimeField label="Start" value={startDraft} onChange={setStartDraft} disabled={busy} />
                <TimeField label="End" value={endDraft} onChange={setEndDraft} disabled={busy} />
              </div>
              <div className="mt-2 flex flex-wrap gap-2">
                <NudgeButton disabled={busy} onClick={applyStartDraft}>
                  Apply start
                </NudgeButton>
                <NudgeButton disabled={busy} onClick={applyEndDraft}>
                  Apply end
                </NudgeButton>
                <NudgeButton disabled={busy} onClick={() => handleNudge('start', -1000)}>
                  Start −1s
                </NudgeButton>
                <NudgeButton disabled={busy} onClick={() => handleNudge('start', 1000)}>
                  Start +1s
                </NudgeButton>
                <NudgeButton disabled={busy} onClick={() => handleNudge('end', -1000)}>
                  End −1s
                </NudgeButton>
                <NudgeButton disabled={busy} onClick={() => handleNudge('end', 1000)}>
                  End +1s
                </NudgeButton>
              </div>

              <label className="mt-4 flex items-center gap-2 text-sm text-white/70">
                <input
                  type="checkbox"
                  checked={copyTags}
                  onChange={(e) => setCopyTags(e.target.checked)}
                  disabled={busy}
                />
                Copy tags from original file
              </label>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row">
              <button
                type="button"
                onClick={saveAsNew}
                disabled={busy || !trimRange.valid}
                className="flex-1 rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
              >
                Save as new file
              </button>
              <button
                type="button"
                onClick={saveOverwrite}
                disabled={busy || !trimRange.valid}
                className="flex-1 rounded-xl border border-white/15 bg-white/[0.04] py-3 font-semibold text-white/90 hover:bg-white/[0.08] disabled:opacity-40"
              >
                Save (replace original)
              </button>
            </div>
          </>
        )}
      </div>
      {ConfirmDialogSlot}
    </PageShell>
  )
}
