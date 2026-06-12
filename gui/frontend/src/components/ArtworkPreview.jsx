import { useEffect, useState } from 'react'

/** Keeps the previous image visible until the next src has finished loading. */
function usePreloadedSrc(src) {
  const [displaySrc, setDisplaySrc] = useState(src || '')

  useEffect(() => {
    if (!src) {
      setDisplaySrc('')
      return
    }
    if (src === displaySrc) return

    let cancelled = false
    const img = new Image()
    img.onload = () => {
      if (!cancelled) setDisplaySrc(src)
    }
    img.onerror = () => {
      if (!cancelled) setDisplaySrc(src)
    }
    img.src = src
    return () => {
      cancelled = true
    }
  }, [src, displaySrc])

  return displaySrc
}

export default function ArtworkPreview({ src, className = '' }) {
  const displaySrc = usePreloadedSrc(src)
  const [visible, setVisible] = useState(!!displaySrc)

  useEffect(() => {
    if (displaySrc) {
      setVisible(true)
      return undefined
    }
    const t = setTimeout(() => setVisible(false), 220)
    return () => clearTimeout(t)
  }, [displaySrc])

  return (
    <div className={`relative overflow-hidden ${className}`}>
      <div
        className={`absolute inset-0 flex items-center justify-center transition-opacity duration-300 ease-out ${
          displaySrc ? 'opacity-0' : visible ? 'opacity-100' : 'opacity-0'
        }`}
        aria-hidden={!!displaySrc}
      >
        <span className="text-4xl text-white/20">♫</span>
      </div>
      {displaySrc && (
        <img
          src={displaySrc}
          alt=""
          className="h-full w-full object-cover transition-opacity duration-300 ease-out"
          draggable={false}
        />
      )}
    </div>
  )
}
