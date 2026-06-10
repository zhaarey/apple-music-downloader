import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  PreviewURL,
  StartBulkDownloadJob,
  PreflightDownloadJob,
  CheckTrackDuplicates,
  RevealInFolder,
} from '../wailsjs/go/main/App'
import {
  parseBulkUrls,
  newQueueItem,
  buildBulkDownloadPlan,
  buildBulkDownloadEntries,
  queueItemDownloadStatus,
  expandQueueItemUrls,
  isAppleMusicURL,
  filterQueueItems,
  countQueueFilters,
  filterPreviewTracks,
  collectDuplicateTracks,
  countAlbumsWithDuplicates,
  setAllDupesPolicy,
  setTrackPolicy,
  isFullySkippedAlbum,
  getTrackChoice,
  summarizeItemTracks,
  applyTrackSelectionPolicy,
  isTrackOnDisk,
} from '../lib/parseBulkUrls'
import BulkAlbumCompareModal from './BulkAlbumCompareModal'
import DownloadFlowLayout from './download/DownloadFlowLayout'
import { parseBulkQueueProgress } from '../lib/bulkQueueProgress'
import { parseJobResult, jobStatusMeta } from '../lib/downloadStatus'
import { formatActionError } from '../lib/formatActionError'
import { reportFrontendError } from '../lib/errorReporting'

const CONCURRENCY = 4

const QUEUE_FILTERS = [
  { id: 'all', label: 'All' },
  { id: 'ready', label: 'Ready' },
  { id: 'duplicates', label: 'Duplicates' },
  { id: 'issues', label: 'Issues' },
  { id: 'on_disk', label: 'Albums w/ dupes' },
  { id: 'loading', label: 'Loading' },
  { id: 'downloading', label: 'Active' },
  { id: 'done', label: 'Done' },
]

const TRACK_FILTERS = [
  { id: 'all', label: 'All tracks' },
  { id: 'missing', label: 'Missing' },
  { id: 'on_disk', label: 'On disk' },
]

async function mapPool(items, limit, fn) {
  const results = new Array(items.length)
  let next = 0
  async function worker() {
    while (next < items.length) {
      const i = next++
      results[i] = await fn(items[i], i)
    }
  }
  const workers = Array.from({ length: Math.min(limit, items.length) }, () => worker())
  await Promise.all(workers)
  return results
}

