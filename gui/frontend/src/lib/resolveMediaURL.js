/** Turn a local-media path into an absolute URL for the current webview origin. */
export function resolveMediaURL(relativePath) {
  const path = relativePath == null ? '' : String(relativePath)
  if (!path || path === '[object Promise]') return ''
  if (/^(https?:|wails:)/i.test(path)) return path
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  if (!origin || origin === 'null') return path
  return `${origin}${path.startsWith('/') ? path : `/${path}`}`
}
