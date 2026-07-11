import { useEffect, useState } from 'react'
import {
  ValidateIPhoneSyncFolder,
  PickFolder,
  PreviewPrepareAlbumForSync,
  PrepareAlbumForSync,
  IsAppleMusicRunning,
} from '../../wailsjs/go/main/App'
import { useConfirm } from '../../hooks/useConfirm'
import { confirmAndPrepareAlbum } from '../../lib/prepareAlbumConfirm'
import AppleSyncResetPanel from '../../components/AppleSyncResetPanel'

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

function isArtworkCheck(check) {
  return check?.id === 'embedded_art' || check?.id === 'sidecar_only'
}

export default function SyncValidationPanel({
  result,
  compact = false,
  onFixMissingArtwork,
  onUseFolderCover,
  folderCoverAvailable = false,
  fixingArtwork = false,
}) {
  if (!result) return null
  const artworkFail = (result.checks || []).find((c) => isArtworkCheck(c) && !c.pass)
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
      {artworkFail && (onFixMissingArtwork || onUseFolderCover) && (
        <div className="mt-3 rounded-lg border border-yellow-500/25 bg-black/20 px-3 py-2.5">
          <p className="text-xs font-medium text-yellow-100/90">Fix in Tag Editor</p>
          <p className="mt-1 text-[11px] leading-relaxed text-white/55">
            Choose an image and Save to embed artwork into this file. Folder art alone will not sync to iPhone.
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            {onFixMissingArtwork && (
              <button
                type="button"
                disabled={fixingArtwork}
                onClick={onFixMissingArtwork}
                className="rounded-lg bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent/90 disabled:opacity-50"
              >
                Choose artwork image…
              </button>
            )}
            {folderCoverAvailable && onUseFolderCover && (
              <button
                type="button"
                disabled={fixingArtwork}
                onClick={onUseFolderCover}
                className="rounded-lg border border-white/15 px-3 py-1.5 text-xs hover:bg-white/5 disabled:opacity-50"
              >
                Use folder cover
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export function FolderSyncValidationPanel({ result }) {
  if (!result) return null
  return (
    <div className="space-y-3">
      <SyncValidationPanel result={{ ready: result.ready, summary: result.summary, checks: result.folder_checks || [] }} compact={!(result.folder_checks || []).length} />
      {(result.folder_checks || []).length > 0 && (
        <div className="rounded-lg border border-white/10 bg-black/20 p-3">
          <p className="text-xs font-medium text-white/55">Folder summary</p>
          <ul className="mt-2 space-y-1.5 text-sm">
            {result.folder_checks.map((c) => (
              <li key={c.id} className="flex gap-2 rounded-lg bg-black/20 px-2 py-1.5">
                <CheckIcon pass={c.pass} severity={c.severity} />
                <div className="min-w-0 flex-1">
                  <span className="font-medium text-white/90">{c.label}</span>
                  <span className={`ml-2 ${severityClass(c.severity, c.pass)}`}>{c.detail}</span>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
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
  platform = 'windows',
  hintAlbumFolder = '',
}) {
  const { requestConfirm, ConfirmDialogSlot } = useConfirm()
  const [busy, setBusy] = useState('')
  const [doneKey, setDoneKey] = useState('')
  const [actionFeedback, setActionFeedback] = useState(null)
  const [musicRunning, setMusicRunning] = useState(false)
  const [panelOpen, setPanelOpen] = useState(false)
  const isWindows = platform !== 'darwin'

  useEffect(() => {
    IsAppleMusicRunning().then(setMusicRunning).catch(() => setMusicRunning(false))
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

  const syncHint =
    'Check files first, reset Apple sync after a stuck sync, embed art if needed. Wrong art on iPhone usually means stale entries on the device — delete the album on the phone, then re-sync that album only.'

  const hintFolderName = hintAlbumFolder ? hintAlbumFolder.split(/[/\\]/).pop() : ''
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
            <p className="mt-0.5 text-xs text-white/45">Check files → reset sync stack → embed artwork if needed</p>
          </div>
          <span className="text-xs text-white/35">{panelOpen ? 'Hide' : 'Show'}</span>
        </div>
      </summary>

      <div className="border-t border-white/[0.06] px-4 pb-4 pt-3">
        <p className="text-xs leading-relaxed text-white/50">{syncHint}</p>

        {musicRunning && (
          <p className="mt-3 rounded-lg border border-yellow-500/25 bg-yellow-500/[0.07] px-3 py-2 text-xs text-yellow-100/90">
            Quit Apple Music before rewriting embedded artwork.
          </p>
        )}

        <SectionDivider />

        <SectionLabel step="1">Check folder</SectionLabel>
        <SectionHint>Read-only scan of .m4a files in the folder (not subfolders).</SectionHint>
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

        {isWindows && (
          <>
            <SectionDivider />
            <SectionLabel step="2">Reset Apple sync</SectionLabel>
            <SectionHint>
              After syncing in Apple Devices, if iPhone artwork still looks wrong until you restart Windows — run this
              instead of rebooting.
            </SectionHint>
            <AppleSyncResetPanel compact className="mt-2" />
          </>
        )}

        <SectionDivider />

        <SectionLabel step={isWindows ? '3' : '2'}>Update album artwork</SectionLabel>
        <SectionHint>
          Embeds one shared JPEG per track. Titles and track numbers unchanged. Quit Apple Music first.
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
