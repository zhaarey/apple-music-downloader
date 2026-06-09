import { useEffect } from 'react'

const VARIANTS = {
  success: 'border-green-400/25 bg-green-500/15 text-green-100',
  info: 'border-white/20 bg-white/10 text-white/85',
  error: 'border-red-400/30 bg-red-500/12 text-red-200',
}

export default function StatusToast({ message, variant = 'success', onDismiss, duration = 2800 }) {
  useEffect(() => {
    if (!message) return undefined
    const t = setTimeout(() => onDismiss?.(), duration)
    return () => clearTimeout(t)
  }, [message, duration, onDismiss])

  if (!message) return null

  return (
    <div className="pointer-events-none absolute inset-x-0 top-3 z-20 flex justify-center px-4">
      <div
        role="status"
        aria-live="polite"
        className={`animate-status-in rounded-full border px-4 py-2 text-sm shadow-lg backdrop-blur-md ${VARIANTS[variant] || VARIANTS.success}`}
      >
        {message}
      </div>
    </div>
  )
}
