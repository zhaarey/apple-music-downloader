import { useState } from 'react'
import { ResetAppleSyncStack, OpenAppleSyncResetLog } from '../wailsjs/go/main/App'

const MIN_ACTION_MS = 450

async function withMinDelay(promise, ms = MIN_ACTION_MS) {
  const [result] = await Promise.all([promise, new Promise((resolve) => setTimeout(resolve, ms))])
  return result
}

export default function AppleSyncResetPanel({ compact = false, className = '' }) {
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState(null)
  const [feedback, setFeedback] = useState(null)

  const runReset = async (elevated = false) => {
    setBusy(true)
    setFeedback(null)
    try {
      const res = await withMinDelay(ResetAppleSyncStack(elevated))
      setResult(res)
      if (res?.ok) {
        setFeedback({
          variant: res.need_elevated ? 'info' : 'success',
          title: res.summary || 'Sync stack reset',
          detail: res.stopped_hint || res.message,
        })
      } else {
        setFeedback({
          variant: 'error',
          title: res?.summary || 'Sync reset failed',
          detail: res?.message || 'Unknown error',
        })
      }
    } catch (err) {
      setFeedback({
        variant: 'error',
        title: 'Sync reset failed',
        detail: String(err?.message || err),
      })
    } finally {
      setBusy(false)
    }
  }

  const buttonClass = (primary, disabled) =>
    `rounded-lg border px-3 py-2 text-sm transition ${
      primary
        ? disabled
          ? 'border-accent/20 bg-accent/5 text-white/40'
          : 'border-accent/40 bg-accent/10 text-white hover:bg-accent/15'
        : disabled
          ? 'border-white/5 text-white/30'
          : 'border-white/15 hover:bg-white/5'
    }`

  return (
    <div className={className}>
      {!compact && (
        <>
          <p className="text-xs leading-relaxed text-white/50">
            Stops Apple Music, Apple Devices sync agents, and restarts the USB device service — the same process-kill
            effect as canceling a Windows restart. Does <strong className="font-medium text-white/70">not</strong> delete
            artwork caches or change your .m4a files.
          </p>
          <p className="mt-2 text-[11px] text-white/35">
            Run after a sync finishes or is stuck, not while files are still copying.
          </p>
        </>
      )}

      <div className={`flex flex-wrap gap-2 ${compact ? '' : 'mt-3'}`}>
        <button
          type="button"
          disabled={busy}
          onClick={() => runReset(false)}
          className={buttonClass(true, busy)}
        >
          {busy ? 'Resetting…' : 'Reset Apple sync'}
        </button>
        {result?.need_elevated && (
          <button
            type="button"
            disabled={busy}
            onClick={() => runReset(true)}
            className={buttonClass(false, busy)}
          >
            Run as administrator
          </button>
        )}
        {result?.log_path && (
          <button type="button" disabled={busy} onClick={() => OpenAppleSyncResetLog().catch(() => {})} className={buttonClass(false, busy)}>
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
