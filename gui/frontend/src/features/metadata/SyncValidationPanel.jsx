import { useEffect, useState } from 'react'
import {
  ValidateIPhoneSyncFolder,
  GetAppleMusicCacheInfo,
  ClearAppleMusicArtworkCache,
  ClearAppTempCache,
  ClearAllSyncCaches,
  PickFolder,
  RunSyncRepair,
  RunSyncRepairElevated,
  PreviewPrepareAlbumForSync,
  PrepareAlbumForSync,
  GetSyncRepairPreparePreview,
  OpenSyncRepairLog,
  IsAppleMusicRunning,
} from '../../wailsjs/go/main/App'
import { useConfirm } from '../../hooks/useConfirm'
import { confirmAndPrepareAlbum, syncRepairConfirmDetails } from '../../lib/prepareAlbumConfirm'

const MIN_ACTION_MS = 450

async function withMinDelay(promise, ms = MIN_ACTION_MS) {
  const [result] = await Promise.all([promise, new Promise((resolve) => setTimeout(resolve, ms))])
  return result
}

function severityClass(severity, pass) {
  if (pass) return 'text-green-400'
  if (severity === 'warn') return 'text-yellow-300'
  return 'text-red-300'
}

function CheckIcon({ pass, severity }) {
  if (pass) return <span className="text-green-400">✓</span>
  if (severity === 'warn') return <span className="text-yellow-300">!</span>
  return <span className="text-red-300">✕</span>
}

