import { useEffect, useMemo, useState } from 'react'
import { TagLocalFileURL, TagPickArtworkFile } from '../wailsjs/go/main/App'
import { resolveMediaURL } from '../lib/resolveMediaURL'
import ArtworkPreview from './ArtworkPreview'

const FIELD_CLASS =
  'mt-1 w-full rounded-lg border border-white/10 bg-surface px-2.5 py-1.5 text-sm focus:border-accent focus:outline-none'

const ART_SOURCES = [
  { id: 'youtube', label: 'YouTube thumbnail' },
  { id: 'custom', label: 'Custom image' },
  { id: 'none', label: 'No artwork' },
]

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

function youtubeThumbURL(meta, track, preview) {
  return meta?.art_url || track?.art_url || preview?.art_url || ''
}

export default function YouTubeMetadataEditor({
  preview,
  selected,
  metaByTrack,
  onChange,
  onSharedChange,
  disabled,
  saveVideo = false,
}) {
  const selectedTracks = (preview?.tracks || []).filter((t) => selected.has(t.num))
  const first = selectedTracks[0] || preview?.tracks?.[0]
  const firstMeta = first ? metaByTrack[first.num] || {} : {}
  const shared = first
    ? {
        album: firstMeta.album || '',
        album_artist: firstMeta.album_artist || '',
        genre: firstMeta.genre || '',
        year: firstMeta.year || '',
        art_source: firstMeta.art_source || 'youtube',
        cover_path: firstMeta.cover_path || '',
      }
    : { album: '', album_artist: '', genre: '', year: '', art_source: 'youtube', cover_path: '' }

  const [customPreviewURL, setCustomPreviewURL] = useState('')

  const youtubePreviewURL = useMemo(() => {
    if (!first) return ''
    return youtubeThumbURL(firstMeta, first, preview)
  }, [first, firstMeta, preview])

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      if (shared.art_source !== 'custom' || !shared.cover_path) {
        setCustomPreviewURL('')
        return
      }
      try {
        const url = await Promise.resolve(TagLocalFileURL(shared.cover_path))
        if (!cancelled) {
          setCustomPreviewURL(typeof url === 'string' ? resolveMediaURL(url) : '')
        }
      } catch {
        if (!cancelled) setCustomPreviewURL('')
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [shared.art_source, shared.cover_path])

  const artworkPreviewSrc =
    shared.art_source === 'custom'
      ? customPreviewURL
      : shared.art_source === 'youtube'
        ? youtubePreviewURL
        : ''

  const applyShared = (patch) => {
    onSharedChange(patch)
  }

  const setArtSource = (artSource) => {
    if (artSource === 'custom') {
      applyShared({ art_source: artSource })
      return
    }
    if (artSource === 'youtube') {
      applyShared({ art_source: artSource, cover_path: '' })
      return
    }
    applyShared({ art_source: artSource, cover_path: '' })
  }

  const pickCustomArtwork = async () => {
    const path = await TagPickArtworkFile()
    if (!path) return
    applyShared({ art_source: 'custom', cover_path: path })
  }

  return (
    <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-sm font-medium">Apple Music metadata</h4>
          <p className="text-xs text-white/50">
            Edit before download · saved as AAC 256 kbps
            {saveVideo ? ' and optional H.264 MP4' : ''} in Album folders
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

      <div className="mt-4 border-t border-white/10 pt-4">
        <p className="text-[11px] uppercase tracking-wide text-white/45">Artwork</p>
        <p className="mt-1 text-xs text-white/50">
          Embedded on every .m4a
          {saveVideo ? ' and [video].mp4' : ''} for iPhone Music library sync
          {selectedTracks.length > 1 && shared.art_source === 'youtube'
            ? ' · playlists use each video’s YouTube thumbnail'
            : ''}
        </p>
        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-start">
          <ArtworkPreview
            src={artworkPreviewSrc}
            className="flex aspect-square w-full max-w-[8.5rem] shrink-0 items-center justify-center rounded-lg bg-black/30"
          />
          <div className="min-w-0 flex-1 space-y-3">
            <div className="flex flex-wrap gap-2">
              {ART_SOURCES.map((opt) => (
                <button
                  key={opt.id}
                  type="button"
                  disabled={disabled}
                  onClick={() => setArtSource(opt.id)}
                  className={`rounded-lg border px-3 py-1.5 text-xs transition-colors ${
                    shared.art_source === opt.id
                      ? 'border-accent bg-accent/15 text-white'
                      : 'border-white/10 text-white/70 hover:border-white/20 hover:bg-white/[0.03]'
                  } disabled:opacity-50`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
            {shared.art_source === 'custom' && (
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  disabled={disabled}
                  onClick={() => void pickCustomArtwork()}
                  className="rounded-lg border border-white/15 px-3 py-2 text-xs transition-colors hover:bg-white/5 disabled:opacity-50"
                >
                  {shared.cover_path ? 'Replace image' : 'Choose image'}
                </button>
                {shared.cover_path && (
                  <span className="truncate text-xs text-white/40" title={shared.cover_path}>
                    {shared.cover_path.split(/[/\\]/).pop()}
                  </span>
                )}
              </div>
            )}
            {shared.art_source === 'youtube' && !youtubePreviewURL && (
              <p className="text-xs text-amber-200/90">No YouTube thumbnail available for this item.</p>
            )}
          </div>
        </div>
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
      art_source: 'youtube',
      cover_path: '',
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
