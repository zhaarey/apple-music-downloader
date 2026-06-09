import { useState } from 'react'
import { Search as searchApi } from '../../wailsjs/go/main/App'

const TYPES = ['album', 'song', 'artist']

export default function SearchTab({ onPreview, embedded = false }) {
  const [type, setType] = useState('album')
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState([])
  const [offset, setOffset] = useState(0)
  const [hasNext, setHasNext] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const runSearch = async (newOffset = 0) => {
    if (!query.trim()) return
    setLoading(true)
    setError('')
    try {
      const res = await searchApi(type, query.trim(), newOffset)
      if (res.error) setError(res.error)
      setHits(res.hits || [])
      setHasNext(!!res.hasNext)
      setOffset(newOffset)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section
      className={`rounded-xl border border-white/10 bg-surface-raised ${
        embedded ? 'p-4' : 'flex h-full flex-col gap-4 p-0 border-0 bg-transparent'
      }`}
    >
      {!embedded && (
        <div>
          <h2 className="text-xl font-semibold">Search Apple Music</h2>
          <p className="mt-1 text-sm text-white/50">Results open in Download to fetch tracks and start a job</p>
        </div>
      )}
      {embedded && (
        <div className="mb-3">
          <h3 className="font-medium">Search catalog</h3>
          <p className="mt-1 text-xs text-white/50">Find albums, songs, or artists — then fetch below to download</p>
        </div>
      )}
      <div className={`flex flex-wrap gap-2 ${embedded ? '' : ''}`}>
        {TYPES.map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setType(t)}
            className={`rounded-lg px-3 py-1.5 text-sm capitalize ${
              type === t ? 'bg-accent text-white' : 'bg-black/20 text-white/70 hover:bg-black/30'
            }`}
          >
            {t}s
          </button>
        ))}
      </div>
      <div className="mt-3 flex gap-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && runSearch(0)}
          placeholder="Search Apple Music…"
          className="flex-1 rounded-xl border border-white/10 bg-black/20 px-4 py-2.5 text-sm"
        />
        <button
          type="button"
          onClick={() => runSearch(0)}
          disabled={loading}
          className="rounded-xl bg-accent px-5 py-2.5 text-sm font-medium disabled:opacity-50"
        >
          Search
        </button>
      </div>
      {error && <p className="mt-2 text-sm text-red-400">{error}</p>}
      {hits.length > 0 && (
        <div className={`mt-3 ${embedded ? 'max-h-52 overflow-y-auto' : 'flex-1 overflow-y-auto'}`}>
          <div className="grid gap-2 sm:grid-cols-2">
            {hits.map((h) => (
              <div
                key={h.id + h.url}
                className="flex gap-3 rounded-lg border border-white/10 bg-black/20 p-2.5 transition hover:border-accent/40"
              >
                {h.art_url ? (
                  <img src={h.art_url} alt="" className="h-14 w-14 shrink-0 rounded-lg object-cover" />
                ) : (
                  <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-lg bg-surface text-xl">♫</div>
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{h.name}</p>
                  <p className="truncate text-xs text-white/50">{h.detail}</p>
                  <button
                    type="button"
                    onClick={() => onPreview(h.url)}
                    className="mt-1 text-xs font-medium text-accent hover:underline"
                  >
                    Use in download →
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
      {hits.length > 0 && (
        <div className="mt-2 flex justify-end gap-2">
          <button
            type="button"
            disabled={offset === 0 || loading}
            onClick={() => runSearch(Math.max(0, offset - 15))}
            className="rounded-lg bg-black/20 px-3 py-1.5 text-xs disabled:opacity-30"
          >
            Previous
          </button>
          <button
            type="button"
            disabled={!hasNext || loading}
            onClick={() => runSearch(offset + 15)}
            className="rounded-lg bg-black/20 px-3 py-1.5 text-xs disabled:opacity-30"
          >
            Next
          </button>
        </div>
      )}
    </section>
  )
}
