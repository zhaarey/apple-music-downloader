import { useEffect, useState } from 'react'
import {
  ValidateIPhoneSyncFolder,
  GetAppleMusicCacheInfo,
  ClearAppleMusicArtworkCache,
  ClearAppTempCache,
  ClearAllSyncCaches,
  PickFolder,
} from '../../wailsjs/go/main/App'

const MIN_ACTION_MS = 450

const ACTION_LABELS = {
  all: 'Clear all sync caches',
  apple: 'Clear Apple Music art cache',
  app: 'Clear app temp files',
  'validate-folder': 'Validate album folder',
}

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
          {feedback.details?.length > 0 ? (
            <ul className="mt-1.5 max-h-24 space-y-0.5 overflow-y-auto text-xs text-white/55">
              {feedback.details.map((line) => (
                <li key={line} className="truncate" title={line}>
                  {line}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
        <button
          type="button"
          onClick={onDismiss}
          className="shrink-0 rounded p-0.5 text-white/45 transition-colors hover:bg-white/10 hover:text-white/80"
          aria-label="Dismiss"
        >
          ×
        </button>
      </div>
    </div>
  )
}

function actionButtonClass({ busy, done, disabled }) {
  const base =
    'inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-xs font-medium transition-all duration-300 ease-apple disabled:cursor-not-allowed'
  if (done) {
    return `${base} border border-green-500/40 bg-green-500/15 text-green-100`
  }
  if (busy) {
    return `${base} border border-white/15 bg-white/10 text-white/90 opacity-90`
  }
  if (disabled) {
    return `${base} border border-white/10 opacity-50`
  }
  return `${base} border border-white/15 hover:bg-white/5`
}

export default function SyncValidationPanel({ result, compact = false }) {
  if (!result) return null
  return (
    <div
      className={`rounded-xl border p-4 ${
        result.ready ? 'border-green-500/30 bg-green-500/5' : 'border-yellow-500/30 bg-yellow-500/5'
      }`}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <p className="font-medium">{result.ready ? 'Ready for iPhone sync' : 'Sync issues found'}</p>
          <p className="mt-1 text-sm text-white/70">{result.summary}</p>
        </div>
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
      <SyncValidationPanel
        result={{
          ready: result.ready,
          summary: result.summary,
          checks: [],
        }}
        compact
      />
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

export function SyncTroubleshootingPanel({ onStatus, folderResult, onFolderResult, platform = 'windows' }) {
  const [cacheInfo, setCacheInfo] = useState(null)
  const [busy, setBusy] = useState('')
  const [doneKey, setDoneKey] = useState('')
  const [actionFeedback, setActionFeedback] = useState(null)

  const refreshCacheInfo = () => {
    GetAppleMusicCacheInfo().then(setCacheInfo).catch(() => {})
  }

  useEffect(() => {
    refreshCacheInfo()
  }, [])

  const markDone = (key) => {
    setDoneKey(key)
    window.setTimeout(() => setDoneKey(''), 2600)
  }

  const showFeedback = (variant, title, message, details) => {
    setActionFeedback({ variant, title, message, details })
  }

  const runClear = async (fn, key) => {
    const label = ACTION_LABELS[key] || 'Action'
    setActionFeedback(null)
    setDoneKey('')
    setBusy(key)
    try {
      const res = await withMinDelay(fn())
      const variant = res.ok ? 'success' : 'error'
      const details =
        res.cleared?.length > 0 ? res.cleared : res.errors?.length > 0 ? res.errors : undefined

      showFeedback(
        variant,
        res.ok ? `${label} — done` : `${label} — failed`,
        res.message || (res.ok ? 'Completed successfully.' : 'Something went wrong.'),
        details,
      )
      onStatus?.(res.message || `${label} complete`, variant)

      if (res.ok) {
        markDone(key)
        refreshCacheInfo()
      }
    } catch (e) {
      const message = String(e?.message || e)
      showFeedback('error', `${label} — failed`, message)
      onStatus?.(message, 'error')
    } finally {
      setBusy('')
    }
  }

  const validateFolder = async () => {
    const folder = await PickFolder()
    if (!folder) return

    const label = ACTION_LABELS['validate-folder']
    setActionFeedback(null)
    setDoneKey('')
    setBusy('validate-folder')
    try {
      const res = await withMinDelay(ValidateIPhoneSyncFolder(folder))
      onFolderResult?.(res)

      const variant = res.ready ? 'success' : 'info'
      showFeedback(
        variant,
        res.ready ? `${label} — all clear` : `${label} — review results`,
        res.summary,
        res.total > 0 ? [`${res.ready_count ?? 0} of ${res.total} track(s) ready`, folder] : [folder],
      )
      onStatus?.(res.summary, variant)
      markDone('validate-folder')
    } catch (e) {
      const message = String(e?.message || e)
      showFeedback('error', `${label} — failed`, message)
      onStatus?.(message, 'error')
    } finally {
      setBusy('')
    }
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

  const buttons = [
    {
      key: 'all',
      label: 'Clear all sync caches',
      primary: true,
      onClick: () => runClear(ClearAllSyncCaches, 'all'),
    },
    {
      key: 'apple',
      label: 'Clear Apple Music art cache',
      onClick: () => runClear(ClearAppleMusicArtworkCache, 'apple'),
    },
    {
      key: 'app',
      label: 'Clear app temp files',
      onClick: () => runClear(ClearAppTempCache, 'app'),
    },
    {
      key: 'validate-folder',
      label: 'Validate album folder',
      onClick: validateFolder,
    },
  ]

  const syncHint = platform === 'darwin'
    ? 'If artwork shows on Mac but not iPhone: Save tags (normalizes JPEG art), clear caches below, re-import the album in Music, delete it on your phone, then sync via Finder.'
    : 'If artwork shows on PC but not iPhone: Save tags (normalizes JPEG art), clear caches below, re-import the album in Apple Music, delete it on your phone, then sync in Apple Devices.'

  return (
    <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
      <h3 className="text-sm font-medium">Sync troubleshooting</h3>
      <p className="mt-1 text-xs text-white/55">{syncHint}</p>
      {cacheInfo?.note && <p className="mt-2 text-xs text-white/45">{cacheInfo.note}</p>}
      {cacheInfo?.paths?.length > 0 && (
        <ul className="mt-2 space-y-0.5 text-[11px] text-white/35">
          {cacheInfo.paths.map((p) => (
            <li key={p} className="truncate" title={p}>
              {p}
            </li>
          ))}
        </ul>
      )}

      <div className="mt-3 flex flex-wrap gap-2">
        {buttons.map(({ key, label, primary, onClick }) => {
          const isBusy = busy === key
          const isDone = doneKey === key
          const isDisabled = !!busy && !isBusy
          return (
            <button
              key={key}
              type="button"
              disabled={isDisabled}
              aria-busy={isBusy}
              onClick={onClick}
              className={
                primary
                  ? `${actionButtonClass({ busy: isBusy, done: isDone, disabled: isDisabled })} bg-accent/90 text-white hover:bg-accent border-transparent`
                  : actionButtonClass({ busy: isBusy, done: isDone, disabled: isDisabled })
              }
            >
              {renderButtonLabel(key, label)}
            </button>
          )
        })}
      </div>

      <ActionFeedbackBanner feedback={actionFeedback} onDismiss={() => setActionFeedback(null)} />

      {folderResult && (
        <div className="mt-4">
          <FolderSyncValidationPanel result={folderResult} />
        </div>
      )}
    </section>
  )
}
