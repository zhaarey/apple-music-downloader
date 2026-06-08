import { useCallback, useEffect, useRef, useState } from 'react'

export default function WaveformEditor({ peaks, durationMs, boundaries, selectedBoundary, onBoundaryChange, onSelectBoundary }) {
  const canvasRef = useRef(null)
  const [dragging, setDragging] = useState(null)

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || !peaks?.bins?.length) return
    const ctx = canvas.getContext('2d')
    const w = canvas.width
    const h = canvas.height
    ctx.clearRect(0, 0, w, h)
    ctx.fillStyle = 'rgba(255,255,255,0.04)'
    ctx.fillRect(0, 0, w, h)

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

    if (durationMs > 0 && boundaries?.length) {
      boundaries.forEach((ms, idx) => {
        if (idx === 0) return
        const x = (ms / durationMs) * w
        const active = selectedBoundary === idx
        const lineWidth = active ? 4 : 3
        const handleHalf = active ? 7 : 5

        // Thick boundary line.
        ctx.strokeStyle = active ? '#a78bfa' : 'rgba(255,255,255,0.65)'
        ctx.lineWidth = lineWidth
        ctx.beginPath()
        ctx.moveTo(x, 0)
        ctx.lineTo(x, h)
        ctx.stroke()

        // Grab handle cap at top for better visibility.
        ctx.fillStyle = active ? '#a78bfa' : 'rgba(255,255,255,0.85)'
        ctx.fillRect(x - handleHalf, 0, handleHalf * 2, 12)
      })
    }
  }, [peaks, durationMs, boundaries, selectedBoundary])

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

  const boundaryNear = (ms) => {
    if (!durationMs) return -1
    const canvas = canvasRef.current
    const w = canvas?.width || 1
    const threshold = (24 / w) * durationMs
    for (let i = 1; i < (boundaries?.length || 0); i++) {
      if (Math.abs(boundaries[i] - ms) <= threshold) return i
    }
    return -1
  }

  const onPointerDown = (e) => {
    if (!durationMs) return
    const ms = msFromEvent(e)
    const idx = boundaryNear(ms)
    if (idx >= 0) {
      setDragging(idx)
      onSelectBoundary?.(idx)
    }
  }

  const onPointerMove = (e) => {
    if (dragging == null) return
    const ms = msFromEvent(e)
    onBoundaryChange?.(dragging, ms)
  }

  const onPointerUp = () => setDragging(null)

  if (!peaks?.bins?.length) {
    return (
      <div className="flex h-28 items-center justify-center rounded-xl border border-white/10 bg-black/20 text-sm text-white/40">
        Load a master file to show waveform
      </div>
    )
  }

  return (
    <canvas
      ref={canvasRef}
      width={1200}
      height={140}
      className="h-28 w-full cursor-crosshair rounded-xl border border-white/10 bg-black/30"
      onMouseDown={onPointerDown}
      onMouseMove={onPointerMove}
      onMouseUp={onPointerUp}
      onMouseLeave={onPointerUp}
    />
  )
}
