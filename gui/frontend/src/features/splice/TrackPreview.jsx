import { useCallback, useEffect, useRef, useState } from 'react'
import { formatMsPrecise } from './spliceTime'

const BOUNDARY_PADDING_MS = 5000

export default function TrackPreview({
  audioURL,
  masterDurationMs,
  trackNumber,
  title,
  startMs,
  endMs,
}) {
  const audioRef = useRef(null)
  const stopTimerRef = useRef(null)
  const [playing, setPlaying] = useState(null)

  const stop = useCallback(() => {
    if (stopTimerRef.current) {
      clearInterval(stopTimerRef.current)
      stopTimerRef.current = null
    }
    const audio = audioRef.current
    if (audio) {
      audio.pause()
    }
    setPlaying(null)
  }, [])

  useEffect(() => () => stop(), [stop])

  useEffect(() => {
    stop()
  }, [audioURL, startMs, endMs, stop])

  const playRange = useCallback(
    (rangeStartMs, rangeEndMs, label) => {
      const audio = audioRef.current
      if (!audio || !audioURL) return

      stop()

      const masterEndMs = masterDurationMs > 0 ? masterDurationMs : rangeEndMs
      const endMsClamped = Math.min(rangeEndMs, masterEndMs)
      const startSec = rangeStartMs / 1000
      const endSec = endMsClamped / 1000
      if (endSec <= startSec) return

      setPlaying(label)

      const begin = () => {
        audio.currentTime = startSec
        audio.play().catch(() => setPlaying(null))
        stopTimerRef.current = setInterval(() => {
          if (audio.currentTime >= endSec - 0.05) stop()
        }, 100)
      }

      if (audio.readyState >= 1) {
        begin()
      } else {
        const onReady = () => begin()
        audio.addEventListener('loadedmetadata', onReady, { once: true })
        audio.load()
      }
    },
    [audioURL, masterDurationMs, stop],
  )

  const playStartCut = () => {
    const pad = BOUNDARY_PADDING_MS
    const start = Math.max(0, startMs - pad)
    const end = Math.min(startMs + pad, endMs)
    playRange(start, end, 'start')
  }

  const playEndCut = () => {
    const pad = BOUNDARY_PADDING_MS
    const start = Math.max(startMs, endMs - pad)
    const end = endMs + pad
    playRange(start, end, 'end')
  }

  const hasSegment = endMs > startMs
  const includeHours = startMs >= 3600000 || endMs >= 3600000
  const startStr = formatMsPrecise(startMs, includeHours)
  const endStr = formatMsPrecise(endMs, includeHours)
  const durStr = formatMsPrecise(endMs - startMs)

  return (
    <div className="rounded-lg border border-white/10 bg-black/20 p-3">
      <audio ref={audioRef} src={audioURL || undefined} preload="metadata" className="hidden" />

      <p className="text-sm text-white/80">
        {hasSegment ? (
          <>
            <span className="font-medium">
              #{trackNumber} {title?.trim() || `Track ${trackNumber}`}
            </span>
            <span className="text-white/55">
              {' '}
              | {startStr} → {endStr} ({durStr})
            </span>
          </>
        ) : (
          'Select a track with a duration to preview cut points.'
        )}
      </p>

      <div className="mt-2 flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={!audioURL || !hasSegment || playing === 'start'}
          onClick={playStartCut}
          className="rounded-lg border border-white/15 px-3 py-1.5 text-xs disabled:opacity-40"
        >
          {playing === 'start' ? 'Playing start…' : 'Play start (±5s)'}
        </button>
        <button
          type="button"
          disabled={!audioURL || !hasSegment || playing === 'end'}
          onClick={playEndCut}
          className="rounded-lg border border-white/15 px-3 py-1.5 text-xs disabled:opacity-40"
        >
          {playing === 'end' ? 'Playing end…' : 'Play end (±5s)'}
        </button>
        <button
          type="button"
          disabled={!playing}
          onClick={stop}
          className="rounded-lg border border-white/15 px-3 py-1.5 text-xs disabled:opacity-40"
        >
          Stop
        </button>
      </div>

      <p className="mt-2 text-xs text-white/45">
        Preview the first and last few seconds around each cut to fine-tune start times and durations.
      </p>
    </div>
  )
}
