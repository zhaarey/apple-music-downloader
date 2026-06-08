/** Ensures project shape is safe for React rendering after Wails round-trips. */
export function normalizeProject(project, fallback = {}) {
  const base = fallback && typeof fallback === 'object' ? fallback : {}
  const p = project && typeof project === 'object' ? project : {}

  const tracks = Array.isArray(p.tracks)
    ? p.tracks.map((t, i) => normalizeTrack(t, i))
    : Array.isArray(base.tracks)
      ? base.tracks.map((t, i) => normalizeTrack(t, i))
      : []

  return {
    ...base,
    ...p,
    master_path: String(p.master_path ?? base.master_path ?? ''),
    output_dir: String(p.output_dir ?? base.output_dir ?? ''),
    master_duration_ms: toMs(p.master_duration_ms ?? base.master_duration_ms),
    tracks,
    album: {
      ...(base.album || {}),
      ...(p.album || {}),
    },
  }
}

function normalizeTrack(track, index) {
  const t = track && typeof track === 'object' ? track : {}
  const durationMs = toMs(t.duration_ms, 60000)
  const startMs = t.start_ms != null ? toMs(t.start_ms) : undefined
  return {
    title: String(t.title ?? `Track ${index + 1}`),
    artist: t.artist != null ? String(t.artist) : '',
    duration_ms: durationMs > 0 ? durationMs : 1000,
    ...(startMs != null ? { start_ms: startMs } : {}),
    ...(t.track_number != null ? { track_number: toInt(t.track_number, index + 1) } : {}),
    album: t.album != null ? String(t.album) : undefined,
    album_artist: t.album_artist != null ? String(t.album_artist) : undefined,
    genre: t.genre != null ? String(t.genre) : undefined,
    year: t.year != null ? String(t.year) : undefined,
    disc_number: t.disc_number != null ? toInt(t.disc_number) : undefined,
    disc_total: t.disc_total != null ? toInt(t.disc_total) : undefined,
  }
}

function toMs(value, fallback = 0) {
  const n = Number(value)
  return Number.isFinite(n) ? Math.max(0, Math.round(n)) : fallback
}

function toInt(value, fallback = null) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? Math.round(n) : fallback
}

export function formatActionError(error, action) {
  const msg = error?.message || String(error ?? 'Unknown error')
  return `${action} failed: ${msg}`
}
