import { useCallback, useEffect, useRef, useState } from 'react'

const BLOB_FALLBACK_MAX_BYTES = 400 * 1024 * 1024

/** Turn a splice-media path into an absolute URL for the current webview origin. */
export function resolveMediaURL(relativePath) {
  const path = relativePath == null ? '' : String(relativePath)
  if (!path || path === '[object Promise]') return ''
  if (/^(https?:|wails:)/i.test(path)) return path
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  if (!origin || origin === 'null') return path
  return `${origin}${path.startsWith('/') ? path : `/${path}`}`
}

async function probeMediaURL(url) {
  const res = await fetch(url, { method: 'GET', headers: { Range: 'bytes=0-1' } })
  if (!res.ok && res.status !== 206) {
    throw new Error(`Audio preview unavailable (HTTP ${res.status})`)
  }
  const type = (res.headers.get('content-type') || '').toLowerCase()
  if (type.includes('text/html')) {
    throw new Error('Audio route misconfigured — restart the app or run via wails dev / built exe')
  }
  const lengthHeader = res.headers.get('content-length')
  const totalHeader = res.headers.get('content-range')?.split('/')[1]
  const size = Number(totalHeader || lengthHeader || 0)
  return { contentType: type, size }
}

async function loadBlobURL(url, maxBytes) {
  const head = await fetch(url, { method: 'HEAD' }).catch(() => null)
  const size = Number(head?.headers.get('content-length') || 0)
  if (size > maxBytes) {
    throw new Error('Master file is too large to buffer — streaming preview failed')
  }
  const res = await fetch(url)
  if (!res.ok) throw new Error(`Audio preview unavailable (HTTP ${res.status})`)
  const blob = await res.blob()
  if ((blob.type || '').includes('text/html')) {
    throw new Error('Audio route returned HTML instead of audio')
  }
  return URL.createObjectURL(blob)
}

/**
 * Loads master audio for HTML5 preview (streaming URL first, blob fallback if needed).
 */
export function useMasterAudio(relativeURL) {
  const audioRef = useRef(null)
  const blobUrlRef = useRef('')
  const [status, setStatus] = useState('idle') // idle | loading | ready | error
  const [error, setError] = useState('')

  const revokeBlob = useCallback(() => {
    if (blobUrlRef.current) {
      URL.revokeObjectURL(blobUrlRef.current)
      blobUrlRef.current = ''
    }
  }, [])

  useEffect(() => {
    const audio = audioRef.current
    const urlInput = relativeURL == null ? '' : String(relativeURL)
    if (!urlInput || urlInput === '[object Promise]') {
      revokeBlob()
      if (audio) {
        audio.pause()
        audio.removeAttribute('src')
        audio.load()
      }
      setStatus('idle')
      setError('')
      return
    }

    let cancelled = false
    const absoluteURL = resolveMediaURL(urlInput)

    async function load() {
      setStatus('loading')
      setError('')
      revokeBlob()
      if (audio) {
        audio.pause()
        audio.removeAttribute('src')
      }

      try {
        await probeMediaURL(absoluteURL)
        if (cancelled || !audio) return

        await new Promise((resolve, reject) => {
          const onReady = () => {
            cleanup()
            resolve()
          }
          const onError = () => {
            cleanup()
            reject(new Error('Could not decode master audio'))
          }
          const cleanup = () => {
            audio.removeEventListener('loadedmetadata', onReady)
            audio.removeEventListener('error', onError)
          }
          audio.addEventListener('loadedmetadata', onReady)
          audio.addEventListener('error', onError)
          audio.src = absoluteURL
          audio.load()
        })

        if (!cancelled) {
          setStatus('ready')
          setError('')
        }
      } catch (streamErr) {
        if (cancelled || !audio) return
        try {
          const blobURL = await loadBlobURL(absoluteURL, BLOB_FALLBACK_MAX_BYTES)
          if (cancelled) {
            URL.revokeObjectURL(blobURL)
            return
          }
          blobUrlRef.current = blobURL
          await new Promise((resolve, reject) => {
            const onReady = () => {
              cleanup()
              resolve()
            }
            const onError = () => {
              cleanup()
              reject(new Error('Could not decode master audio'))
            }
            const cleanup = () => {
              audio.removeEventListener('loadedmetadata', onReady)
              audio.removeEventListener('error', onError)
            }
            audio.addEventListener('loadedmetadata', onReady)
            audio.addEventListener('error', onError)
            audio.src = blobURL
            audio.load()
          })
          if (!cancelled) {
            setStatus('ready')
            setError('')
          }
        } catch (fallbackErr) {
          if (!cancelled) {
            setStatus('error')
            setError(String(fallbackErr?.message || streamErr?.message || fallbackErr))
          }
        }
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [relativeURL, revokeBlob])

  useEffect(() => () => revokeBlob(), [revokeBlob])

  return { audioRef, status, error, isReady: status === 'ready' }
}
