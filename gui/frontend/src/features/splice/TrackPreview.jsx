import { useCallback, useEffect, useRef, useState } from 'react'
import { formatMsPrecise } from './spliceTime'
import { useMasterAudio } from './useMasterAudio'

const BOUNDARY_PADDING_MS = 5000

export default function TrackPreview({
  audioURL,
  masterDurationMs,
  trackNumber,
  title,
  startMs,
  endMs,
}) {
  const { audioRef, status, error, isReady } = useMasterAudio(audioURL)
  const stopTimerRef = useRef(null)
  const stopAtSecRef = useRef(0)
  const [playing, setPlaying] = useState(null)
  const [scrubMs, setScrubMs] = useState(0)
  const [displayMs, setDisplayMs] = useState(0)

  const hasSegment = endMs > startMs
  const segmentDurationMs = hasSegment ? endMs - startMs : 0
  const includeHours = startMs >= 3600000 || endMs >= 3600000

  const stop = useCallback(() => {
    if (stopTimerRef.current) {
      clearInterval(stopTimerRef.current)
      stopTimerRef.current = null
    }
    stopAtSecRef.current = 0
    const audio = audioRef.current
    if (audio) audio.pause()
    setPlaying(null)
  }, [audioRef])

  useEffect(() => () => stop(), [stop])

  useEffect(() => {
    stop()
    setScrubMs(0)
    setDisplayMs(startMs)
  }, [audioURL, startMs, endMs, stop])

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) return undefined

    const onTimeUpdate = () => {
      const posMs = Math.round(audio.currentTime * 1000)
      setDisplayMs(posMs)
      if (hasSegment && posMs >= startMs && posMs <= endMs) {
        setScrubMs(posMs - startMs)
      }
      if (stopAtSecRef.current > 0 && audio.currentTime >= stopAtSecRef.current - 0.03) {
        stop()
      }
    }

    audio.addEventListener('timeupdate', onTimeUpdate)
    return () => audio.removeEventListener('timeupdate', onTimeUpdate)
  }, [audioRef, hasSegment, startMs, endMs, stop])

  const waitForReady = useCallback(async () => {
    const audio = audioRef.current
    if (!audio || !isReady) return false
    if (audio.readyState >= 1) return true
    return new Promise((resolve) => {
      const onReady = () => {
        audio.removeEventListener('loadedmetadata', onReady)
        resolve(true)
      }
      audio.addEventListener('loadedmetadata', onReady)
    })
  }, [audioRef, isReady])

  const playRange = useCallback(
    async (rangeStartMs, rangeEndMs, label) => {
      const audio = audioRef.current
      if (!audio || !isReady || !hasSegment) return

      stop()

      const masterEndMs = masterDurationMs > 0 ? masterDurationMs : rangeEndMs
      const endMsClamped = Math.min(rangeEndMs, masterEndMs)
      const startSec = rangeStartMs / 1000
      const endSec = endMsClamped / 1000
      if (endSec <= startSec) return

      const ready = await waitForReady()
      if (!ready) return

      setPlaying(label)
      stopAtSecRef.current = endSec
      setDisplayMs(rangeStartMs)
      setScrubMs(Math.max(0, Math.min(segmentDurationMs, rangeStartMs - startMs)))

      try {
        audio.currentTime = startSec
        await audio.play()
      } catch {
        stop()
      }
    },
    [audioRef, hasSegment, isReady, masterDurationMs, segmentDurationMs, startMs, stop, waitForReady],
  )

  const playStartCut = () => {
    const pad = BOUNDARY_PADDING_MS
    playRange(Math.max(0, startMs - pad), Math.min(startMs + pad, endMs), 'start')
  }

  const playEndCut = () => {
    const pad = BOUNDARY_PADDING_MS
    playRange(Math.max(startMs, endMs - pad), endMs + pad, 'end')
  }

  const playFromScrub = () => {
    playRange(startMs + scrubMs, endMs, 'scrub')
  }

  const playFullSegment = () => {
    playRange(startMs, endMs, 'full')
  }

  const handleScrub = (relativeMs) => {
    const next = Math.max(0, Math.min(segmentDurationMs, relativeMs))
    setScrubMs(next)
    setDisplayMs(startMs + next)
    const audio = audioRef.current
    if (audio && isReady) {
      audio.pause()
      setPlaying(null)
      stopAtSecRef.current = 0
      audio.currentTime = (startMs + next) / 1000
    }
  }

  const startStr = formatMsPrecise(startMs, includeHours)
  const endStr = formatMsPrecise(endMs, includeHours)
  const durStr = formatMsPrecise(segmentDurationMs)
  const scrubAbsoluteStr = formatMsPrecise(displayMs, includeHours)

  return (
    <div className="rounded-lg border border-white/10 bg-black/20 p-3">
      <audio ref={audioRef} preload="auto" className="hidden" />

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

      {status === 'loading' && hasSegment && (
        <p className="mt-2 text-xs text-white/45">Loading master audio for preview…</p>
      )}
      {error && (
        <p className="mt-2 rounded-lg border border-red-400/30 bg-red-500/10 px-3 py-2 text-xs text-red-200">
          {error}
        </p>
      )}

      <div className="mt-3 space-y-2">
        <div className="flex items-center gap-3">
          <input
            type="range"
            min={0}
            max={Math.max(segmentDurationMs, 1)}
            step={10}
            value={Math.min(scrubMs, segmentDurationMs)}
            disabled={!isReady || !hasSegment}
            onChange={(e) => handleScrub(Number(e.target.value))}
            className="h-2 flex-1 cursor-pointer accent-accent disabled:opacity-40"
            aria-label="Preview position within track"
          />
          <span className="min-w-[9rem] text-right text-xs tabular-nums text-white/60">
            {formatMsPrecise(scrubMs)} / {durStr}
          </span>
        </div>
        <p className="text-xs text-white/45">
          Scrub to any point, then play from there. Position: {scrubAbsoluteStr}
        </p>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={!isReady || !hasSegment}
          onClick={playFromScrub}
          className="rounded-lg bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-muted disabled:opacity-40"
        >
          {playing === 'scrub' ? 'Playing…' : 'Play from here'}
        </button>
        <button
          type="button"
          disabled={!isReady || !hasSegment || playing === 'start'}
          onClick={playStartCut}
          className="rounded-lg border border-white/15 px-3 py-1.5 text-xs disabled:opacity-40"
        >
          {playing === 'start' ? 'Playing start…' : 'Play start (±5s)'}
        </button>
        <button
          type="button"
          disabled={!isReady || !hasSegment || playing === 'end'}
          onClick={playEndCut}
          className="rounded-lg border border-white/15 px-3 py-1.5 text-xs disabled:opacity-40"
        >
          {playing === 'end' ? 'Playing end…' : 'Play end (±5s)'}
        </button>
        <button
          type="button"
          disabled={!isReady || !hasSegment}
          onClick={playFullSegment}
          className="rounded-lg border border-white/15 px-3 py-1.5 text-xs disabled:opacity-40"
        >
          {playing === 'full' ? 'Playing track…' : 'Play full track'}
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
    </div>
  )
}
