function YouTubeFetchSkeleton({ step }) {
  const steps = ['Connecting to YouTube…', 'Reading metadata…', 'Building preview…']
  return (
    <div className="rounded-xl border border-white/10 bg-surface-raised p-4 animate-pulse">
      <div className="flex gap-4">
        <div className="h-24 w-24 shrink-0 rounded-lg bg-white/10" />
        <div className="min-w-0 flex-1 space-y-3">
          <div className="h-3 w-24 rounded bg-white/10" />
          <div className="h-5 w-3/4 rounded bg-white/10" />
          <div className="h-4 w-1/2 rounded bg-white/10" />
          <div className="h-3 w-32 rounded bg-white/10" />
        </div>
      </div>
      <p className="mt-4 text-sm text-accent">{steps[step % steps.length]}</p>
    </div>
  )
}

function YouTubeProgressPanel({ progress, downloading, trackRows }) {
  if (!downloading) return null
  const pct = progress?.percent ?? trackRows.find((r) => r.status === 'downloading')?.percent ?? 0
  const label = progress?.label || trackRows.find((r) => r.status === 'downloading')?.detail || 'Working…'
  return (
    <div className="rounded-xl border border-accent/30 bg-accent/5 p-4">
      <div className="mb-2 flex items-center justify-between gap-3">
        <p className="text-sm font-medium text-white/90">{label}</p>
        <span className="text-sm tabular-nums text-accent">{pct > 0 ? `${pct}%` : '…'}</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-black/30">
        <div
          className="h-full rounded-full bg-accent transition-all duration-300"
          style={{ width: `${Math.max(pct, downloading && pct === 0 ? 8 : 0)}%` }}
        />
      </div>
      {progress?.message?.startsWith('[download]') && (
        <p className="mt-2 truncate text-xs text-white/45">{progress.message}</p>
      )}
    </div>
  )
}

export { YouTubeFetchSkeleton, YouTubeProgressPanel }
