const STATUS_META = {
  success: { label: 'Download complete', className: 'border-green-500/40 bg-green-500/10 text-green-200' },
  partial: { label: 'Completed with issues', className: 'border-yellow-500/40 bg-yellow-500/10 text-yellow-100' },
  failed: { label: 'Download failed', className: 'border-red-500/40 bg-red-500/10 text-red-200' },
  cancelled: { label: 'Download cancelled', className: 'border-white/20 bg-white/5 text-white/70' },
}

export function parseJobResult(engineEvents) {
  const events = engineEvents || []
  const lastJob = [...events].reverse().find((e) => e.type === 'job_complete')
  if (!lastJob) return null
  const recentErrors = events.filter((e) => e.type === 'error').slice(-3).map((e) => e.message)
  return {
    phase: lastJob.phase || 'success',
    message: lastJob.message,
    success: lastJob.success ?? 0,
    failed: lastJob.error ?? 0,
    total: lastJob.total_count ?? 0,
    recentErrors,
  }
}

export function parseTrackRows(previewTracks, engineEvents, jobStarted) {
  if (!previewTracks?.length) return []
  const rows = new Map()
  previewTracks.forEach((t) => {
    rows.set(t.num, {
      num: t.num,
      label: `${t.name} — ${t.artist}`,
      status: jobStarted ? 'queued' : 'idle',
      detail: '',
    })
  })
  for (const ev of engineEvents || []) {
    if (!ev.current) continue
    const existing = rows.get(ev.current) || {
      num: ev.current,
      label: ev.track || ev.message || `Track ${ev.current}`,
      status: 'queued',
      detail: '',
    }
    if (ev.type === 'track_start') {
      existing.status = 'downloading'
      existing.label = ev.track || existing.label
    }
    if (ev.type === 'track_complete') {
      existing.status = ev.phase === 'skipped' ? 'skipped' : 'done'
      existing.detail = ev.message || ''
      existing.label = ev.track || existing.label
    }
    if (ev.type === 'track_failed') {
      existing.status = ev.phase === 'unavailable' ? 'unavailable' : 'failed'
      existing.detail = ev.message || 'Download failed'
      existing.label = ev.track || existing.label
    }
    rows.set(ev.current, existing)
  }
  return [...rows.values()].sort((a, b) => a.num - b.num)
}

export function jobStatusMeta(phase) {
  return STATUS_META[phase] || STATUS_META.failed
}

export function trackStatusIcon(status) {
  switch (status) {
    case 'done':
      return '✓'
    case 'skipped':
      return '↷'
    case 'downloading':
      return '↻'
    case 'failed':
      return '✗'
    case 'unavailable':
      return '−'
    default:
      return '○'
  }
}

export function trackStatusClass(status) {
  switch (status) {
    case 'done':
      return 'text-green-400'
    case 'skipped':
      return 'text-white/50'
    case 'downloading':
      return 'text-accent'
    case 'failed':
      return 'text-red-400'
    case 'unavailable':
      return 'text-yellow-400'
    default:
      return 'text-white/40'
  }
}
