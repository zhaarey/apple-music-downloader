import { useCallback, useEffect, useRef, useState } from 'react'

export default function TrimRangeEditor({
  peaks,
  durationMs,
  startMs,
  endMs,
  onStartChange,
  onEndChange,
  onSeek,
  disabled = false,
}) {
  const canvasRef = useRef(null)
  const [dragging, setDragging] = useState(null)

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || !peaks?.bins?.length || !durationMs) return
    const ctx = canvas.getContext('2d')
    const w = canvas.width
    const h = canvas.height
    ctx.clearRect(0, 0, w, h)

    const startX = (startMs / durationMs) * w
    const endX = (endMs / durationMs) * w

    ctx.fillStyle = 'rgba(0,0,0,0.35)'
    ctx.fillRect(0, 0, startX, h)
    ctx.fillRect(endX, 0, w - endX, h)

    ctx.fillStyle = 'rgba(99,102,241,0.08)'
    ctx.fillRect(startX, 0, endX - startX, h)

    const bins = peaks.bins
    const mid = h / 2
    ctx.strokeStyle = 'rgba(99,102,241,0.85)'
    ctx.lineWidth = 1
    ctx.beginPath()
    for (let i = 0; i < bins.length; i++) {
      const x = (i / bins.length) * w
      const amp = Math.max(Math.abs(bins[i].min), Math.abs(bins[i].max))
      const barH = amp * mid * 0.92
      ctx.moveTo(x, mid - barH)
      ctx.lineTo(x, mid + barH)
    }
    ctx.stroke()

    const handles = [
      { ms: startMs, color: '#34d399', label: 'Start' },
      { ms: endMs, color: '#f87171', label: 'End' },
    ]
    handles.forEach(({ ms, color }) => {
      const x = (ms / durationMs) * w
      ctx.strokeStyle = color
      ctx.lineWidth = 3
      ctx.beginPath()
      ctx.moveTo(x, 0)
      ctx.lineTo(x, h)
      ctx.stroke()
      ctx.fillStyle = color
      ctx.fillRect(x - 6, 0, 12, 14)
    })
  }, [peaks, durationMs, startMs, endMs])

  useEffect(() => {
    draw()
  }, [draw])

  const msFromEvent = (e) => {
    const canvas = canvasRef.current
    const rect = canvas.getBoundingClientRect()
    const x = e.clientX - rect.left
    const ratio = Math.max(0, Math.min(1, x / rect.width))
    return Math.round(ratio * durationMs)
  }

  const handleNear = (ms) => {
    if (!durationMs) return null
    const canvas = canvasRef.current
    const w = canvas?.width || 1
    const threshold = (20 / w) * durationMs
    if (Math.abs(startMs - ms) <= threshold) return 'start'
    if (Math.abs(endMs - ms) <= threshold) return 'end'
    return null
  }

  const onPointerDown = (e) => {
    if (disabled || !durationMs) return
    const ms = msFromEvent(e)
    const near = handleNear(ms)
    if (near) {
      setDragging(near)
      return
    }
    onSeek?.(ms)
  }

  const onPointerMove = (e) => {
    if (!dragging || disabled) return
    const ms = msFromEvent(e)
    if (dragging === 'start') onStartChange?.(ms)
    else onEndChange?.(ms)
  }

  const onPointerUp = () => setDragging(null)

  if (!peaks?.bins?.length) {
    return (
      <div className="flex h-28 items-center justify-center rounded-xl border border-white/10 bg-black/20 text-sm text-white/40">
        Waveform loads after you open a file
      </div>
    )
  }

  return (
    <canvas
      ref={canvasRef}
      width={1200}
      height={140}
      className={`h-28 w-full rounded-xl border border-white/10 bg-black/30 ${disabled ? 'opacity-50' : 'cursor-crosshair'}`}
      onMouseDown={onPointerDown}
      onMouseMove={onPointerMove}
      onMouseUp={onPointerUp}
      onMouseLeave={onPointerUp}
    />
  )
}
