import { useState } from 'react'
import { Search as searchApi } from '../wailsjs/go/main/App'

const TYPES = ['album', 'song', 'artist']

export default function SearchTab({ onPreview }) {
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
    <div className="flex h-full flex-col gap-4">
      <div>
        <h2 className="text-xl font-semibold">Search Apple Music</h2>
        <p className="mt-1 text-sm text-white/50">Results open in Download to fetch tracks and start a job</p>
      </div>
      <div className="flex flex-wrap gap-2">
        {TYPES.map((t) => (
          <button
            key={t}
            onClick={() => setType(t)}
            className={`rounded-lg px-4 py-2 text-sm capitalize ${
              type === t ? 'bg-accent' : 'bg-surface-raised text-white/70'
            }`}
          >
            {t}s
          </button>
        ))}
      </div>
      <div className="flex gap-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && runSearch(0)}
          placeholder="Search Apple Music…"
          className="flex-1 rounded-xl border border-white/10 bg-surface-raised px-4 py-3"
        />
        <button onClick={() => runSearch(0)} disabled={loading} className="rounded-xl bg-accent px-6 py-3 font-medium">
          Search
        </button>
      </div>
      {error && <p className="text-sm text-red-400">{error}</p>}
      <div className="flex-1 overflow-y-auto">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {hits.map((h) => (
            <div
              key={h.id + h.url}
              className="flex gap-3 rounded-xl border border-white/10 bg-surface-raised p-3 transition hover:border-accent/50"
            >
              {h.art_url ? (
                <img src={h.art_url} alt="" className="h-16 w-16 rounded-lg object-cover" />
              ) : (
                <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-surface text-2xl">♫</div>
              )}
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{h.name}</p>
                <p className="truncate text-xs text-white/50">{h.detail}</p>
                <button
                  onClick={() => onPreview(h.url)}
                  className="mt-2 text-xs font-medium text-accent hover:underline"
                >
                  Preview & download →
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="flex justify-center gap-2">
        <button
          disabled={offset === 0 || loading}
          onClick={() => runSearch(Math.max(0, offset - 15))}
          className="rounded-lg bg-surface-raised px-4 py-2 text-sm disabled:opacity-30"
        >
          Previous
        </button>
        <button
          disabled={!hasNext || loading}
          onClick={() => runSearch(offset + 15)}
          className="rounded-lg bg-surface-raised px-4 py-2 text-sm disabled:opacity-30"
        >
          Next
        </button>
      </div>
    </div>
  )
}
