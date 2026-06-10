/** @returns {'track'|'album'|'playlist'|null} */
export function parseSpotifyLinkKind(raw) {
  const t = String(raw || '').trim()
  const m = t.match(/spotify\.com\/(track|album|playlist)\//i) || t.match(/^spotify:(track|album|playlist):/i)
  if (!m) return null
  return m[1].toLowerCase()
}

export function spotifyUnsupportedKind(raw) {
  const kind = parseSpotifyLinkKind(raw)
  if (kind === 'playlist' || kind === 'album') return kind
  return null
}

export function spotifyUnsupportedMessage(kind) {
  if (kind === 'playlist') {
    return 'Spotify playlists are not supported. Build the list on Apple Music first (e.g. with a transfer tool), then paste the Apple Music playlist link here.'
  }
  if (kind === 'album') {
    return 'Spotify albums are not supported. Paste one Spotify track link at a time, or use the matching Apple Music album link.'
  }
  return 'That Spotify link type is not supported.'
}

export function formatUnmatchedForClipboard(items) {
  return (items || [])
    .filter((item) => item.match_status === 'not_found')
    .map((item) => `${item.spotify_title} — ${item.spotify_artist}`)
    .join('\n')
}

export function indicesForStatus(items, statuses) {
  const set = new Set(statuses)
  const out = []
  ;(items || []).forEach((item, idx) => {
    if (set.has(item.match_status) && item.apple_hit?.url) out.push(idx)
  })
  return out
}

export function collectUrlsFromIndices(items, indices) {
  const urls = []
  indices.forEach((idx) => {
    const url = items[idx]?.apple_hit?.url
    if (url) urls.push(url)
  })
  return urls
}
