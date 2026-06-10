import { useEffect, useState } from 'react'

const VARIANTS = {
  success: 'border-green-400/25 bg-green-500/15 text-green-100',
  info: 'border-white/20 bg-white/10 text-white/85',
  error: 'border-red-400/30 bg-red-500/12 text-red-200',
}

export default function StatusToast({ message, variant = 'success', onDismiss, duration = 2800 }) {
  const [copied, setCopied] = useState(false)
  const showCopy = variant === 'error' && Boolean(message)
  const dismissMs = variant === 'error' ? Math.max(duration, 12000) : duration

  useEffect(() => {
    setCopied(false)
  }, [message])

  useEffect(() => {
    if (!message) return undefined
    const t = setTimeout(() => onDismiss?.(), dismissMs)
    return () => clearTimeout(t)
  }, [message, dismissMs, onDismiss])

  const copyMessage = async () => {
    if (!message) return
    try {
      await navigator.clipboard.writeText(message)
      setCopied(true)
      setTimeout(() => setCopied(false), 1600)
    } catch {
      // Fallback for environments without clipboard API
      const el = document.createElement('textarea')
      el.value = message
      el.setAttribute('readonly', '')
      el.style.position = 'absolute'
      el.style.left = '-9999px'
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
      setCopied(true)
      setTimeout(() => setCopied(false), 1600)
    }
  }

  if (!message) return null

  return (
    <div className="pointer-events-none absolute inset-x-0 top-3 z-20 flex justify-center px-4">
      <div
        role="status"
        aria-live="polite"
        className={`pointer-events-auto flex max-w-3xl items-center gap-2 rounded-full border px-4 py-2 text-sm shadow-lg backdrop-blur-md ${VARIANTS[variant] || VARIANTS.success}`}
      >
        <span className="min-w-0 flex-1 break-words text-left">{message}</span>
        {showCopy && (
          <button
            type="button"
            onClick={copyMessage}
            className="shrink-0 rounded-md border border-current/25 px-2 py-0.5 text-xs font-medium opacity-90 transition hover:opacity-100"
            title="Copy error message"
          >
            {copied ? 'Copied' : 'Copy'}
          </button>
        )}
        {variant === 'error' && (
          <button
            type="button"
            onClick={onDismiss}
            className="shrink-0 rounded-md px-1.5 text-xs opacity-70 transition hover:opacity-100"
            aria-label="Dismiss"
          >
            ✕
          </button>
        )}
      </div>
    </div>
  )
}