function statusBadge(status, downloadPhase) {
  if (downloadPhase === 'downloading') {
    return { text: 'Downloading…', className: 'border-accent/40 bg-accent/10 text-accent' }
  }
  if (downloadPhase === 'done') {
    return { text: 'Done', className: 'border-green-500/40 bg-green-500/10 text-green-300' }
  }
  switch (status) {
    case 'loading':
      return { text: 'Loading…', className: 'border-white/20 bg-white/5 text-white/60' }
    case 'ready':
      return { text: 'Ready', className: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300' }
    case 'error':
      return { text: 'Error', className: 'border-red-500/40 bg-red-500/10 text-red-300' }
    default:
      return { text: 'Pending', className: 'border-white/15 bg-white/[0.03] text-white/45' }
  }
}

function FilterTabs({ tabs, active, counts, onChange, disabled }) {
  return (
    <div className="flex flex-wrap gap-1 border-b border-white/10 px-4 py-2">
      {tabs.map((tab) => {
        const count = counts[tab.id] ?? 0
        if (tab.id !== 'all' && count === 0) return null
        const isActive = active === tab.id
        const isIssues = tab.id === 'issues' && count > 0
        const isDupes = tab.id === 'duplicates' && count > 0
        return (
          <button
            key={tab.id}
            type="button"
            disabled={disabled}
            onClick={() => onChange(tab.id)}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium transition disabled:opacity-40 ${
              isActive
                ? isIssues
                  ? 'bg-red-500/20 text-red-200 ring-1 ring-red-500/40'
                  : isDupes
                    ? 'bg-emerald-500/20 text-emerald-200 ring-1 ring-emerald-500/40'
                    : 'bg-accent/20 text-accent ring-1 ring-accent/40'
                : isIssues
                  ? 'text-red-300/90 hover:bg-red-500/10'
                  : isDupes
                    ? 'text-emerald-300/90 hover:bg-emerald-500/10'
                    : 'text-white/55 hover:bg-white/5 hover:text-white/80'
            }`}
          >
            {tab.label}
            <span className={`ml-1.5 tabular-nums ${isActive ? 'opacity-90' : 'opacity-50'}`}>{count}</span>
          </button>
        )
      })}
    </div>
  )
}

function DuplicateTracksPanel({
  tracks,
  albumCount,
  onJumpToItem,
  onCompareAlbum,
  onRemoveFullyOnDisk,
  fullyOnDiskCount,
  downloading,
}) {
  if (tracks.length === 0) {
    return (
      <p className="px-4 py-8 text-center text-sm text-white/45">
        No duplicate tracks detected yet. Duplicates appear after previews finish loading and your output / extra
        check folders are scanned.
      </p>
    )
  }

  const byAlbum = useMemo(() => {
    const groups = new Map()
    for (const t of tracks) {
      const key = t.itemId
      if (!groups.has(key)) {
        groups.set(key, { album: t.album, artist: t.artist, art: t.albumArt, itemId: t.itemId, tracks: [] })
      }
      groups.get(key).tracks.push(t)
    }
    return [...groups.values()]
  }, [tracks])

  return (
    <div className="max-h-[min(420px,50vh)] overflow-y-auto">
      <div className="space-y-2 border-b border-white/10 bg-emerald-500/[0.06] px-4 py-3">
        <p className="text-xs text-emerald-100/90">
          <strong className="font-medium text-emerald-200">{tracks.length}</strong> track{tracks.length !== 1 ? 's' : ''}{' '}
          already on disk across{' '}
          <strong className="font-medium text-emerald-200">{albumCount}</strong> album{albumCount !== 1 ? 's' : ''}.
        </p>
        <p className="text-xs text-emerald-100/70">
          By default these are <strong className="font-medium">skipped</strong> when the download queue runs. Use{' '}
          <strong className="font-medium">Compare</strong> on an album to re-download specific tracks instead.
        </p>
        {!downloading && (
          <div className="flex flex-wrap gap-2 pt-1">
            {fullyOnDiskCount > 0 && (
              <button
                type="button"
                onClick={onRemoveFullyOnDisk}
                className="rounded-lg border border-white/15 bg-black/20 px-3 py-1.5 text-xs text-white/80 hover:bg-white/5"
              >
                Remove {fullyOnDiskCount} fully on-disk album{fullyOnDiskCount !== 1 ? 's' : ''} from queue
              </button>
            )}
          </div>
        )}
      </div>
      <ul className="divide-y divide-white/5">
        {byAlbum.map((group) => (
          <li key={group.itemId} className="px-4 py-3">
            <div className="mb-2 flex items-center gap-2">
              {group.art ? (
                <img src={group.art} alt="" className="h-8 w-8 shrink-0 rounded object-cover" />
              ) : (
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded bg-surface text-xs text-white/30">
                  ♪
                </div>
              )}
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{group.album}</p>
                <p className="truncate text-xs text-white/45">{group.artist}</p>
              </div>
              <button
                type="button"
                onClick={() => onCompareAlbum(group.itemId)}
                className="shrink-0 text-xs font-medium text-accent hover:underline"
              >
                Compare
              </button>
              <button
                type="button"
                onClick={() => onJumpToItem(group.itemId)}
                className="shrink-0 text-xs text-white/45 hover:text-white/70"
              >
                In queue
              </button>
            </div>
            <ul className="space-y-1 rounded-lg bg-black/20 px-3 py-2">
              {group.tracks.map((t) => (
                <li key={t.id} className="flex items-start justify-between gap-3 text-xs">
                  <div className="min-w-0">
                    <p className="truncate text-white/85">
                      <span className="text-white/40">{t.num}.</span> {t.name}
                    </p>
                    {t.artist && t.artist !== group.artist && (
                      <p className="truncate text-white/40">{t.artist}</p>
                    )}
                    {t.root && <p className="truncate text-[10px] text-white/35">Found in: {t.root}</p>}
                  </div>
                  {t.path && (
                    <button
                      type="button"
                      onClick={() => RevealInFolder(t.path)}
                      className="shrink-0 rounded-md border border-emerald-500/30 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-emerald-300 hover:bg-emerald-500/10"
                    >
                      Show file
                    </button>
                  )}
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </div>
  )
}

function BulkQueueItemRow({
  item,
  downloading,
  dlPhase,
  isExpanded,
  trackFilter,
  editingId,
  editUrlDraft,
  onToggleExpand,
  onTrackFilterChange,
  onStartEditUrl,
  onEditDraftChange,
  onSaveEditUrl,
  onCancelEdit,
  onRetry,
  onRemove,
  onCompare,
  showCompare,
  onTrackPolicyChange,
  onSelectAllTracks,
}) {
  const badge = statusBadge(item.status, downloading ? dlPhase : null)
  const dupCount = item.duplicateInfo?.existing_count ?? 0
  const selectedCount = item.duplicateInfo?.selected_count ?? 0
  const urlCount = expandQueueItemUrls(item).length
  const isEditing = editingId === item.id
  const tracks = item.preview?.tracks || []
  const filteredTracks = filterPreviewTracks(tracks, item.duplicateInfo, trackFilter)
  const missingCount = selectedCount > 0 ? selectedCount - dupCount : 0
  const trackSummary = summarizeItemTracks(item)
  const fullySkipped = isFullySkippedAlbum(item)

  return (
    <li
      className={`px-4 py-3 ${item.status === 'error' ? 'bg-red-500/[0.04]' : ''} ${
        dlPhase === 'downloading' ? 'bg-accent/[0.04]' : ''
      }`}
    >
      <div className="flex gap-3">
        {item.preview?.art_url ? (
          <img src={item.preview.art_url} alt="" className="h-14 w-14 shrink-0 rounded-lg object-cover" />
        ) : (
          <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-lg bg-surface text-lg text-white/30">
            ♪
          </div>
        )}
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              {!isEditing ? (
                <>
                  <p className="truncate font-medium">{item.preview?.title || 'Apple Music link'}</p>
                  <p className="truncate text-xs text-white/45">
                    {item.preview?.subtitle || item.preview?.type || 'Waiting for preview…'}
                  </p>
                </>
              ) : (
                <div className="space-y-2">
                  <label className="text-[10px] font-medium uppercase tracking-wide text-white/45">Edit URL</label>
                  <input
                    type="url"
                    value={editUrlDraft}
                    onChange={(e) => onEditDraftChange(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') onSaveEditUrl(item.id)
                      if (e.key === 'Escape') onCancelEdit()
                    }}
                    placeholder="https://music.apple.com/..."
                    className="w-full rounded-lg border border-accent/40 bg-surface px-3 py-2 text-sm focus:border-accent focus:outline-none"
                    autoFocus
                  />
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => onSaveEditUrl(item.id)}
                      className="rounded-lg bg-accent px-3 py-1 text-xs font-medium hover:bg-accent-muted"
                    >
                      Save &amp; refetch
                    </button>
                    <button
                      type="button"
                      onClick={onCancelEdit}
                      className="rounded-lg border border-white/15 px-3 py-1 text-xs text-white/60 hover:bg-white/5"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}
            </div>
            <span
              className={`shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium uppercase ${badge.className}`}
            >
              {badge.text}
            </span>
          </div>

          {!isEditing && (
            <p className="mt-1 truncate font-mono text-[11px] text-white/35" title={item.url}>
              {item.url}
            </p>
          )}

          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-white/40">
            {item.preview?.type && <span>{item.preview.type}</span>}
            {item.preview?.track_count > 0 && (
              <span>
                {item.preview.track_count} item{item.preview.track_count !== 1 ? 's' : ''}
              </span>
            )}
            {urlCount > 1 && <span>{urlCount} albums in discography</span>}
            {dupCount > 0 && selectedCount > 0 && (
              <span className="text-emerald-400/90">
                {trackSummary.skip} skip · {trackSummary.download} download
                {trackSummary.redownload > 0 ? ` · ${trackSummary.redownload} re-dl` : ''}
              </span>
            )}
            {fullySkipped && (
              <span className="text-yellow-400/90">excluded from download</span>
            )}
          </div>

          {fullySkipped && (
            <p className="mt-2 rounded-lg border border-yellow-500/25 bg-yellow-500/10 px-2.5 py-1.5 text-xs text-yellow-100">
              All tracks set to skip — this album will not download.
            </p>
          )}

          {item.error && (
            <p className="mt-2 rounded-lg border border-red-500/30 bg-red-500/10 px-2.5 py-1.5 text-xs text-red-200">
              {item.error}
            </p>
          )}
        </div>

        {!downloading && !isEditing && (
          <div className="flex shrink-0 flex-col gap-1 text-right">
            {showCompare && dupCount > 0 && (
              <button type="button" onClick={() => onCompare(item)} className="text-xs font-medium text-accent hover:underline">
                Compare
              </button>
            )}
            <button type="button" onClick={() => onStartEditUrl(item)} className="text-xs text-accent hover:underline">
              Edit URL
            </button>
            {item.status === 'error' && (
              <button type="button" onClick={() => onRetry(item)} className="text-xs text-accent hover:underline">
                Retry
              </button>
            )}
            <button type="button" onClick={() => onRemove(item.id)} className="text-xs text-white/40 hover:text-red-300">
              Remove
            </button>
          </div>
        )}
      </div>

      {tracks.length > 0 && (
        <div className="mt-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={() => onToggleExpand(item.id)}
                className="text-xs text-white/45 hover:text-white/70"
              >
                {isExpanded ? 'Hide tracks' : `Show ${tracks.length} tracks`}
              </button>
              {isExpanded && !downloading && tracks.length > 1 && (
                <>
                  <button
                    type="button"
                    onClick={() => onSelectAllTracks(item.id, true)}
                    className="text-[10px] text-accent hover:underline"
                  >
                    Select all
                  </button>
                  <button
                    type="button"
                    onClick={() => onSelectAllTracks(item.id, false)}
                    className="text-[10px] text-white/45 hover:text-white/70"
                  >
                    Select none
                  </button>
                </>
              )}
            </div>
            {isExpanded && dupCount > 0 && (
              <div className="flex gap-1">
                {TRACK_FILTERS.map((f) => {
                  const n =
                    f.id === 'all'
                      ? tracks.length
                      : f.id === 'on_disk'
                        ? dupCount
                        : missingCount
                  if (f.id !== 'all' && n === 0) return null
                  return (
                    <button
                      key={f.id}
                      type="button"
                      onClick={() => onTrackFilterChange(f.id)}
                      className={`rounded-md px-2 py-0.5 text-[10px] ${
                        trackFilter === f.id
                          ? 'bg-white/10 text-white/90'
                          : 'text-white/45 hover:text-white/70'
                      }`}
                    >
                      {f.label} ({n})
                    </button>
                  )
                })}
              </div>
            )}
          </div>
          {isExpanded && (
            <ul className="mt-2 max-h-36 space-y-1 overflow-y-auto rounded-lg bg-black/20 px-3 py-2 text-xs">
              {filteredTracks.length === 0 ? (
                <li className="py-2 text-center text-white/40">No tracks match this filter.</li>
              ) : (
                filteredTracks.map((t) => {
                  const dup = item.duplicateInfo?.tracks?.find((d) => d.num === t.num)
                  const onDisk = dup?.on_disk
                  const choice = getTrackChoice(item, t.num)
                  const included = choice !== 'skip'
                  return (
                    <li key={t.num} className="flex items-center justify-between gap-2 text-white/60">
                      <label className="flex min-w-0 flex-1 cursor-pointer items-center gap-2">
                        {!downloading && (
                          <input
                            type="checkbox"
                            checked={included}
                            onChange={() =>
                              onTrackPolicyChange(item.id, t.num, included ? 'skip' : 'download')
                            }
                            className="shrink-0 rounded border-white/20"
                          />
                        )}
                        <span className="truncate">
                          {t.num}. {t.name}
                        </span>
                      </label>
                      <span className="flex shrink-0 items-center gap-2">
                        {choice === 'skip' && onDisk && (
                          <span className="text-emerald-400">skip</span>
                        )}
                        {choice === 'redownload' && (
                          <span className="text-amber-300">re-dl</span>
                        )}
                        {onDisk ? (
                          <button
                            type="button"
                            className="text-emerald-400 hover:underline"
                            onClick={() => dup?.existing_path && RevealInFolder(dup.existing_path)}
                          >
                            on disk
                          </button>
                        ) : (
                          <span className="text-white/35">{choice === 'skip' ? 'skip' : 'missing'}</span>
                        )}
                      </span>
                    </li>
                  )
                })
              )}
            </ul>
          )}
        </div>
      )}
    </li>
  )
}

export default function BulkDownloadQueue({
  quality,
  qualityOptions,
  settings,
  hasToken,
  downloading,
  jobStarted,
  engineEvents,
  onDownloadStart,
  onDownloadEnd,
  onQualityChange,
  addUrlRequest,
}) {
  const [pasteText, setPasteText] = useState('')
  const [queue, setQueue] = useState([])
  const [loadingQueue, setLoadingQueue] = useState(false)
  const [fetchError, setFetchError] = useState('')
  const [activeJobPlan, setActiveJobPlan] = useState(null)
  const [expandedId, setExpandedId] = useState(null)
  const [queueFilter, setQueueFilter] = useState('all')
  const [trackFilter, setTrackFilter] = useState('all')
  const [editingId, setEditingId] = useState(null)
  const [editUrlDraft, setEditUrlDraft] = useState('')
  const [compareItemId, setCompareItemId] = useState(null)
  const [showPasteArea, setShowPasteArea] = useState(true)
  const autoSwitchedIssues = useRef(false)

  const queueProgress = useMemo(
    () => (downloading || jobStarted ? parseBulkQueueProgress(engineEvents) : null),
    [downloading, jobStarted, engineEvents],
  )

  const jobResult = useMemo(() => {
    if (downloading || !jobStarted) return null
    return parseJobResult(engineEvents)
  }, [downloading, jobStarted, engineEvents])

  const readyItems = useMemo(() => queue.filter((q) => q.status === 'ready'), [queue])

  const itemDownloadPhase = useMemo(() => {
    if (!activeJobPlan?.plan?.length || !queueProgress?.current) return {}
    const map = {}
    for (const entry of activeJobPlan.plan) {
      map[entry.itemId] = queueItemDownloadStatus(entry, queueProgress.current)
    }
    return map
  }, [activeJobPlan, queueProgress])

  const filterCounts = useMemo(() => countQueueFilters(queue, itemDownloadPhase), [queue, itemDownloadPhase])

  const visibleFilters = useMemo(() => {
    if (!downloading && !jobStarted) {
      return QUEUE_FILTERS.filter((f) => !['downloading', 'done'].includes(f.id))
    }
    return QUEUE_FILTERS
  }, [downloading, jobStarted])

  const filteredQueue = useMemo(
    () => filterQueueItems(queue, queueFilter, itemDownloadPhase),
    [queue, queueFilter, itemDownloadPhase],
  )

  const duplicateTracks = useMemo(() => collectDuplicateTracks(queue), [queue])
  const albumsWithDupes = useMemo(() => countAlbumsWithDuplicates(queue), [queue])
  const queueDownloadPlan = useMemo(() => buildBulkDownloadEntries(queue), [queue])
  const compareItem = useMemo(
    () => queue.find((q) => q.id === compareItemId) || null,
    [queue, compareItemId],
  )
  const fullyOnDiskAlbumCount = useMemo(
    () => queue.filter((item) => item.status === 'ready' && isFullySkippedAlbum(item)).length,
    [queue],
  )

  const jumpToQueueItem = (itemId) => {
    setQueueFilter('on_disk')
    setExpandedId(itemId)
    setTrackFilter('on_disk')
  }

  const removeFullyOnDiskAlbums = () => {
    if (downloading) return
    setQueue((prev) => prev.filter((item) => !(item.status === 'ready' && isFullySkippedAlbum(item))))
  }

  const excludeAlbumFromQueue = (id) => {
    if (downloading) return
    removeItem(id)
    setCompareItemId(null)
  }

  const handleTrackPolicySave = (itemId, numOrBulk, choice) => {
    setQueue((prev) =>
      prev.map((item) => {
        if (item.id !== itemId) return item
        if (numOrBulk === '_all_dupes') return setAllDupesPolicy(item, choice)
        return setTrackPolicy(item, numOrBulk, choice)
      }),
    )
  }

  const setAllTracksIncluded = (itemId, included) => {
    if (downloading) return
    setQueue((prev) =>
      prev.map((item) => {
        if (item.id !== itemId) return item
        const trackPolicy = { ...(item.trackPolicy || {}) }
        for (const t of item.preview?.tracks || []) {
          if (included) {
            if (!isTrackOnDisk(item, t.num)) {
              delete trackPolicy[t.num]
            }
          } else {
            trackPolicy[t.num] = 'skip'
          }
        }
        return { ...item, trackPolicy }
      }),
    )
  }

  const openCompare = (item) => {
    setCompareItemId(item.id)
  }

  const duplicateFoldersKey = JSON.stringify(settings?.['duplicate-check-folders'] || [])

  const loadPreviews = useCallback(async (items) => {
    setLoadingQueue(true)
    setFetchError('')
    try {
      await mapPool(items, CONCURRENCY, async (item) => {
        setQueue((prev) =>
          prev.map((q) => (q.id === item.id ? { ...q, status: 'loading', error: '', duplicateInfo: null } : q)),
        )
        try {
          const preview = await PreviewURL(item.url)
          if (preview?.error) {
            setQueue((prev) =>
              prev.map((q) =>
                q.id === item.id ? { ...q, status: 'error', error: preview.error, preview: null } : q,
              ),
            )
            return
          }
          setQueue((prev) =>
            prev.map((q) => {
              if (q.id !== item.id) return q
              let next = { ...q, status: 'ready', error: '', preview }
              if (preview.url && preview.url !== q.url) {
                next.url = preview.url
              }
              if (q.pendingSelectedNums?.length && preview.tracks?.length) {
                next = applyTrackSelectionPolicy(next, q.pendingSelectedNums)
                next.pendingSelectedNums = null
              }
              return next
            }),
          )
        } catch (e) {
          setQueue((prev) =>
            prev.map((q) =>
              q.id === item.id
                ? { ...q, status: 'error', error: formatActionError(e, 'Fetch preview'), preview: null }
                : q,
            ),
          )
        }
      })
    } finally {
      setLoadingQueue(false)
    }
  }, [])

  useEffect(() => {
    if (loadingQueue || autoSwitchedIssues.current) return
    if (filterCounts.issues > 0 && queueFilter === 'all') {
      setQueueFilter('issues')
      autoSwitchedIssues.current = true
    }
  }, [filterCounts.issues, loadingQueue, queueFilter])

  const addUrlsFromText = async () => {
    const urls = parseBulkUrls(pasteText)
    if (urls.length === 0) {
      setFetchError('Paste one or more Apple Music links (album, playlist, song, or artist).')
      return
    }
    setFetchError('')
    const existing = new Set(queue.map((q) => q.url.toLowerCase()))
    const fresh = urls.filter((u) => !existing.has(u.toLowerCase())).map(newQueueItem)
    if (fresh.length === 0) {
      setFetchError('Those links are already in the queue.')
      return
    }
    setQueue((prev) => [...prev, ...fresh])
    setPasteText('')
    await loadPreviews(fresh)
  }

  const removeItem = (id) => {
    if (downloading) return
    setQueue((prev) => prev.filter((q) => q.id !== id))
    if (editingId === id) {
      setEditingId(null)
      setEditUrlDraft('')
    }
  }

  const clearQueue = () => {
    if (downloading) return
    setQueue([])
    setActiveJobPlan(null)
    setFetchError('')
    setQueueFilter('all')
    setEditingId(null)
    autoSwitchedIssues.current = false
  }

  const retryItem = async (item) => {
    if (downloading) return
    await loadPreviews([item])
  }

  const startEditUrl = (item) => {
    if (downloading) return
    setEditingId(item.id)
    setEditUrlDraft(item.url)
    setFetchError('')
  }

  const cancelEditUrl = () => {
    setEditingId(null)
    setEditUrlDraft('')
  }

  const saveEditUrl = async (id) => {
    const trimmed = editUrlDraft.trim()
    if (!isAppleMusicURL(trimmed)) {
      setFetchError('Enter a valid Apple Music URL (music.apple.com).')
      return
    }
    if (queue.some((q) => q.id !== id && q.url.toLowerCase() === trimmed.toLowerCase())) {
      setFetchError('That URL is already in the queue.')
      return
    }
    const item = queue.find((q) => q.id === id)
    if (!item) return
    const updated = {
      ...item,
      url: trimmed,
      status: 'pending',
      error: '',
      preview: null,
      duplicateInfo: null,
    }
    setQueue((prev) => prev.map((q) => (q.id === id ? updated : q)))
    setEditingId(null)
    setEditUrlDraft('')
    setFetchError('')
    await loadPreviews([updated])
  }

  useEffect(() => {
    if (!addUrlRequest?.url) return
    const url = addUrlRequest.url.trim()
    if (!url) return
    if (queue.some((q) => q.url.toLowerCase() === url.toLowerCase())) return
    const item = newQueueItem(url, { selectedNums: addUrlRequest.selectedNums })
    setQueue((prev) => [...prev, item])
    loadPreviews([item])
  }, [addUrlRequest?.ts])

  useEffect(() => {
    if (!readyItems.length || downloading) return undefined
    let cancelled = false
    const timer = setTimeout(async () => {
      for (const item of readyItems) {
        if (cancelled) break
        try {
          const res = await CheckTrackDuplicates(item.url, quality, [], 'audio', 'apple', [], item.preview)
          if (cancelled) return
          setQueue((prev) => prev.map((q) => (q.id === item.id ? { ...q, duplicateInfo: res } : q)))
        } catch (e) {
          reportFrontendError('BulkDownloadQueue.duplicates', e)
        }
      }
    }, 400)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [readyItems.map((i) => i.id).join(','), quality, duplicateFoldersKey, downloading])

  const startBulkDownload = async () => {
    const { entries, stats } = queueDownloadPlan
    if (downloading || entries.length === 0) {
      if (entries.length === 0 && readyItems.length > 0) {
        setFetchError('Every track is set to skip — adjust choices in Compare or add more albums.')
      }
      return
    }

    setFetchError('')
    try {
      const check = await PreflightDownloadJob(entries[0].url, quality, 'audio', 'apple')
      if (!check?.ready) {
        const failed = (check.checks || []).filter((c) => !c.ok && c.blocking).map((c) => c.detail || c.label)
        setFetchError(failed[0] || check.summary || 'Fix requirements before downloading.')
        return
      }
    } catch (e) {
      setFetchError(formatActionError(e, 'Pre-flight check'))
      return
    }

    const { plan } = buildBulkDownloadPlan(readyItems.filter((i) => !isFullySkippedAlbum(i)))
    setActiveJobPlan({ plan, flatUrls: entries.map((e) => e.url), stats })
    setQueueFilter('downloading')
    onDownloadStart?.()
    try {
      await StartBulkDownloadJob(entries, quality)
    } catch (err) {
      const msg = typeof err === 'string' ? err : err?.message || String(err)
      setFetchError(msg)
      onDownloadEnd?.(null)
      setActiveJobPlan(null)
    }
  }

  useEffect(() => {
    if (!downloading && jobStarted) {
      onDownloadEnd?.(parseJobResult(engineEvents))
      if (filterCounts.issues > 0) setQueueFilter('issues')
    }
  }, [downloading, jobStarted, engineEvents, onDownloadEnd])

  const jobMeta = jobResult ? jobStatusMeta(jobResult.phase) : null

  const bulkFlowPhase = useMemo(() => {
    if (downloading) return 'downloading'
    if (jobStarted && jobResult) return 'done'
    if (queue.length > 0) return 'review'
    return 'link'
  }, [downloading, jobStarted, jobResult, queue.length])

  useEffect(() => {
    if (bulkFlowPhase === 'downloading' || bulkFlowPhase === 'done') {
      setShowPasteArea(false)
    }
  }, [bulkFlowPhase])

  const bulkFooter =
    bulkFlowPhase === 'link' ? (
      <button
        type="button"
        onClick={addUrlsFromText}
        disabled={!pasteText.trim() || downloading || loadingQueue}
        className="w-full rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
      >
        {loadingQueue ? 'Loading previews…' : 'Add to queue'}
      </button>
    ) : bulkFlowPhase === 'review' ? (
      <button
        type="button"
        onClick={startBulkDownload}
        disabled={downloading || queueDownloadPlan.entries.length === 0 || loadingQueue}
        className="w-full rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted disabled:opacity-40"
      >
        {queueDownloadPlan.entries.length === 0
          ? 'Nothing to download (all skipped)'
          : `Download queue · ${queueDownloadPlan.stats.albums} album${queueDownloadPlan.stats.albums !== 1 ? 's' : ''} · ${queueDownloadPlan.stats.tracksToDownload} track${queueDownloadPlan.stats.tracksToDownload !== 1 ? 's' : ''}`}
      </button>
    ) : bulkFlowPhase === 'downloading' ? (
      <p className="w-full py-3 text-center text-sm font-medium text-accent">
        Downloading queue ({queueProgress?.current || 0}/{queueProgress?.total || queueDownloadPlan.stats.urls})…
      </p>
    ) : bulkFlowPhase === 'done' ? (
      <div className="flex flex-col gap-2 sm:flex-row">
        <button
          type="button"
          onClick={() => {
            setShowPasteArea(true)
            setQueueFilter('all')
          }}
          className="flex-1 rounded-xl bg-accent py-3 font-semibold hover:bg-accent-muted"
        >
          Add more links
        </button>
        <button
          type="button"
          onClick={clearQueue}
          className="flex-1 rounded-xl border border-white/15 bg-white/[0.04] py-3 font-semibold text-white/90 hover:bg-white/[0.08]"
        >
          Clear queue
        </button>
      </div>
    ) : null

  return (
    <DownloadFlowLayout
      title="Bulk download queue"
      subtitle={
        bulkFlowPhase === 'link'
          ? 'Paste many Apple Music links — previews load in parallel'
          : bulkFlowPhase === 'review'
            ? 'Review queue, fix issues, then download in one run'
            : bulkFlowPhase === 'downloading'
              ? 'Queue downloading sequentially — see progress below'
              : 'Queue finished — add more links or clear the queue'
      }
      phase={bulkFlowPhase}
      stepperVariant="bulk"
      footer={bulkFooter}
      footerNote={
        bulkFlowPhase === 'review'
          ? 'Duplicates default to skip. Use Compare on album rows to re-download or exclude tracks.'
          : undefined
      }
    >
      {!hasToken && bulkFlowPhase !== 'downloading' && (
        <p className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-200">
          AAC downloads need your media-user-token in Settings.
        </p>
      )}

      {(bulkFlowPhase === 'link' || showPasteArea) && (
        <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
          <label className="text-xs font-medium uppercase tracking-wide text-white/45">Paste links</label>
          <textarea
            value={pasteText}
            onChange={(e) => setPasteText(e.target.value)}
            disabled={downloading || loadingQueue}
            rows={4}
            placeholder={`https://music.apple.com/us/album/worlds/1440857923\nhttps://music.apple.com/us/album/nurture/...\n(one per line)`}
            className="mt-2 w-full resize-y rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm focus:border-accent focus:outline-none disabled:opacity-50"
          />
          {bulkFlowPhase !== 'link' && (
            <button
              type="button"
              onClick={() => setShowPasteArea(false)}
              className="mt-2 text-xs text-white/45 hover:text-white/70"
            >
              Hide paste area
            </button>
          )}
        </div>
      )}

      {fetchError && (
        <p className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">{fetchError}</p>
      )}

      {jobResult && jobMeta && !downloading && (
        <div className={`rounded-xl border px-4 py-3 ${jobMeta.className}`}>
          <p className="font-semibold">{jobMeta.label}</p>
          <p className="mt-1 text-sm opacity-90">{jobResult.message}</p>
        </div>
      )}

      {queue.length > 0 && (
        <div className="rounded-xl border border-white/10 bg-surface-raised">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
            <div>
              <p className="text-sm font-medium">
                Queue · {readyItems.length} ready
                {filterCounts.duplicates > 0 && (
                  <span className="ml-2 text-emerald-400">· {filterCounts.duplicates} duplicate tracks</span>
                )}
                {filterCounts.issues > 0 && (
                  <span className="ml-2 text-red-300">· {filterCounts.issues} need attention</span>
                )}
              </p>
              {downloading && queueProgress && (
                <p className="mt-0.5 text-xs text-accent">
                  {queueProgress.label || `Item ${queueProgress.current} of ${queueProgress.total}`}
                </p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <label className="text-xs text-white/50">Quality</label>
              <select
                value={quality}
                onChange={(e) => onQualityChange?.(e.target.value)}
                disabled={downloading}
                className="rounded-lg border border-white/10 bg-surface px-2 py-1 text-sm"
              >
                {qualityOptions.map((q) => (
                  <option key={q.id} value={q.id}>
                    {q.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <FilterTabs
            tabs={visibleFilters}
            active={queueFilter}
            counts={filterCounts}
            onChange={setQueueFilter}
          />

          {downloading && queueProgress?.total > 0 && (
            <div className="border-b border-white/10 px-4 py-2">
              <div className="h-1.5 overflow-hidden rounded-full bg-black/30">
                <div
                  className="h-full rounded-full bg-accent transition-all duration-300"
                  style={{ width: `${Math.round((queueProgress.current / queueProgress.total) * 100)}%` }}
                />
              </div>
            </div>
          )}

          {queueFilter === 'duplicates' ? (
            <DuplicateTracksPanel
              tracks={duplicateTracks}
              albumCount={albumsWithDupes}
              onJumpToItem={jumpToQueueItem}
              onCompareAlbum={(id) => setCompareItemId(id)}
              onRemoveFullyOnDisk={removeFullyOnDiskAlbums}
              fullyOnDiskCount={fullyOnDiskAlbumCount}
              downloading={downloading}
            />
          ) : filteredQueue.length === 0 ? (
            <p className="px-4 py-8 text-center text-sm text-white/45">
              No items in this view.{' '}
              <button type="button" onClick={() => setQueueFilter('all')} className="text-accent hover:underline">
                Show all
              </button>
            </p>
          ) : (
            <ul className="max-h-[min(420px,50vh)] divide-y divide-white/5 overflow-y-auto">
              {filteredQueue.map((item) => (
                <BulkQueueItemRow
                  key={item.id}
                  item={item}
                  downloading={downloading}
                  dlPhase={itemDownloadPhase[item.id] || 'waiting'}
                  isExpanded={expandedId === item.id}
                  trackFilter={trackFilter}
                  editingId={editingId}
                  editUrlDraft={editUrlDraft}
                  onToggleExpand={(id) => {
                    setExpandedId((prev) => (prev === id ? null : id))
                    setTrackFilter('all')
                  }}
                  onTrackFilterChange={setTrackFilter}
                  onStartEditUrl={startEditUrl}
                  onEditDraftChange={setEditUrlDraft}
                  onSaveEditUrl={saveEditUrl}
                  onCancelEdit={cancelEditUrl}
                  onRetry={retryItem}
                  onRemove={removeItem}
                  onCompare={openCompare}
                  showCompare={queueFilter === 'on_disk' || (item.duplicateInfo?.existing_count ?? 0) > 0}
                  onTrackPolicyChange={handleTrackPolicySave}
                  onSelectAllTracks={setAllTracksIncluded}
                />
              ))}
            </ul>
          )}

          <div className="border-t border-white/10 px-4 py-4">
            {(queueDownloadPlan.stats.tracksSkipped > 0 ||
              queueDownloadPlan.stats.tracksRedownload > 0 ||
              queueDownloadPlan.stats.albumsFullySkipped > 0) &&
              !downloading &&
              bulkFlowPhase === 'review' && (
              <p className="rounded-lg border border-emerald-500/25 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-100">
                Download plan: <strong className="font-medium">{queueDownloadPlan.stats.tracksToDownload}</strong> track
                {queueDownloadPlan.stats.tracksToDownload !== 1 ? 's' : ''} will download
                {queueDownloadPlan.stats.tracksSkipped > 0 && (
                  <>
                    {' '}
                    · <strong className="font-medium">{queueDownloadPlan.stats.tracksSkipped}</strong> skipped (already on
                    disk)
                  </>
                )}
                {queueDownloadPlan.stats.tracksRedownload > 0 && (
                  <>
                    {' '}
                    · <strong className="font-medium">{queueDownloadPlan.stats.tracksRedownload}</strong> re-download
                  </>
                )}
                {queueDownloadPlan.stats.albumsFullySkipped > 0 && (
                  <>
                    {' '}
                    · <strong className="font-medium">{queueDownloadPlan.stats.albumsFullySkipped}</strong> album
                    {queueDownloadPlan.stats.albumsFullySkipped !== 1 ? 's' : ''} fully skipped
                  </>
                )}
              </p>
            )}
            {filterCounts.issues > 0 && !downloading && bulkFlowPhase === 'review' && (
              <p className="rounded-lg border border-yellow-500/25 bg-yellow-500/10 px-3 py-2 text-xs text-yellow-100">
                {filterCounts.issues} item{filterCounts.issues !== 1 ? 's have' : ' has'} preview errors — use{' '}
                <strong className="font-medium">Issues</strong> filter, edit the URL, then Save &amp; refetch.
              </p>
            )}
          </div>
        </div>
      )}

      <BulkAlbumCompareModal
        item={compareItem}
        open={Boolean(compareItemId && compareItem)}
        onClose={() => setCompareItemId(null)}
        onSave={handleTrackPolicySave}
        onRemoveAlbum={excludeAlbumFromQueue}
      />

      {queue.length === 0 && !loadingQueue && bulkFlowPhase === 'link' && (
        <p className="rounded-lg border border-white/10 bg-white/[0.02] px-3 py-2 text-sm text-white/55">
          Example: paste several album links, then use filters to spot errors or tracks already on disk.
        </p>
      )}
    </DownloadFlowLayout>
  )
}
