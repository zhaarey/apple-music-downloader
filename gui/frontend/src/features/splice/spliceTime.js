const MIN_TRACK_MS = 1000

/** Format milliseconds as m:ss or h:mm:ss (no fractional part). */
export function formatMs(ms) {
  if (!ms) return '0:00'
  const s = Math.floor(ms / 1000)
  const m = Math.floor(s / 60)
  const h = Math.floor(m / 60)
  if (h > 0) return `${h}:${String(m % 60).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`
  return `${m}:${String(s % 60).padStart(2, '0')}`
}

/** Format milliseconds with sub-second precision, e.g. 12:34.560 */
export function formatMsPrecise(ms, includeHours = false) {
  const value = Math.max(0, ms || 0)
  const totalSeconds = Math.floor(value / 1000)
  const remainderMs = value % 1000
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  if (hours > 0 || includeHours) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}.${String(remainderMs).padStart(3, '0')}`
  }
  return `${minutes}:${String(seconds).padStart(2, '0')}.${String(remainderMs).padStart(3, '0')}`
}

/** Parse m:ss, h:mm:ss, mm:ss.ms, or plain seconds to milliseconds. */
export function parseTimeInput(value) {
  const text = String(value || '').trim()
  if (!text) return 0

  let base = text
  let fractionalMs = 0
  const dotIdx = text.lastIndexOf('.')
  if (dotIdx >= 0) {
    const frac = text.slice(dotIdx + 1)
    base = text.slice(0, dotIdx)
    if (/^\d+$/.test(frac)) {
      const padded = frac.padEnd(3, '0').slice(0, 3)
      fractionalMs = Number(padded)
    }
  }

  const parts = base.split(':').map((p) => p.trim()).filter(Boolean)
  if (parts.some((p) => Number.isNaN(Number(p)))) return null

  let seconds = 0
  if (parts.length === 1) {
    seconds = Number(parts[0])
  } else if (parts.length === 2) {
    seconds = Number(parts[0]) * 60 + Number(parts[1])
  } else if (parts.length === 3) {
    seconds = Number(parts[0]) * 3600 + Number(parts[1]) * 60 + Number(parts[2])
  } else {
    return null
  }

  return Math.max(0, Math.round(seconds * 1000) + fractionalMs)
}

function resolveStarts(tracks, masterDurationMs) {
  const n = tracks.length
  const starts = new Array(n).fill(0)
  starts[0] = tracks[0]?.start_ms != null ? Math.max(0, tracks[0].start_ms) : 0
  for (let i = 1; i < n; i++) {
    if (tracks[i].start_ms != null) {
      starts[i] = Math.max(0, tracks[i].start_ms)
    } else {
      starts[i] = starts[i - 1] + (tracks[i - 1].duration_ms || 0)
    }
  }
  if (masterDurationMs > 0) {
    for (let i = 1; i < n; i++) {
      const maxPos = masterDurationMs - MIN_TRACK_MS * (n - i)
      if (starts[i] > maxPos) starts[i] = maxPos
      const minPos = starts[i - 1] + MIN_TRACK_MS
      if (starts[i] < minPos) starts[i] = minPos
    }
  }
  return starts
}

/** Returns { startMs, endMs, durationMs } for a track index. */
export function computeTrackSegment(tracks, masterDurationMs, trackIdx) {
  if (!tracks?.length || trackIdx < 0 || trackIdx >= tracks.length) return null
  const starts = resolveStarts(tracks, masterDurationMs)
  const startMs = starts[trackIdx]
  let endMs
  if (trackIdx < tracks.length - 1) {
    endMs = starts[trackIdx + 1]
  } else if (masterDurationMs > 0) {
    endMs = masterDurationMs
  } else {
    endMs = startMs + (tracks[trackIdx].duration_ms || 0)
  }
  const durationMs = Math.max(0, endMs - startMs)
  return { startMs, endMs, durationMs }
}
