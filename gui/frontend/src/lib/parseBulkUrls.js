const APPLE_MUSIC_RE = /https?:\/\/(?:music\.apple\.com|itunes\.apple\.com)\/\S+/gi

/** Split pasted text into unique Apple Music URLs (one per line, comma, or semicolon). */
export function parseBulkUrls(text) {
  const raw = String(text || '')
  const found = raw.match(APPLE_MUSIC_RE) || []
  const seen = new Set()
  const out = []
  for (let u of found) {
    u = u.trim().replace(/[)\]}"'`,]+$/, '')
    const key = u.toLowerCase()
    if (!seen.has(key)) {
      seen.add(key)
      out.push(u)
    }
  }
  return out
}

export function isAppleMusicURL(raw) {
  return /music\.apple\.com|itunes\.apple\.com/i.test(String(raw || '').trim())
}

export function newQueueItem(url, { selectedNums } = {}) {
  return {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    url,
    status: 'pending',
    error: '',
    preview: null,
    duplicateInfo: null,
    /** @type {Record<number, 'skip'|'download'|'redownload'>} */
    trackPolicy: {},
    excludedFromDownload: false,
    pendingSelectedNums: selectedNums?.length ? [...selectedNums] : null,
  }
}

/** Apply an explicit track selection (e.g. from single-download tab) onto queue track policy. */
export function applyTrackSelectionPolicy(item, selectedNums) {
  const tracks = item.preview?.tracks || []
  if (!tracks.length || !selectedNums?.length) return item
  const selectedSet = new Set(selectedNums)
  const trackPolicy = { ...(item.trackPolicy || {}) }
  for (const t of tracks) {
    if (!selectedSet.has(t.num)) {
      trackPolicy[t.num] = 'skip'
    } else if (trackPolicy[t.num] === 'skip' && !isTrackOnDisk(item, t.num)) {
      delete trackPolicy[t.num]
    }
  }
  return { ...item, trackPolicy }
}

export function isTrackOnDisk(item, num) {
  return Boolean(item.duplicateInfo?.tracks?.find((d) => d.num === num && d.on_disk))
}

export function getDupTrackInfo(item, num) {
  return item.duplicateInfo?.tracks?.find((d) => d.num === num)
}

/** Effective choice for one track (explicit policy or default from duplicate scan). */
export function getTrackChoice(item, num) {
  if (item.trackPolicy?.[num]) return item.trackPolicy[num]
  return isTrackOnDisk(item, num) ? 'skip' : 'download'
}

export function setTrackPolicy(item, num, choice) {
  return { ...item, trackPolicy: { ...item.trackPolicy, [num]: choice } }
}

export function setAllDupesPolicy(item, choice) {
  const next = { ...item.trackPolicy }
  for (const t of item.preview?.tracks || []) {
    if (isTrackOnDisk(item, t.num)) next[t.num] = choice
  }
  return { ...item, trackPolicy: next }
}

export function summarizeItemTracks(item) {
  const tracks = item.preview?.tracks || []
  let download = 0
  let skip = 0
  let redownload = 0
  for (const t of tracks) {
    const c = getTrackChoice(item, t.num)
    if (c === 'skip') skip++
    else if (c === 'redownload') redownload++
    else download++
  }
  return { total: tracks.length, download, skip, redownload }
}

export function isFullySkippedAlbum(item) {
  const s = summarizeItemTracks(item)
  return s.total > 0 && s.download === 0 && s.redownload === 0
}

export function buildBulkDownloadEntries(queue) {
  const entries = []
  const stats = {
    albums: 0,
    urls: 0,
    tracksToDownload: 0,
    tracksSkipped: 0,
    tracksRedownload: 0,
    albumsExcluded: 0,
    albumsFullySkipped: 0,
  }

  for (const item of queue) {
    if (item.status !== 'ready' || item.excludedFromDownload) {
      if (item.excludedFromDownload) stats.albumsExcluded++
      continue
    }

    const urls = expandQueueItemUrls(item)
    const tracks = item.preview?.tracks || []
    const selectedNums = []
    const forceNums = []

    if (tracks.length > 0) {
      for (const t of tracks) {
        const choice = getTrackChoice(item, t.num)
        if (choice === 'skip') {
          stats.tracksSkipped++
        } else if (choice === 'redownload') {
          selectedNums.push(t.num)
          forceNums.push(t.num)
          stats.tracksRedownload++
          stats.tracksToDownload++
        } else {
          selectedNums.push(t.num)
          stats.tracksToDownload++
        }
      }
      if (selectedNums.length === 0) {
        stats.albumsFullySkipped++
        continue
      }
    }

    stats.albums++
    for (const url of urls) {
      entries.push({
        url: item.preview?.url || item.url,
        selected_track_nums: tracks.length > 0 ? selectedNums : [],
        force_track_nums: forceNums,
      })
      stats.urls++
    }
  }

  return { entries, stats }
}

