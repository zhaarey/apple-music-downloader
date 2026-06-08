import { useMemo } from 'react'
import { parseJobResult, jobStatusMeta, trackStatusIcon } from '../lib/downloadStatus'

export default function QueueTab({ logs, engineEvents, downloading, onCancel, onOpenFolder, jobSession }) {
  const jobResult = useMemo(() => jobSession || parseJobResult(engineEvents), [jobSession, engineEvents])
  const meta = jobResult ? jobStatusMeta(jobResult.phase) : null

  const failedTracks = useMemo(() => {
    return (engineEvents || [])
      .filter((e) => e.type === 'track_failed')
      .slice(-20)
      .map((e) => ({
        label: e.track || e.message,
        detail: e.message,
        phase: e.phase,
      }))
  }, [engineEvents])

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Activity</h2>
          <p className="text-sm text-white/50">
            {downloading ? 'Download in progress…' : 'Recent jobs, track results, and detailed logs'}
          </p>
        </div>
        <div className="flex gap-2">
          {downloading && (
            <button onClick={onCancel} className="rounded-lg border border-red-500/50 px-4 py-2 text-sm text-red-400">
              Cancel
            </button>
          )}
          <button onClick={onOpenFolder} className="rounded-lg bg-surface-raised px-4 py-2 text-sm hover:bg-surface-hover">
            Open output folder
          </button>
        </div>
      </div>

      {jobResult && meta && (
        <div className={`rounded-xl border px-4 py-3 ${meta.className}`}>
          <p className="font-semibold">{meta.label}</p>
          <p className="mt-1 text-sm opacity-90">{jobResult.message}</p>
          {jobResult.total > 0 && (
            <p className="mt-2 text-xs opacity-80">
              {jobResult.success} succeeded · {jobResult.failed} failed/unavailable
            </p>
          )}
        </div>
      )}

      {failedTracks.length > 0 && (
        <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-4">
          <p className="mb-2 text-sm font-medium text-red-200">Recent track issues</p>
          <ul className="max-h-32 space-y-1 overflow-y-auto text-xs">
            {failedTracks.map((t, i) => (
              <li key={i} className="text-red-200/90">
                <span className="text-red-400">{trackStatusIcon(t.phase === 'unavailable' ? 'unavailable' : 'failed')}</span>{' '}
                <span className="font-medium">{t.label}</span>
                {t.detail && t.detail !== t.label && <span className="text-red-200/70"> — {t.detail}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="flex-1 overflow-y-auto rounded-xl border border-white/10 bg-black/30 p-4 font-mono text-xs">
        {logs.length === 0 ? (
          <p className="text-white/40">Activity log will appear here when downloads run.</p>
        ) : (
          logs.map((l, i) => (
            <div key={i} className="border-b border-white/5 py-1 text-white/80">
              <span className="text-white/40">[{l.time}]</span>{' '}
              {l.type === 'error' || l.type === 'track_failed' ? (
                <span className="text-red-400">{l.msg}</span>
              ) : l.type === 'track_complete' ? (
                <span className="text-green-400">{l.msg}</span>
              ) : (
                l.msg
              )}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
