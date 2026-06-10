/** Merge per-track preview patches (e.g. after swapping a song URL). */
export function mergeAppleTrackDisplay(tracks, patches = {}) {
  if (!tracks?.length) return []
  return tracks.map((t) => ({
    ...t,
    ...(patches[t.num] || {}),
    url: patches[t.num]?.url ?? t.url,
  }))
}

/** Build backend track URL overrides for playlist/album rows. */
export function collectTrackURLOverrides(tracks, originalTracks) {
  const origByNum = Object.fromEntries((originalTracks || []).map((t) => [t.num, t.url]))
  const overrides = []
  for (const t of tracks || []) {
    const url = String(t.url || '').trim()
    const orig = String(origByNum[t.num] || '').trim()
    if (url && orig && url !== orig) {
      overrides.push({ num: t.num, url })
    }
  }
  return overrides
}
