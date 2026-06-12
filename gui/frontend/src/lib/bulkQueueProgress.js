import { sliceEventsForCurrentJob } from './downloadStatus'

/** Latest bulk queue position from engine progress events (phase === "queue"). */
export function parseBulkQueueProgress(engineEvents) {
  const events = sliceEventsForCurrentJob(engineEvents)
  for (let i = events.length - 1; i >= 0; i--) {
    const ev = events[i]
    if (ev.type === 'progress' && ev.phase === 'queue') {
      return {
        current: ev.current ?? 0,
        total: ev.total ?? 0,
        label: ev.message || '',
        url: ev.track || '',
      }
    }
  }
  return null
}
