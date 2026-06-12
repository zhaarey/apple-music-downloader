import { LogFrontendError, GetLogPath, OpenLogFile } from '../wailsjs/go/main/App'

export function reportFrontendError(source, error, extra = '') {
  const message = error?.message || String(error ?? 'Unknown error')
  const stack = error?.stack || ''
  const detail = [stack, extra].filter(Boolean).join('\n\n')
  console.error(`[${source}]`, error, extra || '')
  try {
    LogFrontendError?.(source, message, detail)
  } catch {
    // Backend unavailable (e.g. browser-only dev)
  }
  return message
}

export function installGlobalErrorHandlers() {
  if (typeof window === 'undefined' || window.__auraErrorHandlersInstalled) return
  window.__auraErrorHandlersInstalled = true

  window.addEventListener('error', (event) => {
    const loc = event.filename ? `${event.filename}:${event.lineno}:${event.colno}` : ''
    reportFrontendError(
      'window.error',
      event.error || new Error(event.message || 'Script error'),
      loc,
    )
  })

  window.addEventListener('unhandledrejection', (event) => {
    reportFrontendError('unhandledrejection', event.reason)
  })
}

export async function openAppLogFile() {
  try {
    await OpenLogFile?.()
  } catch {
    const path = await GetLogPath?.()
    if (path) window.alert(`Log file: ${path}`)
  }
}