export function compareRowsForItem(item) {
  return (item.preview?.tracks || []).map((t) => {
    const dup = getDupTrackInfo(item, t.num)
    const choice = getTrackChoice(item, t.num)
    return {
      num: t.num,
      name: t.name,
      artist: t.artist,
      onDisk: Boolean(dup?.on_disk),
      existingPath: dup?.existing_path || '',
      existingRoot: dup?.existing_root_label || '',
      expectedPath: dup?.expected_path || '',
      choice,
    }
  })
}

/** URLs sent to the backend for one queue row (artist → many album URLs). */
export function expandQueueItemUrls(item) {
  if (!item?.preview) return []
  if (item.preview.type === 'Artist') {
    return (item.preview.tracks || []).map((t) => t.url).filter(Boolean)
  }
  return [item.url]
}

export function buildBulkDownloadPlan(readyItems) {
  const plan = []
  const flatUrls = []
  for (const item of readyItems) {
    const urls = expandQueueItemUrls(item)
    if (urls.length === 0) continue
    const entry = { itemId: item.id, urls, startIndex: flatUrls.length }
    plan.push(entry)
    flatUrls.push(...urls)
  }
  return { plan, flatUrls }
}

export function queueItemDownloadStatus(entry, currentOneBased) {
  if (!currentOneBased || currentOneBased < 1) return 'waiting'
  const idx = currentOneBased - 1
  const end = entry.startIndex + entry.urls.length - 1
  if (idx > end) return 'done'
  if (idx >= entry.startIndex) return 'downloading'
  return 'waiting'
}

/** Bucket for queue list filters. */
export function queueItemFilterGroup(item, downloadPhase) {
  if (item.status === 'error') return 'issues'
  if (item.status === 'loading' || item.status === 'pending') return 'loading'
  if (downloadPhase === 'done') return 'done'
  if (downloadPhase === 'downloading') return 'downloading'
  if (item.status === 'ready') {
    const dup = item.duplicateInfo?.existing_count ?? 0
    const selected = item.duplicateInfo?.selected_count ?? 0
    if (dup > 0 && selected > 0 && dup >= selected) return 'on_disk'
    if (dup > 0) return 'partial_disk'
    return 'ready'
  }
  return 'all'
}

export function filterQueueItems(queue, filter, itemDownloadPhase) {
  if (filter === 'all') return queue
  return queue.filter((item) => {
    const phase = itemDownloadPhase[item.id]
    const group = queueItemFilterGroup(item, phase)
    if (filter === 'issues') return group === 'issues'
    if (filter === 'loading') return group === 'loading'
    if (filter === 'ready') return group === 'ready'
    if (filter === 'on_disk') return group === 'on_disk' || group === 'partial_disk'
    if (filter === 'downloading') return group === 'downloading'
    if (filter === 'done') return group === 'done'
    return true
  })
}

export function countQueueFilters(queue, itemDownloadPhase) {
  const counts = {
    all: queue.length,
    ready: 0,
    issues: 0,
    on_disk: 0,
    duplicates: countDuplicateTracks(queue),
    loading: 0,
    downloading: 0,
    done: 0,
  }
  for (const item of queue) {
    const group = queueItemFilterGroup(item, itemDownloadPhase[item.id])
    if (counts[group] !== undefined) counts[group]++
    if (group === 'partial_disk') counts.on_disk++
  }
  return counts
}

export function filterPreviewTracks(tracks, duplicateInfo, trackFilter) {
  if (!tracks?.length || trackFilter === 'all') return tracks
  return tracks.filter((t) => {
    const onDisk = duplicateInfo?.tracks?.find((d) => d.num === t.num)?.on_disk
    if (trackFilter === 'on_disk') return onDisk
    if (trackFilter === 'missing') return !onDisk
    return true
  })
}

/** Flat list of every track already on disk across the bulk queue. */
export function collectDuplicateTracks(queue) {
  const out = []
  for (const item of queue) {
    if (item.status !== 'ready' || !item.duplicateInfo?.tracks?.length) continue
    const album = item.preview?.title || 'Unknown album'
    const albumArtist = item.preview?.subtitle || ''
    for (const dup of item.duplicateInfo.tracks) {
      if (!dup.on_disk) continue
      const previewTrack = item.preview?.tracks?.find((p) => p.num === dup.num)
      out.push({
        id: `${item.id}-${dup.num}`,
        itemId: item.id,
        num: dup.num,
        name: previewTrack?.name || dup.expected_filename || `Track ${dup.num}`,
        artist: previewTrack?.artist || albumArtist,
        album,
        albumArt: item.preview?.art_url || '',
        path: dup.existing_path || '',
        root: dup.existing_root_label || '',
        expectedPath: dup.expected_path || '',
        itemUrl: item.url,
      })
    }
  }
  return out.sort((a, b) => {
    const albumCmp = a.album.localeCompare(b.album, undefined, { sensitivity: 'base' })
    if (albumCmp !== 0) return albumCmp
    return a.num - b.num
  })
}

export function countDuplicateTracks(queue) {
  return collectDuplicateTracks(queue).length
}

export function countAlbumsWithDuplicates(queue) {
  return queue.filter((item) => (item.duplicateInfo?.existing_count ?? 0) > 0).length
}
