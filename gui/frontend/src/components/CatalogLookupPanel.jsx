import { useEffect, useMemo, useState } from 'react'
import { Search as searchApi, ResolveSpotifyLink } from '../../wailsjs/go/main/App'
import {
  detectCatalogInput,
  extractAppleMusicUrls,
  inputKindHint,
  inputKindLabel,
  lookupButtonLabel,
  LOADING_MESSAGES,
  matchStatusMeta,
  matchMethodLabel,
  spotifySummary,
  defaultSelectedMatchIndices,
} from '../lib/catalogInput'
import { formatActionError } from '../lib/formatActionError'
import SpotifyMigrationGuide from './SpotifyMigrationGuide'
import {
  spotifyNeedsCredentials,
  formatUnmatchedForClipboard,
  indicesForStatus,
  collectUrlsFromIndices,
} from '../lib/spotifyMigration'

const SEARCH_TYPES = [
  { id: 'song', label: 'Songs' },
  { id: 'album', label: 'Albums' },
  { id: 'artist', label: 'Artists' },
]

function LookupStatusBanner({ message }) {
  return (
    <div className="mt-3 flex items-center gap-3 rounded-xl border border-accent/25 bg-accent/10 px-4 py-3 text-sm text-accent">
      <span className="inline-flex h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-accent/30 border-t-accent" />
      <span>{message}</span>
    </div>
  )
}

function StepLabel({ n, label }) {
  return (
    <p className="text-[11px] font-medium uppercase tracking-wide text-white/40">
      Step {n} · {label}
    </p>
  )
}

