import { useState } from 'react'
import {
  summarizeTrackRows,
  trackStatusIcon,
  trackStatusClass,
} from '../../lib/downloadStatus'

export default function TrackStatusPanel({
  trackRows,
  visible = true,
  downloading = false,
  youtubeProgress = null,
  title = 'Track status',
}) {
  const [expanded, setExpanded] = useState(false)

  if (!visible || !trackRows?.length) return null

  const summary = summarizeTrackRows(trackRows)
  const allOk = summary.failed === 0 && summary.unavailable === 0 && summary.downloading === 0 && summary.queued === 0
  const showCollapsed = allOk && !downloading && !expanded

  const overallPct = youtubeProgress?.percent ?? trackRows.find((r) => r.status === 'downloading')?.percent ?? 0
  const overallLabel =
    youtubeProgress?.label || trackRows.find((r) => r.status === 'downloading')?.detail || 'Working…'

  return (
    <div className="rounded-xl border border-white/10 bg-black/20 p-4">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm font-medium">{title}</p>
        <div className="flex flex-wrap items-center gap-2 text-xs">
          {summary.done > 0 && (
            <span className="rounded-full bg-green-500/10 px-2 py-0.5 text-green-300">{summary.done} done</span>
          )}
          {summary.skipped > 0 && (
            <span className="rounded-full bg-white/5 px-2 py-0.5 text-white/50">{summary.skipped} skipped</span>
          )}
          {(summary.failed > 0 || summary.unavailable > 0) && (
            <span className="rounded-full bg-red-500/10 px-2 py-0.5 text-red-300">
              {summary.failed + summary.unavailable} failed
            </span>
          )}
        </div>
      </div>

      {downloading && youtubeProgress != null && (
        <div className="mb-3 rounded-lg border border-accent/25 bg-accent/5 p-3">
          <div className="mb-2 flex items-center justify-between gap-3">
            <p className="text-sm font-medium text-white/90">{overallLabel}</p>
            <span className="text-sm tabular-nums text-accent">{overallPct > 0 ? `${overallPct}%` : '…'}</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-black/30">
            <div
              className="h-full rounded-full bg-accent transition-all duration-300"
              style={{ width: `${Math.max(overallPct, downloading && overallPct === 0 ? 8 : 0)}%` }}
            />
          </div>
          {youtubeProgress?.message?.startsWith('[download]') && (
            <p className="mt-2 truncate text-xs text-white/45">{youtubeProgress.message}</p>
          )}
        </div>
      )}

      {showCollapsed ? (
        <button
          type="button"
          onClick={() => setExpanded(true)}
          className="text-xs text-white/45 hover:text-white/70"
        >
          Show details ({summary.total} track{summary.total !== 1 ? 's' : ''})
        </button>
      ) : (
        <>
          {allOk && !downloading && (
            <button
              type="button"
              onClick={() => setExpanded(false)}
              className="mb-2 text-xs text-white/45 hover:text-white/70"
            >
              Hide details
            </button>
          )}
          <ul className="max-h-48 space-y-2 overflow-y-auto text-xs">
            {trackRows.map((r) => (
              <li key={r.num} className="rounded-lg bg-white/[0.02] px-2 py-1.5">
                <div className="flex items-start gap-2">
                  <span className={`mt-0.5 w-4 shrink-0 ${trackStatusClass(r.status)}`}>
                    {trackStatusIcon(r.status)}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-white/90">{r.label}</p>
                    {r.sublabel && <p className="truncate text-[11px] text-white/40">{r.sublabel}</p>}
                    {r.detail && <p className={`mt-0.5 ${trackStatusClass(r.status)}`}>{r.detail}</p>}
                    {r.status === 'downloading' && r.percent > 0 && !youtubeProgress && (
                      <div className="mt-1.5 h-1 overflow-hidden rounded-full bg-black/30">
                        <div
                          className="h-full rounded-full bg-accent transition-all"
                          style={{ width: `${r.percent}%` }}
                        />
                      </div>
                    )}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  )
}
