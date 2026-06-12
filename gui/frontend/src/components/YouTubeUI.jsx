function FetchPreviewSkeleton({ step, youtubeMode }) {
  const steps = youtubeMode
    ? ['Connecting to YouTube…', 'Reading video metadata…', 'Building preview…']
    : ['Connecting to Apple Music…', 'Loading catalog metadata…', 'Building preview…']
  return (
    <div className="rounded-xl border border-accent/20 bg-surface-raised p-4">
      <div className="flex gap-4 animate-pulse">
        <div className="h-24 w-24 shrink-0 rounded-lg bg-white/10" />
        <div className="min-w-0 flex-1 space-y-3">
          <div className="h-3 w-24 rounded bg-white/10" />
          <div className="h-5 w-3/4 rounded bg-white/10" />
          <div className="h-4 w-1/2 rounded bg-white/10" />
          <div className="h-3 w-32 rounded bg-white/10" />
        </div>
      </div>
      <div className="mt-4 flex items-center gap-2 text-sm text-accent">
        <svg className="h-4 w-4 animate-spin" viewBox="0 0 16 16" fill="none" aria-hidden>
          <circle cx="8" cy="8" r="5.5" stroke="currentColor" strokeOpacity="0.25" strokeWidth="1.5" />
          <path d="M8 2.5a5.5 5.5 0 0 1 5.5 5.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </svg>
        <span>{steps[step % steps.length]}</span>
      </div>
      <p className="mt-2 text-xs text-white/45">This can take a few seconds for long playlists or slow connections.</p>
    </div>
  )
}

function FetchStatusBanner({ message, variant = 'info' }) {
  if (!message) return null
  const styles = {
    info: 'border-accent/25 bg-accent/10 text-white/80',
    error: 'border-red-400/30 bg-red-500/10 text-red-200',
    success: 'border-green-400/25 bg-green-500/10 text-green-100',
  }
  return (
    <div
      role="status"
      aria-live="polite"
      className={`rounded-xl border px-4 py-3 text-sm ${styles[variant] || styles.info}`}
    >
      {message}
    </div>
  )
}

function YouTubeFetchSkeleton({ step }) {
  return <FetchPreviewSkeleton step={step} youtubeMode />
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

export { FetchPreviewSkeleton, FetchStatusBanner, YouTubeFetchSkeleton, YouTubeProgressPanel }
