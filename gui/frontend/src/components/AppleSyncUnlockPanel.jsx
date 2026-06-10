import { useState } from 'react'
import { ReleaseAppleSyncLock, OpenAppleSyncUnlockLog } from '../wailsjs/go/main/App'

const MIN_ACTION_MS = 450

async function withMinDelay(promise, ms = MIN_ACTION_MS) {
  const [result] = await Promise.all([promise, new Promise((resolve) => setTimeout(resolve, ms))])
  return result
}

export default function AppleSyncUnlockPanel({ compact = false, className = '' }) {
  const [busy, setBusy] = useState('')
  const [result, setResult] = useState(null)
  const [feedback, setFeedback] = useState(null)

  const runUnlock = async (restartService, elevated, key) => {
    setBusy(key)
    setFeedback(null)
    try {
      const res = await withMinDelay(ReleaseAppleSyncLock(restartService, elevated))
      setResult(res)
      if (res?.ok) {
        setFeedback({
          variant: res.need_elevated ? 'info' : 'success',
          title: res.summary || 'Sync agents released',
          detail: res.killed_hint || res.message,
        })
      } else {
        setFeedback({
          variant: 'error',
          title: res?.summary || 'Sync unlock failed',
          detail: res?.message || 'Unknown error',
        })
      }
    } catch (err) {
      setFeedback({
        variant: 'error',
        title: 'Sync unlock failed',
        detail: String(err?.message || err),
      })
    } finally {
      setBusy('')
    }
  }

  const buttonClass = (active, disabled) =>
    `rounded-lg border px-3 py-2 text-sm transition ${
      active
        ? 'border-accent/40 bg-accent/10 text-white'
        : disabled
          ? 'border-white/5 text-white/30'
          : 'border-white/15 hover:bg-white/5'
    }`

  return (
    <div className={className}>
      {!compact && (
        <>
          <p className="text-xs leading-relaxed text-white/50">
            After syncing in Apple Devices, artwork sometimes stays wrong until Windows releases a stuck background
            agent (<code className="text-white/60">AMPDevicesAgent</code>). This does the same thing as canceling a
            restart — without rebooting.
          </p>
          <p className="mt-2 text-[11px] text-white/35">
            Run only when sync finished or is stuck. Does not change your .m4a files.
          </p>
        </>
      )}

      <div className={`flex flex-wrap gap-2 ${compact ? '' : 'mt-3'}`}>
        <button
          type="button"
          disabled={!!busy}
          onClick={() => runUnlock(false, false, 'unlock')}
          className={buttonClass(busy === 'unlock', !!busy && busy !== 'unlock')}
        >
          {busy === 'unlock' ? 'Releasing…' : 'Release sync lock'}
        </button>
        <button
          type="button"
          disabled={!!busy}
          onClick={() => runUnlock(true, false, 'unlock-svc')}
          className={buttonClass(busy === 'unlock-svc', !!busy && busy !== 'unlock-svc')}
          title="Also restarts Apple Mobile Device Service (USB stack)"
        >
          {busy === 'unlock-svc' ? 'Releasing…' : 'Release + restart USB service'}
        </button>
        {result?.need_elevated && (
          <button
            type="button"
            disabled={!!busy}
            onClick={() => runUnlock(true, true, 'unlock-admin')}
            className={buttonClass(busy === 'unlock-admin', !!busy && busy !== 'unlock-admin')}
          >
            {busy === 'unlock-admin' ? 'Requesting…' : 'Run as administrator'}
          </button>
        )}
        {result?.log_path && (
          <button
            type="button"
            disabled={!!busy}
            onClick={() => OpenAppleSyncUnlockLog().catch(() => {})}
            className={buttonClass(false, !!busy)}
          >
            Open log
          </button>
        )}
      </div>

      {feedback && (
        <div
          role="status"
          className={`mt-3 rounded-lg border px-3 py-2 text-xs ${
            feedback.variant === 'success'
              ? 'border-green-500/30 bg-green-500/10 text-green-100'
              : feedback.variant === 'error'
                ? 'border-red-500/30 bg-red-500/10 text-red-100'
                : 'border-sky-500/30 bg-sky-500/10 text-sky-100'
          }`}
        >
          <p className="font-medium">{feedback.title}</p>
          {feedback.detail ? <p className="mt-1 text-white/70">{feedback.detail}</p> : null}
        </div>
      )}

      {!compact && result?.manual_steps?.length > 0 && (
        <ul className="mt-3 list-inside list-disc space-y-1 text-[11px] text-white/40">
          {result.manual_steps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ul>
      )}
    </div>
  )
}