function SpinnerIcon() {
  return (
    <svg className="h-3.5 w-3.5 animate-spin" viewBox="0 0 16 16" fill="none" aria-hidden>
      <circle cx="8" cy="8" r="5.5" stroke="currentColor" strokeOpacity="0.25" strokeWidth="1.5" />
      <path d="M8 2.5a5.5 5.5 0 0 1 5.5 5.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

function DoneIcon() {
  return (
    <svg className="h-3.5 w-3.5" viewBox="0 0 16 16" fill="none" aria-hidden>
      <path
        d="M3.5 8.5 6.5 11.5 12.5 4.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function ActionFeedbackBanner({ feedback, onDismiss }) {
  if (!feedback) return null
  const styles = {
    success: 'border-green-500/35 bg-green-500/10 text-green-100',
    error: 'border-red-500/35 bg-red-500/10 text-red-100',
    info: 'border-sky-500/30 bg-sky-500/10 text-sky-100',
  }
  return (
    <div
      role="status"
      aria-live="polite"
      className={`mt-3 rounded-lg border px-3 py-2.5 text-sm animate-status-in ${styles[feedback.variant] || styles.info}`}
    >
      <div className="flex items-start gap-2">
        <span className="mt-0.5 shrink-0 font-semibold">
          {feedback.variant === 'success' ? '✓' : feedback.variant === 'error' ? '✕' : 'ℹ'}
        </span>
        <div className="min-w-0 flex-1">
          <p className="font-medium">{feedback.title}</p>
          {feedback.message ? <p className="mt-0.5 text-white/75">{feedback.message}</p> : null}
        </div>
        <button type="button" onClick={onDismiss} className="shrink-0 rounded p-0.5 text-white/45 hover:bg-white/10" aria-label="Dismiss">
          ×
        </button>
      </div>
    </div>
  )
}

function actionButtonClass({ busy, done, disabled, danger }) {
  const base =
    'inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-xs font-medium transition-all duration-300 ease-apple disabled:cursor-not-allowed'
  if (done) return `${base} border border-green-500/40 bg-green-500/15 text-green-100`
  if (busy) return `${base} border border-white/15 bg-white/10 text-white/90 opacity-90`
  if (disabled) return `${base} border border-white/10 opacity-50`
  if (danger) return `${base} border border-red-500/35 bg-red-500/10 text-red-100 hover:bg-red-500/20`
  return `${base} border border-white/15 hover:bg-white/5`
}

function RepairStepsPanel({ result }) {
  if (!result?.steps?.length) return null
  return (
    <div className="mt-3 space-y-1.5">
      <p className="text-xs font-medium uppercase tracking-wide text-white/45">Repair progress</p>
      <ul className="space-y-1 text-sm">
        {result.steps.map((step) => (
          <li key={step.id} className={`flex gap-2 rounded-lg px-2 py-1.5 ${step.ok ? 'bg-green-500/10' : step.skipped ? 'bg-white/5' : 'bg-red-500/10'}`}>
            <span className={step.ok ? 'text-green-400' : step.skipped ? 'text-white/40' : 'text-red-300'}>
              {step.ok ? '✓' : step.skipped ? '–' : '✕'}
            </span>
            <div className="min-w-0">
              <p className="font-medium text-white/90">{step.label}</p>
              {step.detail ? <p className="text-xs text-white/55">{step.detail}</p> : null}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

function ManualStepsChecklist({ steps }) {
  if (!steps?.length) return null
  return (
    <div className="mt-3 rounded-lg border border-white/10 bg-black/20 p-3">
      <p className="text-xs font-medium uppercase tracking-wide text-white/45">Next on your iPhone</p>
      <ol className="mt-2 list-decimal space-y-1 pl-4 text-sm text-white/70">
        {steps.map((step) => (
          <li key={step}>{step}</li>
        ))}
      </ol>
    </div>
  )
}

function SectionLabel({ step, children }) {
  return (
    <div className="mt-5 flex items-baseline gap-2 first:mt-0">
      {step ? (
        <span className="shrink-0 rounded bg-white/5 px-1.5 py-0.5 text-[10px] font-medium tabular-nums text-white/35">
          {step}
        </span>
      ) : null}
      <p className="text-xs font-medium text-white/70">{children}</p>
    </div>
  )
}

function SectionHint({ children }) {
  return <p className="mt-1 text-xs leading-relaxed text-white/45">{children}</p>
}

function SectionDivider() {
  return <div className="mt-5 border-t border-white/[0.06] first:hidden" />
}

export default function SyncValidationPanel({ result, compact = false }) {
  if (!result) return null
  return (
    <div
      className={`rounded-xl border p-4 ${
        result.ready ? 'border-green-500/30 bg-green-500/5' : 'border-yellow-500/30 bg-yellow-500/5'
      }`}
    >
      <div>
        <p className="font-medium">{result.ready ? 'Ready for iPhone sync' : 'Sync issues found'}</p>
        <p className="mt-1 text-sm text-white/70">{result.summary}</p>
      </div>
      {!compact && (result.checks || []).length > 0 && (
        <ul className="mt-3 max-h-56 space-y-1.5 overflow-y-auto text-sm">
          {result.checks.map((c) => (
            <li key={c.id} className="flex gap-2 rounded-lg bg-black/20 px-2 py-1.5">
              <CheckIcon pass={c.pass} severity={c.severity} />
              <div className="min-w-0 flex-1">
                <span className="font-medium text-white/90">{c.label}</span>
                <span className={`ml-2 ${severityClass(c.severity, c.pass)}`}>{c.detail}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function FolderSyncValidationPanel({ result }) {
  if (!result) return null
  return (
    <div className="space-y-3">
      <SyncValidationPanel result={{ ready: result.ready, summary: result.summary, checks: [] }} compact />
      {(result.files || []).map((file) => (
        <div key={file.path} className="rounded-lg border border-white/10 bg-black/20 p-3">
          <p className="truncate text-xs text-white/50" title={file.path}>
            {file.path.split(/[/\\]/).pop()}
          </p>
          <SyncValidationPanel result={file} />
        </div>
      ))}
    </div>
  )
}

export function SyncTroubleshootingPanel({
  onStatus,
  folderResult,
  onFolderResult,
  platform: _platform = 'windows',
  hintAlbumFolder = '',
}) {
  const { requestConfirm, ConfirmDialogSlot } = useConfirm()
  const [cacheInfo, setCacheInfo] = useState(null)
  const [busy, setBusy] = useState('')
  const [doneKey, setDoneKey] = useState('')
  const [actionFeedback, setActionFeedback] = useState(null)
  const [repairResult, setRepairResult] = useState(null)
  const [musicRunning, setMusicRunning] = useState(false)
  const [panelOpen, setPanelOpen] = useState(false)

  const refreshCacheInfo = () => {
    GetAppleMusicCacheInfo().then(setCacheInfo).catch(() => {})
    IsAppleMusicRunning().then(setMusicRunning).catch(() => setMusicRunning(false))
  }

  useEffect(() => {
    refreshCacheInfo()
  }, [])

  const markDone = (key) => {
    setDoneKey(key)
    window.setTimeout(() => setDoneKey(''), 2600)
  }

  const showFeedback = (variant, title, message) => {
    setActionFeedback({ variant, title, message })
  }

  const renderButtonLabel = (key, idleLabel) => {
    if (busy === key) {
      return (
        <>
          <SpinnerIcon />
          Working…
        </>
      )
    }
    if (doneKey === key) {
      return (
        <>
          <DoneIcon />
          Done
        </>
      )
    }
    return idleLabel
  }

  const validateFolderAt = async (folder, key = 'validate-folder') => {
    if (!folder) return
    setActionFeedback(null)
    setDoneKey('')
    setBusy(key)
    try {
      const res = await withMinDelay(ValidateIPhoneSyncFolder(folder))
      onFolderResult?.(res)
      const variant = res.ready ? 'success' : 'info'
      showFeedback(variant, res.ready ? 'Folder check — all clear' : 'Folder check — review results', res.summary)
      onStatus?.(res.summary, variant)
      markDone(key)
    } catch (e) {
      const message = String(e?.message || e)
      showFeedback('error', 'Folder check — failed', message)
      onStatus?.(message, 'error')
    } finally {
      setBusy('')
    }
  }

  const validateFolderPick = async () => {
    const folder = await PickFolder()
    if (folder) await validateFolderAt(folder)
  }

  const prepareFolderAt = async (folder, key = 'prepare') => {
    if (!folder) return
    setActionFeedback(null)
    setDoneKey('')
    setBusy(key)
    try {
      const outcome = await confirmAndPrepareAlbum({
        requestConfirm,
        folder,
        PreviewPrepareAlbumForSync,
        PrepareAlbumForSync,
      })
      if (outcome?.cancelled) {
        onStatus?.('Prepare cancelled.', 'info')
        return
      }
      if (outcome?.error) {
        showFeedback('error', 'Prepare — no tracks', outcome.error)
        onStatus?.(outcome.error, 'error')
        return
      }
      const res = outcome.result
      const validation = await ValidateIPhoneSyncFolder(folder)
      onFolderResult?.(validation)
      const variant = res.errors?.length ? 'error' : 'success'
      showFeedback(variant, res.errors?.length ? 'Prepare — issues' : 'Prepare — done', res.summary)
      onStatus?.(res.summary, variant)
      if (!res.errors?.length) markDone(key)
    } catch (e) {
      const message = String(e?.message || e)
      showFeedback('error', 'Prepare — failed', message)
      onStatus?.(message, 'error')
    } finally {
      setBusy('')
    }
  }

  const prepareFolderPick = async () => {
    const folder = await PickFolder()
    if (folder) await prepareFolderAt(folder)
  }

  const runRepair = async () => {
    setActionFeedback(null)
    setDoneKey('')
    try {
      const preview = await GetSyncRepairPreparePreview()
      const confirmed = await requestConfirm({
        title: 'Run full library artwork repair?',
        message:
          'Updates embedded artwork on tracks under your download folders (recursive), then clears PC caches. Text metadata is preserved; matching tracks are skipped. Does not reset iPhone — delete affected albums on the device before re-syncing.',
        details: syncRepairConfirmDetails(preview),
        confirmLabel: preview?.track_count ? `Repair ${preview.track_count} track(s)` : 'Run full repair',
      })
      if (!confirmed) {
        onStatus?.('Sync repair cancelled.', 'info')
        return
      }
    } catch (e) {
      onStatus?.(String(e?.message || e), 'error')
      return
    }

    setBusy('repair')
    try {
      const res = await withMinDelay(
        RunSyncRepair({ skip_prepare: false, cache_only: false, force_if_music_running: false }),
      )
      setRepairResult(res)
      refreshCacheInfo()
      const variant = res.ok ? 'success' : res.need_elevated ? 'info' : 'error'
      showFeedback(variant, res.ok ? 'Full repair — done' : 'Full repair — review', res.summary)
      onStatus?.(res.summary, variant)
      if (res.ok) markDone('repair')
    } catch (e) {
      showFeedback('error', 'Full repair — failed', String(e?.message || e))
      onStatus?.(String(e?.message || e), 'error')
    } finally {
      setBusy('')
    }
  }

  const runClear = async (fn, key, label) => {
    setActionFeedback(null)
    setDoneKey('')
    setBusy(key)
    try {
      const res = await withMinDelay(fn())
      const variant = res.ok ? 'success' : 'error'
      showFeedback(variant, res.ok ? `${label} — done` : `${label} — failed`, res.message)
      onStatus?.(res.message || `${label} complete`, variant)
      if (res.ok) {
        markDone(key)
        refreshCacheInfo()
      }
    } catch (e) {
      showFeedback('error', `${label} — failed`, String(e?.message || e))
    } finally {
      setBusy('')
    }
  }

  const runElevated = async () => {
    setBusy('elevated')
    try {
      const res = await withMinDelay(RunSyncRepairElevated())
      setRepairResult((prev) => ({ ...prev, ...res, steps: [...(prev?.steps || []), ...(res.steps || [])] }))
      showFeedback(res.ok ? 'success' : 'error', res.ok ? 'Admin cache clear — done' : 'Admin cache clear — failed', res.summary)
      if (res.ok) {
        markDone('elevated')
        refreshCacheInfo()
      }
    } finally {
      setBusy('')
    }
  }

  const syncHint =
    'Start at the top and work down only if needed. Wrong art on iPhone after a good PC import usually means stale entries on the device — delete the album on the phone, then re-sync that album only.'

  const hintFolderName = hintAlbumFolder ? hintAlbumFolder.split(/[/\\]/).pop() : ''
  const cacheBlocked = musicRunning || !!busy
  const fileChangeBlocked = musicRunning || !!busy

  return (
    <details
      open={panelOpen}
      onToggle={(e) => setPanelOpen(e.target.open)}
      className="rounded-xl border border-white/10 bg-white/[0.02]"
    >
      {ConfirmDialogSlot}
      <summary className="cursor-pointer list-none px-4 py-3 [&::-webkit-details-marker]:hidden">
        <div className="flex items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-medium text-white/90">Sync repair tools</h3>
            <p className="mt-0.5 text-xs text-white/45">Safest actions first — checks, then PC caches, then file changes</p>
          </div>
          <span className="text-xs text-white/35">{panelOpen ? 'Hide' : 'Show'}</span>
        </div>
      </summary>

      <div className="border-t border-white/[0.06] px-4 pb-4 pt-3">
        <p className="text-xs leading-relaxed text-white/50">{syncHint}</p>

        <p className="mt-2 text-[11px] leading-relaxed text-white/35">
          Nothing here touches your iPhone directly. Cache clears affect PC only. Prepare updates embedded cover art only — titles and track numbers stay put. Tracks already correct are skipped.
        </p>

        {musicRunning && (
          <p className="mt-3 rounded-lg border border-yellow-500/25 bg-yellow-500/[0.07] px-3 py-2 text-xs text-yellow-100/90">
            Quit Apple Music before clearing caches or rewriting artwork.
          </p>
        )}

        <SectionDivider />

        <SectionLabel step="1">Check folder</SectionLabel>
        <SectionHint>
          Read-only. Scans .m4a files directly in the folder (not subfolders). Nothing is written to disk.
        </SectionHint>
        <div className="mt-2 flex flex-wrap gap-2">
          {hintAlbumFolder && (
            <button
              type="button"
              disabled={!!busy}
              onClick={() => validateFolderAt(hintAlbumFolder, 'validate-hint')}
              className={actionButtonClass({ busy: busy === 'validate-hint', done: doneKey === 'validate-hint', disabled: !!busy && busy !== 'validate-hint' })}
            >
              {renderButtonLabel('validate-hint', `Check open folder${hintFolderName ? `: ${hintFolderName}` : ''}`)}
            </button>
          )}
          <button
            type="button"
            disabled={!!busy}
            onClick={validateFolderPick}
            className={actionButtonClass({ busy: busy === 'validate-folder', done: doneKey === 'validate-folder', disabled: !!busy && busy !== 'validate-folder' })}
          >
            {renderButtonLabel('validate-folder', 'Check chosen folder…')}
          </button>
        </div>

        <SectionDivider />

        <SectionLabel step="2">Clear PC caches</SectionLabel>
        <SectionHint>
          Deletes Apple Music artwork cache folders on this PC only — never your music files or iPhone library. Quit Apple Music first. Re-import albums afterward so PC picks up fresh embedded art.
        </SectionHint>
        {cacheInfo?.note ? <p className="mt-1.5 text-[11px] text-white/35">{cacheInfo.note}</p> : null}
        <div className="mt-2 flex flex-wrap gap-2">
          <button
            type="button"
            disabled={cacheBlocked}
            onClick={() => runClear(ClearAllSyncCaches, 'all', 'Clear all caches')}
            className={actionButtonClass({ busy: busy === 'all', done: doneKey === 'all', disabled: cacheBlocked && busy !== 'all' })}
          >
            {renderButtonLabel('all', 'Clear all sync caches')}
          </button>
          <button
            type="button"
            disabled={cacheBlocked}
            onClick={() => runClear(ClearAppleMusicArtworkCache, 'apple', 'Clear Apple Music cache')}
            className={actionButtonClass({ busy: busy === 'apple', done: doneKey === 'apple', disabled: cacheBlocked && busy !== 'apple' })}
          >
            {renderButtonLabel('apple', 'Apple Music art cache')}
          </button>
          <button
            type="button"
            disabled={!!busy}
            onClick={() => runClear(ClearAppTempCache, 'app', 'Clear app temp')}
            className={actionButtonClass({ busy: busy === 'app', done: doneKey === 'app', disabled: !!busy && busy !== 'app' })}
          >
            {renderButtonLabel('app', 'App temp files')}
          </button>
        </div>

        <SectionDivider />

        <SectionLabel step="3">Update album artwork</SectionLabel>
        <SectionHint>
          Embeds one shared JPEG cover in each .m4a in the folder (direct files only). Title, artist, and track numbers are not changed. Tracks that already match are skipped. If every track already shares the same art, a folder cover.jpg is not used. Quit Apple Music before running.
        </SectionHint>
        <div className="mt-2 flex flex-wrap gap-2">
          {hintAlbumFolder && (
            <button
              type="button"
              disabled={fileChangeBlocked}
              onClick={() => prepareFolderAt(hintAlbumFolder, 'prepare-hint')}
              className={actionButtonClass({
                busy: busy === 'prepare-hint',
                done: doneKey === 'prepare-hint',
                disabled: fileChangeBlocked && busy !== 'prepare-hint',
                danger: true,
              })}
            >
              {renderButtonLabel('prepare-hint', `Update open folder${hintFolderName ? `: ${hintFolderName}` : ''}`)}
            </button>
          )}
          <button
            type="button"
            disabled={fileChangeBlocked}
            onClick={prepareFolderPick}
            className={actionButtonClass({
              busy: busy === 'prepare',
              done: doneKey === 'prepare',
              disabled: fileChangeBlocked && busy !== 'prepare',
              danger: true,
            })}
          >
            {renderButtonLabel('prepare', 'Update chosen folder…')}
          </button>
        </div>

        <SectionDivider />

        <details className="rounded-lg border border-white/10 bg-black/10 px-3 py-2.5">
          <summary className="cursor-pointer text-xs font-medium text-white/65 [&::-webkit-details-marker]:hidden">
            <span className="inline-flex items-center gap-2">
              <span className="rounded bg-white/5 px-1.5 py-0.5 text-[10px] font-medium tabular-nums text-white/35">4</span>
              Heavy repair — entire library
            </span>
          </summary>
          <SectionHint>
            Last resort. Same as step 3 but across all configured download folders (including subfolders), then clears PC caches. Cannot fix iPhone stale art by itself — remove albums from Apple Music, re-import, delete on iPhone, sync selected albums first.
          </SectionHint>
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={fileChangeBlocked}
              onClick={runRepair}
              className={`${actionButtonClass({ busy: busy === 'repair', done: doneKey === 'repair', disabled: fileChangeBlocked && busy !== 'repair', danger: true })} px-4 py-2.5 text-sm`}
            >
              {renderButtonLabel('repair', 'Repair entire library')}
            </button>
            {repairResult?.need_elevated && (
              <button
                type="button"
                disabled={!!busy}
                onClick={runElevated}
                className={actionButtonClass({ busy: busy === 'elevated', done: doneKey === 'elevated', disabled: !!busy && busy !== 'elevated' })}
              >
                {renderButtonLabel('elevated', 'Run as administrator')}
              </button>
            )}
            {repairResult?.log_path && (
              <button
                type="button"
                disabled={!!busy}
                onClick={() => OpenSyncRepairLog().catch(() => {})}
                className={actionButtonClass({ busy: false, done: false, disabled: !!busy })}
              >
                Open repair log
              </button>
            )}
          </div>
          <RepairStepsPanel result={repairResult} />
          <ManualStepsChecklist steps={repairResult?.manual_steps} />
        </details>

        <ActionFeedbackBanner feedback={actionFeedback} onDismiss={() => setActionFeedback(null)} />

        {folderResult && (
          <div className="mt-4">
            <FolderSyncValidationPanel result={folderResult} />
          </div>
        )}
      </div>
    </details>
  )
}
