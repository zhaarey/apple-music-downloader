const STATUS_META = {
  success: { label: 'Download complete', className: 'border-green-500/40 bg-green-500/10 text-green-200' },
  partial: { label: 'Completed with issues', className: 'border-yellow-500/40 bg-yellow-500/10 text-yellow-100' },
  failed: { label: 'Download failed', className: 'border-red-500/40 bg-red-500/10 text-red-200' },
  cancelled: { label: 'Download cancelled', className: 'border-white/20 bg-white/5 text-white/70' },
}

const PHASE_LABELS = {
  audio: 'Extracting audio',
  video: 'Saving MP4 video',
  postprocess: 'Processing',
  download: 'Downloading',
}

/** Events belonging to the current/last job (from last job_start onward). */
export function sliceEventsForCurrentJob(engineEvents) {
  const events = engineEvents || []
  for (let i = events.length - 1; i >= 0; i--) {
    if (events[i].type === 'job_start') {
      return events.slice(i)
    }
  }
  return events
}

export function parseJobResult(engineEvents) {
  const events = sliceEventsForCurrentJob(engineEvents)
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
    handoff: lastJob.handoff || null,
    masterPath: lastJob.master_path || lastJob.handoff?.master_path || '',
  }
}

function progressPercent(ev) {
  if (!ev?.total) return 0
  return Math.min(100, Math.round((ev.current / ev.total) * 100))
}

function formatProgressDetail(ev) {
  if (!ev) return ''
  const pct = progressPercent(ev)
  const phase = PHASE_LABELS[ev.phase] || PHASE_LABELS.download
  if (pct > 0) return `${phase} · ${pct}%`
  return ev.message || phase
}

export function parseYouTubeProgress(engineEvents) {
  const events = sliceEventsForCurrentJob(engineEvents)
  for (let i = events.length - 1; i >= 0; i--) {
    const ev = events[i]
    if (ev.type === 'progress') {
      return {
        percent: progressPercent(ev),
        message: ev.message || '',
        phase: ev.phase || 'download',
        label: PHASE_LABELS[ev.phase] || PHASE_LABELS.download,
      }
    }
  }
  return null
}

export function parseTrackRows(previewTracks, engineEvents, jobStarted) {
  if (!previewTracks?.length) return []
  const rows = new Map()
  previewTracks.forEach((t) => {
    rows.set(t.num, {
      num: t.num,
      label: t.name,
      sublabel: t.artist,
      status: jobStarted ? 'queued' : 'idle',
      detail: '',
      percent: 0,
    })
  })
  if (!jobStarted) {
    return [...rows.values()].sort((a, b) => a.num - b.num)
  }

  let lastProgress = null
  for (const ev of sliceEventsForCurrentJob(engineEvents)) {
    if (ev.type === 'progress') {
      lastProgress = ev
    }
    if (!ev.current && ev.type !== 'progress') continue
    const key = ev.current || 1
    const existing = rows.get(key) || {
      num: key,
      label: ev.track || ev.message || `Item ${key}`,
      sublabel: '',
      status: 'queued',
      detail: '',
      percent: 0,
    }
    if (ev.type === 'track_start') {
      existing.status = 'downloading'
      existing.label = existing.label || ev.track || existing.label
      existing.detail = ev.message || 'Starting…'
    }
    if (ev.type === 'track_complete') {
      existing.status = ev.phase === 'skipped' ? 'skipped' : 'done'
      existing.detail = ev.message || ''
      existing.percent = 100
      existing.label = ev.track || existing.label
    }
    if (ev.type === 'track_failed') {
      existing.status = ev.phase === 'unavailable' ? 'unavailable' : 'failed'
      existing.detail = ev.message || 'Download failed'
      existing.label = ev.track || existing.label
    }
    if (ev.type === 'progress' && (existing.status === 'downloading' || existing.status === 'queued')) {
      existing.status = 'downloading'
      existing.percent = progressPercent(ev)
      existing.detail = formatProgressDetail(ev)
    }
    rows.set(key, existing)
  }
  if (rows.size === 1 && lastProgress) {
    const only = [...rows.values()][0]
    if (only.status === 'downloading' || only.status === 'queued') {
      only.percent = progressPercent(lastProgress)
      only.detail = formatProgressDetail(lastProgress)
      rows.set(only.num, only)
    }
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