export default function CatalogLookupPanel({
  disabled = false,
  onAddAppleUrls,
  onSelectAppleUrl,
  showSearchTypes = true,
  settings = null,
  onOpenSettings,
}) {
  const [copiedUnmatched, setCopiedUnmatched] = useState(false)

  const bulkMode = Boolean(onAddAppleUrls)
  const [query, setQuery] = useState('')
  const [searchType, setSearchType] = useState('song')
  const [loading, setLoading] = useState(false)
  const [loadingMsg, setLoadingMsg] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [appleHits, setAppleHits] = useState([])
  const [searchOffset, setSearchOffset] = useState(0)
  const [hasNext, setHasNext] = useState(false)
  const [spotifyResult, setSpotifyResult] = useState(null)
  const [selectedMatchIdx, setSelectedMatchIdx] = useState(() => new Set())
  const [showTips, setShowTips] = useState(false)
  const [spotifyFilter, setSpotifyFilter] = useState('ready')

  const inputKind = useMemo(() => detectCatalogInput(query), [query])
  const hint = inputKindHint(inputKind, bulkMode)

  const phase = loading ? 'loading' : spotifyResult ? 'spotify' : appleHits.length ? 'search' : 'idle'

  useEffect(() => {
    if (!loading) {
      setLoadingMsg('')
      return undefined
    }
    const msgs = LOADING_MESSAGES[inputKind] || LOADING_MESSAGES.search
    let i = 0
    setLoadingMsg(msgs[0])
    if (msgs.length <= 1) return undefined
    const timer = setInterval(() => {
      i = (i + 1) % msgs.length
      setLoadingMsg(msgs[i])
    }, 2200)
    return () => clearInterval(timer)
  }, [loading, inputKind])

  const resetResults = () => {
    setAppleHits([])
    setSpotifyResult(null)
    setSelectedMatchIdx(new Set())
    setHasNext(false)
    setSearchOffset(0)
    setNotice('')
    setSpotifyFilter('ready')
  }

  const runLookup = async (offset = 0) => {
    const trimmed = query.trim()
    if (!trimmed) return
    setLoading(true)
    setError('')
    setNotice('')
    try {
      const kind = detectCatalogInput(trimmed)
      if (kind === 'apple') {
        resetResults()
        const urls = extractAppleMusicUrls(trimmed)
        if (urls.length === 0) {
          setError('That doesn’t look like a valid Apple Music link yet.')
          return
        }
        if (onAddAppleUrls) {
          onAddAppleUrls(urls)
          setQuery('')
          setNotice(
            urls.length === 1
              ? 'Added 1 link to your queue — loading preview next.'
              : `Added ${urls.length} links to your queue — loading previews next.`,
          )
          return
        }
        if (urls.length === 1 && onSelectAppleUrl) {
          onSelectAppleUrl(urls[0])
          setQuery('')
          return
        }
        setError('Paste one link here, or switch to bulk queue for multiple links.')
        return
      }

      if (kind === 'spotify') {
        setAppleHits([])
        if (spotifyNeedsCredentials(trimmed, settings)) {
          setError(null)
          setSpotifyResult(null)
          return
        }
        const res = await ResolveSpotifyLink(trimmed)
        if (res?.error) {
          setError(res.error)
          setSpotifyResult(null)
          return
        }
        setSpotifyResult(res)
        setSelectedMatchIdx(defaultSelectedMatchIndices(res.items))
        if (res.uncertain > 0) setSpotifyFilter('all')
        return
      }

      setSpotifyResult(null)
      const res = await searchApi(searchType, trimmed, offset)
      if (res.error) setError(res.error)
      setAppleHits(res.hits || [])
      setHasNext(!!res.hasNext)
      setSearchOffset(offset)
    } catch (e) {
      setError(formatActionError(e, 'Lookup'))
    } finally {
      setLoading(false)
    }
  }

  const toggleMatch = (idx) => {
    setSelectedMatchIdx((prev) => {
      const next = new Set(prev)
      if (next.has(idx)) next.delete(idx)
      else next.add(idx)
      return next
    })
  }

  const pushSpotifyUrls = (urls, meta = {}) => {
    if (!urls.length) {
      setError('No tracks selected — pick at least one match below.')
      return
    }
    if (onAddAppleUrls) {
      onAddAppleUrls(urls, { source: 'spotify', ...meta })
      setQuery('')
      resetResults()
      setNotice(
        `Step 3 · Added ${urls.length} song${urls.length !== 1 ? 's' : ''} to your queue — previews load below. Review, then Download queue.`,
      )
      return
    }
    if (urls.length === 1 && onSelectAppleUrl) {
      onSelectAppleUrl(urls[0])
      setQuery('')
      resetResults()
    }
  }

  const addSpotifyMatches = () => {
    if (!spotifyResult?.items?.length) return
    const urls = collectUrlsFromIndices(spotifyResult.items, [...selectedMatchIdx])
    pushSpotifyUrls(urls, {
      readyCount: spotifyFilterCounts.ready,
      missingCount: spotifyFilterCounts.missing,
      uncertainCount: spotifyFilterCounts.review,
    })
  }

  const addReadyMatches = () => {
    if (!spotifyResult?.items?.length) return
    const idxs = indicesForStatus(spotifyResult.items, ['matched'])
    setSelectedMatchIdx(new Set(idxs))
    pushSpotifyUrls(collectUrlsFromIndices(spotifyResult.items, idxs), {
      readyCount: idxs.length,
      missingCount: spotifyFilterCounts.missing,
    })
  }

  const addAllMatchedIncludingReview = () => {
    if (!spotifyResult?.items?.length) return
    const idxs = indicesForStatus(spotifyResult.items, ['matched', 'uncertain'])
    setSelectedMatchIdx(new Set(idxs))
    pushSpotifyUrls(collectUrlsFromIndices(spotifyResult.items, idxs), {
      readyCount: spotifyFilterCounts.ready,
      uncertainCount: spotifyFilterCounts.review,
    })
  }

  const copyUnmatched = async () => {
    if (!spotifyResult?.items?.length) return
    const text = formatUnmatchedForClipboard(spotifyResult.items)
    if (!text) return
    try {
      await navigator.clipboard.writeText(text)
      setCopiedUnmatched(true)
      setTimeout(() => setCopiedUnmatched(false), 2500)
    } catch {
      setError('Could not copy to clipboard.')
    }
  }

  const filteredSpotifyItems = useMemo(() => {
    if (!spotifyResult?.items) return []
    return spotifyResult.items
      .map((item, idx) => ({ item, idx }))
      .filter(({ item }) => {
        if (spotifyFilter === 'all') return true
        if (spotifyFilter === 'ready') return item.match_status === 'matched'
        if (spotifyFilter === 'review') return item.match_status === 'uncertain'
        if (spotifyFilter === 'missing') return item.match_status === 'not_found'
        return true
      })
  }, [spotifyResult, spotifyFilter])

  const spotifyFilterCounts = useMemo(() => {
    const items = spotifyResult?.items || []
    return {
      all: items.length,
      ready: items.filter((i) => i.match_status === 'matched').length,
      review: items.filter((i) => i.match_status === 'uncertain').length,
      missing: items.filter((i) => i.match_status === 'not_found').length,
    }
  }, [spotifyResult])

  return (
    <section className="space-y-3">
      {bulkMode && (
        <SpotifyMigrationGuide settings={settings} onOpenSettings={onOpenSettings} />
      )}

      <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
      <StepLabel n={bulkMode ? 2 : 1} label={bulkMode ? 'Paste Spotify or search' : 'Find music'} />
      <h3 className="mt-1 text-base font-semibold text-white/95">
        {bulkMode ? 'Find songs on Apple Music' : 'What would you like to download?'}
      </h3>
      <p className="mt-1 text-sm text-white/50">{hint}</p>

      {query.trim() && inputKind !== 'search' && !loading && (
        <div className="mt-3 inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/20 px-3 py-1 text-xs text-white/60">
          <span className="h-1.5 w-1.5 rounded-full bg-accent" />
          Detected {inputKindLabel(inputKind)}
        </div>
      )}

      {showSearchTypes && inputKind === 'search' && query.trim().length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {SEARCH_TYPES.map((t) => (
            <button
              key={t.id}
              type="button"
              disabled={disabled || loading}
              onClick={() => setSearchType(t.id)}
              className={`rounded-full px-3 py-1 text-xs font-medium transition ${
                searchType === t.id
                  ? 'bg-accent text-white'
                  : 'bg-black/20 text-white/55 hover:bg-black/30 hover:text-white/80'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      )}

      <div className="mt-3 flex gap-2">
        <input
          value={query}
          onChange={(e) => {
            setQuery(e.target.value)
            if (error) setError('')
            if (notice && phase === 'idle') setNotice('')
          }}
          onKeyDown={(e) => e.key === 'Enter' && runLookup(0)}
          disabled={disabled || loading}
          placeholder="Song name, Apple Music link, or Spotify link…"
          className="flex-1 rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm placeholder:text-white/30 disabled:opacity-50"
        />
        <button
          type="button"
          onClick={() => runLookup(0)}
          disabled={disabled || loading || !query.trim() || spotifyNeedsCredentials(query, settings)}
          className="shrink-0 rounded-xl bg-accent px-5 py-3 text-sm font-semibold disabled:opacity-40"
        >
          {lookupButtonLabel(inputKind, loading)}
        </button>
      </div>

      {!query.trim() && phase === 'idle' && (
        <button
          type="button"
          onClick={() => setShowTips((v) => !v)}
          className="mt-2 text-xs text-white/40 hover:text-white/65"
        >
          {showTips ? 'Hide tips' : 'Quick tips'}
        </button>
      )}

      {showTips && phase === 'idle' && (
        <ul className="mt-2 space-y-1 rounded-lg border border-white/10 bg-black/15 px-3 py-2.5 text-xs text-white/50">
          <li>Paste a Spotify playlist — we match tracks on Apple Music using ISRC when possible.</li>
          <li>Paste Apple Music album or song links to queue them directly.</li>
          <li>Spotify playlists need free API keys in Settings → Spotify matching.</li>
        </ul>
      )}

      {loading && loadingMsg && <LookupStatusBanner message={loadingMsg} />}

      {notice && !error && (
        <p className="mt-3 rounded-lg border border-emerald-500/25 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-100">
          {notice}
        </p>
      )}

      {error && (
        <div className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">
          <p>{error}</p>
          {(error.includes('Settings') || error.includes('API keys')) && onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              className="mt-2 text-xs font-medium text-accent hover:underline"
            >
              Open Settings → Spotify matching
            </button>
          )}
        </div>
      )}

      {spotifyNeedsCredentials(query, settings) && !loading && (
        <div className="mt-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2.5 text-sm text-amber-100">
          <p className="font-medium">Playlist migration needs a one-time Spotify setup</p>
          <p className="mt-1 text-xs text-amber-100/80">
            Add free API keys in Settings, make the playlist public on Spotify, then press Find on Apple Music.
          </p>
          {onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              className="mt-2 text-xs font-medium text-accent hover:underline"
            >
              Set up Spotify matching →
            </button>
          )}
        </div>
      )}

      {appleHits.length > 0 && (
        <div className="mt-4 space-y-2">
          <StepLabel n={2} label="Pick a result" />
          <p className="text-sm text-white/50">
            {appleHits.length} result{appleHits.length !== 1 ? 's' : ''} — choose one to continue.
          </p>
          <div className="max-h-56 overflow-y-auto">
            <div className="grid gap-2 sm:grid-cols-2">
              {appleHits.map((h) => (
                <div
                  key={h.id + h.url}
                  className="flex gap-3 rounded-lg border border-white/10 bg-black/20 p-2.5 transition hover:border-accent/40"
                >
                  {h.art_url ? (
                    <img src={h.art_url} alt="" className="h-14 w-14 shrink-0 rounded-lg object-cover" />
                  ) : (
                    <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-lg bg-surface text-xl">
                      ♫
                    </div>
                  )}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="truncate text-sm font-medium">{h.name}</p>
                      <span className="shrink-0 rounded bg-white/10 px-1.5 py-0.5 text-[10px] uppercase text-white/45">
                        {h.type}
                      </span>
                    </div>
                    <p className="truncate text-xs text-white/45">{h.detail}</p>
                    <button
                      type="button"
                      onClick={() => {
                        if (onAddAppleUrls) onAddAppleUrls([h.url])
                        else onSelectAppleUrl?.(h.url)
                      }}
                      className="mt-1.5 text-xs font-medium text-accent hover:underline"
                    >
                      {bulkMode ? 'Add to queue →' : 'Preview this →'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={searchOffset === 0 || loading}
              onClick={() => runLookup(Math.max(0, searchOffset - 15))}
              className="rounded-lg bg-black/20 px-3 py-1.5 text-xs disabled:opacity-30"
            >
              Previous
            </button>
            <button
              type="button"
              disabled={!hasNext || loading}
              onClick={() => runLookup(searchOffset + 15)}
              className="rounded-lg bg-black/20 px-3 py-1.5 text-xs disabled:opacity-30"
            >
              Next
            </button>
          </div>
        </div>
      )}

      {spotifyResult?.items?.length > 0 && (
        <div className="mt-4 space-y-3">
          <StepLabel n={bulkMode ? 3 : 2} label="Review matches" />
          <div className="rounded-xl border border-[#1DB954]/25 bg-[#1DB954]/[0.08] px-3 py-2.5">
            <p className="text-sm font-medium text-emerald-50">{spotifyResult.source_title || 'Spotify'}</p>
            <p className="mt-0.5 text-xs text-emerald-100/75">{spotifySummary(spotifyResult)}</p>
            <p className="mt-2 text-xs text-emerald-100/60">
              {bulkMode
                ? 'Ready matches are pre-selected. Add them to the queue, then scroll down to download.'
                : 'Confident matches are pre-selected. Uncheck anything you don’t want.'}
            </p>
          </div>

          <div className="flex flex-wrap gap-1">
            {[
              ['all', 'All'],
              ['ready', 'Ready'],
              ['review', 'Check'],
              ['missing', 'Missing'],
            ].map(([id, label]) => {
              const count = spotifyFilterCounts[id]
              if (id !== 'all' && count === 0) return null
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => setSpotifyFilter(id)}
                  className={`rounded-full px-2.5 py-1 text-[11px] font-medium ${
                    spotifyFilter === id
                      ? 'bg-white/15 text-white'
                      : 'bg-black/20 text-white/45 hover:text-white/70'
                  }`}
                >
                  {label} ({count})
                </button>
              )
            })}
          </div>

          <ul className="max-h-72 space-y-2 overflow-y-auto">
            {filteredSpotifyItems.length === 0 ? (
              <li className="py-6 text-center text-sm text-white/40">No tracks in this filter.</li>
            ) : (
              filteredSpotifyItems.map(({ item, idx }) => {
                const meta = matchStatusMeta(item.match_status)
                const hit = item.apple_hit
                const canSelect = hit?.url && item.match_status !== 'not_found'
                const method = matchMethodLabel(item.match_method)
                return (
                  <li
                    key={`${item.spotify_title}-${idx}`}
                    className="rounded-lg border border-white/10 bg-black/20 p-3"
                  >
                    <div className="flex gap-2">
                      {canSelect && bulkMode && (
                        <input
                          type="checkbox"
                          checked={selectedMatchIdx.has(idx)}
                          onChange={() => toggleMatch(idx)}
                          className="mt-1 shrink-0"
                          aria-label={`Include ${item.spotify_title}`}
                        />
                      )}
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="truncate text-sm font-medium">{item.spotify_title}</p>
                          <span
                            className={`rounded border px-1.5 py-0.5 text-[10px] font-medium ${meta.className}`}
                            title={meta.hint}
                          >
                            {meta.label}
                          </span>
                          {method && (
                            <span className="rounded bg-white/5 px-1.5 py-0.5 text-[10px] text-white/40">
                              {method}
                            </span>
                          )}
                        </div>
                        <p className="truncate text-xs text-white/40">{item.spotify_artist}</p>
                        {hit ? (
                          <p className="mt-1 truncate text-xs text-accent/90">
                            Apple Music → {hit.name}
                          </p>
                        ) : (
                          <p className="mt-1 text-xs text-red-300/75">{meta.hint}</p>
                        )}
                      </div>
                      {hit?.art_url && (
                        <img src={hit.art_url} alt="" className="h-11 w-11 shrink-0 rounded object-cover" />
                      )}
                    </div>
                    {!bulkMode && canSelect && (
                      <button
                        type="button"
                        onClick={() => onSelectAppleUrl?.(hit.url)}
                        className="mt-2 text-xs font-medium text-accent hover:underline"
                      >
                        Preview on Apple Music →
                      </button>
                    )}
                  </li>
                )
              })
            )}
          </ul>

          {bulkMode && (
            <div className="flex flex-col gap-2 sm:flex-row">
              {spotifyFilterCounts.ready > 0 && (
                <button
                  type="button"
                  onClick={addReadyMatches}
                  className="flex-1 rounded-xl bg-accent py-3 text-sm font-semibold hover:bg-accent-muted"
                >
                  Add {spotifyFilterCounts.ready} ready track{spotifyFilterCounts.ready !== 1 ? 's' : ''} to queue
                </button>
              )}
              {spotifyFilterCounts.review > 0 && (
                <button
                  type="button"
                  onClick={addAllMatchedIncludingReview}
                  className="flex-1 rounded-xl border border-white/15 bg-black/20 py-3 text-sm font-medium text-white/85 hover:bg-black/30"
                >
                  Include {spotifyFilterCounts.review} to review
                </button>
              )}
            </div>
          )}

          {bulkMode && (
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={addSpotifyMatches}
                disabled={selectedMatchIdx.size === 0}
                className="rounded-lg border border-white/15 px-3 py-1.5 text-xs text-white/70 hover:bg-white/5 disabled:opacity-40"
              >
                {selectedMatchIdx.size === 0
                  ? 'Add custom selection'
                  : `Add ${selectedMatchIdx.size} selected`}
              </button>
              {spotifyFilterCounts.missing > 0 && (
                <button
                  type="button"
                  onClick={copyUnmatched}
                  className="rounded-lg border border-white/15 px-3 py-1.5 text-xs text-white/70 hover:bg-white/5"
                >
                  {copiedUnmatched ? 'Copied!' : `Copy ${spotifyFilterCounts.missing} unmatched`}
                </button>
              )}
            </div>
          )}
        </div>
      )}
      </div>
    </section>
  )
}
