import { useCallback, useState } from 'react'

const MIN_SELECTION_MS = 1000

export function clampTrimRange(startMs, endMs, durationMs) {
  const duration = Math.max(0, durationMs || 0)
  let start = Math.max(0, Math.min(startMs ?? 0, duration))
  let end = Math.max(0, Math.min(endMs ?? duration, duration))
  if (end < start) end = start
  if (end - start < MIN_SELECTION_MS && duration >= MIN_SELECTION_MS) {
    if (start + MIN_SELECTION_MS <= duration) {
      end = start + MIN_SELECTION_MS
    } else {
      end = duration
      start = Math.max(0, duration - MIN_SELECTION_MS)
    }
  }
  return { startMs: start, endMs: end, durationMs: Math.max(0, end - start) }
}

export function useTrimRange() {
  const [startMs, setStartMsState] = useState(0)
  const [endMs, setEndMsState] = useState(0)

  const resetRange = useCallback((durationMs) => {
    const d = Math.max(0, durationMs || 0)
    setStartMsState(0)
    setEndMsState(d)
  }, [])

  const setStartMs = useCallback(
    (value, durationMs) => {
      const next = clampTrimRange(value, endMs, durationMs)
      setStartMsState(next.startMs)
      setEndMsState(next.endMs)
      return next
    },
    [endMs],
  )

  const setEndMs = useCallback(
    (value, durationMs) => {
      const next = clampTrimRange(startMs, value, durationMs)
      setStartMsState(next.startMs)
      setEndMsState(next.endMs)
      return next
    },
    [startMs],
  )

  const nudge = useCallback(
    (field, deltaMs, durationMs) => {
      if (field === 'start') return setStartMs(startMs + deltaMs, durationMs)
      return setEndMs(endMs + deltaMs, durationMs)
    },
    [startMs, endMs, setStartMs, setEndMs],
  )

  const selectionMs = Math.max(0, endMs - startMs)
  const valid = selectionMs >= MIN_SELECTION_MS

  return {
    startMs,
    endMs,
    selectionMs,
    valid,
    minSelectionMs: MIN_SELECTION_MS,
    resetRange,
    setStartMs,
    setEndMs,
    nudge,
  }
}
