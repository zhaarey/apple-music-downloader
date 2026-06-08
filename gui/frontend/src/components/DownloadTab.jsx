import { useEffect, useState } from 'react'
import { DetectURLType } from '../wailsjs/go/main/App'

const QUALITIES = [
  { id: 'aac', label: 'AAC', desc: 'Works immediately — no wrapper needed', needsWrapper: false },
  { id: 'alac', label: 'Lossless (ALAC)', desc: 'Requires wrapper decrypt service', needsWrapper: true },
  { id: 'atmos', label: 'Dolby Atmos', desc: 'Requires wrapper decrypt service', needsWrapper: true },
]

export default function DownloadTab({ settings, deps, prefillUrl, onPrefillConsumed, onDownload, downloading }) {
  const [url, setUrl] = useState('')
  const [quality, setQuality] = useState('aac')
  const [singleSong, setSingleSong] = useState(false)
  const [allArtistAlbums, setAllArtistAlbums] = useState(false)
  const [urlType, setUrlType] = useState('')

  const wrapperOk = deps?.some((d) => d.name?.includes('wrapper') && d.ok)

  useEffect(() => {
    if (prefillUrl) {
      setUrl(prefillUrl)
      onPrefillConsumed()
    }
  }, [prefillUrl])

  useEffect(() => {
    if (url.length > 20) {
      DetectURLType(url).then(setUrlType)
    } else {
      setUrlType('')
    }
  }, [url])

  const submit = () => {
    if (!url.trim() || downloading) return
    onDownload({
      urls: [url.trim()],
      quality,
      singleSong,
      selectTracks: false,
      allArtistAlbums,
    })
  }

  return (
    <div className="mx-auto flex h-full max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-xl font-semibold">Download from URL</h2>
        <p className="mt-1 text-sm text-white/50">Paste an Apple Music album, song, playlist, or artist link</p>
      </div>

      <div className="relative">
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://music.apple.com/us/album/..."
          className="w-full rounded-xl border border-white/10 bg-surface-raised px-4 py-4 pr-24 text-sm focus:border-accent focus:outline-none"
        />
        {urlType && (
          <span className="absolute right-3 top-1/2 -translate-y-1/2 rounded-full bg-accent/20 px-3 py-1 text-xs text-accent">
            {urlType}
          </span>
        )}
      </div>

      <div>
        <p className="mb-2 text-sm text-white/60">Quality</p>
        <div className="grid gap-2 sm:grid-cols-3">
          {QUALITIES.map((q) => {
            const blocked = q.needsWrapper && !wrapperOk
            return (
              <button
                key={q.id}
                type="button"
                disabled={blocked}
                onClick={() => setQuality(q.id)}
                className={`rounded-xl border p-3 text-left transition ${
                  quality === q.id
                    ? 'border-accent bg-accent/10'
                    : blocked
                      ? 'border-white/5 opacity-40'
                      : 'border-white/10 hover:border-white/20'
                }`}
              >
                <div className="font-medium">{q.label}</div>
                <div className="mt-1 text-xs text-white/50">{q.desc}</div>
              </button>
            )
          })}
        </div>
        {!wrapperOk && (quality === 'alac' || quality === 'atmos') && (
          <p className="mt-2 text-sm text-yellow-400">
            Wrapper not detected. ALAC/Atmos need manual wrapper setup — see README-WINDOWS.md or Settings.
          </p>
        )}
      </div>

      <div className="flex flex-wrap gap-4 text-sm">
        <label className="flex items-center gap-2">
          <input type="checkbox" checked={singleSong} onChange={(e) => setSingleSong(e.target.checked)} />
          Single song only
        </label>
        <label className="flex items-center gap-2 opacity-50" title="Use CLI --select for interactive track picking">
          <input type="checkbox" disabled checked={false} />
          Select tracks (CLI only)
        </label>
        <label className="flex items-center gap-2">
          <input type="checkbox" checked={allArtistAlbums} onChange={(e) => setAllArtistAlbums(e.target.checked)} />
          All artist albums
        </label>
      </div>

      <button
        onClick={submit}
        disabled={!url.trim() || downloading}
        className="rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
      >
        {downloading ? 'Downloading…' : 'Start download'}
      </button>
    </div>
  )
}
