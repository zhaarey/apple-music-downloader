export const MIGRATION_STEPS = [
  {
    title: 'Make playlist public on Spotify',
    detail: 'Open the playlist → ⋯ menu → Make public. Private playlists cannot be read.',
  },
  {
    title: 'Add Spotify API keys (one-time)',
    detail: 'Settings → Spotify matching. Free developer account — needed to read playlist track lists.',
  },
  {
    title: 'Paste playlist link here',
    detail: 'We match each song on Apple Music using ISRC, then you add them to the queue.',
  },
  {
    title: 'Download queue',
    detail: 'Review the queue, then run Download queue — same flow as any Apple Music bulk job.',
  },
]

export function hasSpotifyCredentials(settings) {
  const id = String(settings?.['spotify-client-id'] || '').trim()
  const secret = String(settings?.['spotify-client-secret'] || '').trim()
  return id.length > 0 && secret.length > 0
}

/** @returns {'track'|'album'|'playlist'|null} */
export function parseSpotifyLinkKind(raw) {
  const t = String(raw || '').trim()
  const m = t.match(/spotify\.com\/(track|album|playlist)\//i) || t.match(/^spotify:(track|album|playlist):/i)
  if (!m) return null
  return m[1].toLowerCase()
}

export function spotifyNeedsCredentials(raw, settings) {
  if (hasSpotifyCredentials(settings)) return false
  const kind = parseSpotifyLinkKind(raw)
  return kind === 'playlist' || kind === 'album'
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
