const FIELD_CLASS =
  'mt-1 w-full rounded-lg border border-white/10 bg-surface px-2.5 py-1.5 text-sm focus:border-accent focus:outline-none'

function MetaField({ label, value, onChange, disabled, mono }) {
  return (
    <div>
      <label className="text-[11px] uppercase tracking-wide text-white/45">{label}</label>
      <input
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        className={`${FIELD_CLASS}${mono ? ' font-mono text-xs' : ''}`}
      />
    </div>
  )
}

export default function YouTubeMetadataEditor({ preview, selected, metaByTrack, onChange, onSharedChange, disabled }) {
  const selectedTracks = (preview?.tracks || []).filter((t) => selected.has(t.num))
  const first = selectedTracks[0] || preview?.tracks?.[0]
  const shared = first
    ? {
        album: metaByTrack[first.num]?.album || '',
        album_artist: metaByTrack[first.num]?.album_artist || '',
        genre: metaByTrack[first.num]?.genre || '',
        year: metaByTrack[first.num]?.year || '',
      }
    : { album: '', album_artist: '', genre: '', year: '' }

  const applyShared = (patch) => {
    onSharedChange(patch)
  }

  return (
    <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-sm font-medium">Apple Music metadata</h4>
          <p className="text-xs text-white/50">
            Edit before download · saved as AAC 256 kbps in Album folders
          </p>
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <MetaField
          label="Album"
          value={shared.album}
          onChange={(v) => applyShared({ album: v })}
          disabled={disabled}
        />
        <MetaField
          label="Album artist"
          value={shared.album_artist}
          onChange={(v) => applyShared({ album_artist: v })}
          disabled={disabled}
        />
        <MetaField
          label="Genre"
          value={shared.genre}
          onChange={(v) => applyShared({ genre: v })}
          disabled={disabled}
        />
        <MetaField
          label="Year"
          value={shared.year}
          onChange={(v) => applyShared({ year: v })}
          disabled={disabled}
          mono
        />
      </div>

      {selectedTracks.length > 1 && (
        <div className="mt-4 border-t border-white/10 pt-4">
          <p className="mb-2 text-xs font-medium text-white/60">Per-track titles</p>
          <ul className="max-h-56 space-y-2 overflow-y-auto">
            {selectedTracks.map((t) => {
              const meta = metaByTrack[t.num] || {}
              return (
                <li key={t.num} className="grid gap-2 rounded-lg bg-white/[0.02] p-2 sm:grid-cols-[2rem_1fr_1fr] sm:items-center">
                  <span className="text-xs text-white/40">{t.num}</span>
                  <input
                    value={meta.title || ''}
                    onChange={(e) => onChange(t.num, { title: e.target.value })}
                    disabled={disabled}
                    placeholder="Title"
                    className={FIELD_CLASS}
                  />
                  <input
                    value={meta.artist || ''}
                    onChange={(e) => onChange(t.num, { artist: e.target.value })}
                    disabled={disabled}
                    placeholder="Artist"
                    className={FIELD_CLASS}
                  />
                </li>
              )
            })}
          </ul>
        </div>
      )}

      {selectedTracks.length === 1 && first && (
        <div className="mt-4 grid gap-3 border-t border-white/10 pt-4 sm:grid-cols-2">
          <MetaField
            label="Title"
            value={metaByTrack[first.num]?.title || ''}
            onChange={(v) => onChange(first.num, { title: v })}
            disabled={disabled}
          />
          <MetaField
            label="Artist"
            value={metaByTrack[first.num]?.artist || ''}
            onChange={(v) => onChange(first.num, { artist: v })}
            disabled={disabled}
          />
        </div>
      )}
    </div>
  )
}

export function buildMetaFromPreview(preview) {
  const map = {}
  ;(preview?.tracks || []).forEach((t) => {
    map[t.num] = {
      num: t.num,
      title: t.name || '',
      artist: t.artist || '',
      album: t.album || preview?.title || '',
      album_artist: t.album_artist || t.artist || preview?.subtitle?.split(' · ')[0] || '',
      genre: t.genre || 'DJ Mix',
      year: t.year || String(new Date().getFullYear()),
      track_number: t.track_number || t.num,
      disc_number: t.disc_number || 1,
      art_url: t.art_url || preview?.art_url || '',
    }
  })
  return map
}

export function metaPayload(metaByTrack, selected) {
  const total = selected.size
  return [...selected]
    .sort((a, b) => a - b)
    .map((num) => {
      const meta = metaByTrack[num]
      if (!meta) return null
      return { ...meta, track_total: total, track_number: meta.track_number || num }
    })
    .filter(Boolean)
}
