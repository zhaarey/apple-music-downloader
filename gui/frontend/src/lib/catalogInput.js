const APPLE_MUSIC_RE = /https?:\/\/(?:music\.apple\.com|itunes\.apple\.com)\/\S+/gi
const SPOTIFY_RE = /https?:\/\/(?:open\.)?spotify\.com\/(?:track|album|playlist)\/[a-zA-Z0-9]+(?:\?\S*)?/gi

/** @typedef {'apple' | 'spotify' | 'search'} CatalogInputKind */

export function detectCatalogInput(raw) {
  const t = String(raw || '').trim()
  if (!t) return 'search'
  if (SPOTIFY_RE.test(t) || /^spotify:(track|album|playlist):/i.test(t)) return 'spotify'
  if (APPLE_MUSIC_RE.test(t) || /music\.apple\.com|itunes\.apple\.com/i.test(t)) return 'apple'
  return 'search'
}

export function extractAppleMusicUrls(text) {
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

export function inputKindLabel(kind) {
  switch (kind) {
    case 'spotify':
      return 'Spotify link'
    case 'apple':
      return 'Apple Music link'
    default:
      return 'Search'
  }
}

export function inputKindHint(kind, bulkMode) {
  switch (kind) {
    case 'spotify':
      return bulkMode
        ? 'Paste a public Spotify playlist link — we’ll match songs on Apple Music for your download queue.'
        : 'We’ll match this track on Apple Music (ISRC when available).'
    case 'apple':
      return bulkMode
        ? 'We’ll add these Apple Music links to your queue and load previews.'
        : 'We’ll open a preview so you can confirm before downloading.'
    default:
      return 'Search the Apple Music catalog — or paste a link anytime.'
  }
}

export function lookupButtonLabel(kind, loading) {
  if (loading) return 'Working…'
  if (kind === 'spotify') return 'Find on Apple Music'
  if (kind === 'apple') return 'Continue'
  return 'Search'
}

export const LOADING_MESSAGES = {
  apple: ['Adding your Apple Music link…'],
  search: ['Searching Apple Music…'],
  spotify: [
    'Reading your Spotify link…',
    'Matching by ISRC on Apple Music…',
    'Checking titles for anything ISRC missed…',
  ],
}

export function matchStatusMeta(status) {
  switch (status) {
    case 'matched':
      return {
        label: 'Ready',
        hint: 'Confident match — safe to queue.',
        className: 'text-emerald-300 bg-emerald-500/15 border-emerald-500/30',
      }
    case 'uncertain':
      return {
        label: 'Check this',
        hint: 'Close match — give it a quick look.',
        className: 'text-amber-200 bg-amber-500/15 border-amber-500/30',
      }
    default:
      return {
        label: 'No match',
        hint: 'Not on Apple Music or too different to auto-match.',
        className: 'text-red-300 bg-red-500/15 border-red-500/30',
      }
  }
}

export function matchMethodLabel(method) {
  if (method === 'isrc') return 'ISRC match'
  if (method === 'catalog') return 'Title search'
  return ''
}

export function spotifySummary(result) {
  if (!result) return ''
  const parts = []
  if (result.isrc_matched > 0) parts.push(`${result.isrc_matched} by ISRC`)
  if (result.matched - (result.isrc_matched || 0) > 0) {
    parts.push(`${result.matched - (result.isrc_matched || 0)} by title`)
  }
  if (result.uncertain > 0) parts.push(`${result.uncertain} to review`)
  if (result.missing > 0) parts.push(`${result.missing} not found`)
  return parts.join(' · ')
}

export function defaultSelectedMatchIndices(items) {
  const selected = new Set()
  ;(items || []).forEach((item, idx) => {
    if (item.match_status === 'matched' && item.apple_hit?.url) {
      selected.add(idx)
    }
  })
  return selected
}
