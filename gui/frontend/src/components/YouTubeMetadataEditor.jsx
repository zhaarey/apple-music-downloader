import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  TagLocalFileURL,
  TagPickArtworkFile,
  TagAnalyzeArtworkSource,
  TagApplyOptimizedArtwork,
} from '../wailsjs/go/main/App'
import { resolveMediaURL } from '../lib/resolveMediaURL'
import ArtworkAppleOptions from './ArtworkAppleOptions'
import { youtubeDeliveryIncludesAudio, youtubeDeliveryIncludesVideo } from '../lib/youtubeDelivery'
import {
  loadArtworkSourceAnalysis,
  optimizedPreviewFromAnalysis,
} from '../lib/artworkApple'
import { formatActionError } from '../lib/formatActionError'

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

function resolveArtworkSource(shared, meta, preview) {
  if (shared.art_source === 'none') return { coverPath: '', artURL: '' }
  if (shared.art_source === 'custom') return { coverPath: shared.cover_path || '', artURL: '' }
  return { coverPath: '', artURL: youtubeThumbURL(meta, null, preview) }
}

export default function YouTubeMetadataEditor({
  preview,
  selected,
  metaByTrack,
  onChange,
  onSharedChange,
  disabled,
  deliveryMode = 'audio',
}) {
  const includesAudio = youtubeDeliveryIncludesAudio(deliveryMode)
  const includesVideo = youtubeDeliveryIncludesVideo(deliveryMode)
  const formatHint = includesAudio && includesVideo
    ? 'AAC 256 kbps and H.264 MP4'
    : includesVideo
      ? 'H.264 MP4'
      : 'AAC 256 kbps'
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
        optimize_artwork: Boolean(firstMeta.optimize_artwork),
      }
    : {
        album: '',
        album_artist: '',
        genre: '',
        year: '',
        art_source: 'youtube',
        cover_path: '',
        optimize_artwork: false,
      }

  const [customPreviewURL, setCustomPreviewURL] = useState('')
  const [artworkAnalysis, setArtworkAnalysis] = useState(null)
  const [optimizedPreviewURL, setOptimizedPreviewURL] = useState('')
  const [applyingOptimization, setApplyingOptimization] = useState(false)
  const [artworkError, setArtworkError] = useState('')

  const youtubePreviewURL = useMemo(() => {
    if (!first) return ''
    return youtubeThumbURL(firstMeta, first, preview)
  }, [first, firstMeta, preview])

  const artworkSource = useMemo(
    () => resolveArtworkSource(shared, firstMeta, preview),
    [shared.art_source, shared.cover_path, firstMeta, preview],
  )

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

  const refreshArtworkAnalysis = useCallback(async () => {
    if (shared.art_source === 'none') {
      setArtworkAnalysis(null)
      setOptimizedPreviewURL('')
      return
    }
    const { coverPath, artURL } = artworkSource
    if (!coverPath && !artURL) {
      setArtworkAnalysis(null)
      setOptimizedPreviewURL('')
      return
    }
    const analysis = await loadArtworkSourceAnalysis({ coverPath, artURL }, TagAnalyzeArtworkSource)
    setArtworkAnalysis(analysis)
    setOptimizedPreviewURL(
      shared.optimize_artwork && analysis ? optimizedPreviewFromAnalysis(analysis) : '',
    )
  }, [shared.art_source, shared.optimize_artwork, artworkSource])

  useEffect(() => {
    void refreshArtworkAnalysis()
  }, [refreshArtworkAnalysis])

  const applyShared = (patch) => {
    onSharedChange(patch)
  }

  const setArtSource = (artSource) => {
    if (artSource === 'custom') {
      applyShared({ art_source: artSource })
      return
    }
    if (artSource === 'youtube') {
      applyShared({ art_source: artSource, cover_path: '', optimize_artwork: false })
      return
    }
    applyShared({ art_source: artSource, cover_path: '', optimize_artwork: false })
  }

  const pickCustomArtwork = async () => {
    const path = await TagPickArtworkFile()
    if (!path) return
    applyShared({ art_source: 'custom', cover_path: path, optimize_artwork: false })
  }

  const applyOptimization = async () => {
    const { coverPath, artURL } = artworkSource
    if (!coverPath && !artURL) return
    setApplyingOptimization(true)
    setArtworkError('')
    try {
      const result = await TagApplyOptimizedArtwork(coverPath, artURL)
      if (!result?.path) {
        setArtworkError('Optimization did not produce a cover file.')
        return
      }
      applyShared({
        art_source: 'custom',
        cover_path: result.path,
        optimize_artwork: true,
      })
      setArtworkAnalysis(result)
      const url = await Promise.resolve(TagLocalFileURL(result.path))
      setCustomPreviewURL(typeof url === 'string' ? resolveMediaURL(url) : '')
      setOptimizedPreviewURL('')
    } catch (e) {
      setArtworkError(formatActionError(e, 'Optimize artwork'))
    } finally {
      setApplyingOptimization(false)
    }
  }

  return (
    <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-sm font-medium">Apple Music metadata</h4>
          <p className="text-xs text-white/50">
            Edit before download · saved as {formatHint} in Album folders
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
        <p className="text-[11px] uppercase tracking-wide text-white/45">Artwork source</p>
        <div className="mt-2 flex flex-wrap gap-2">
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
        {shared.art_source === 'youtube' && selectedTracks.length > 1 && (
          <p className="mt-2 text-xs text-white/45">
            Playlist mode uses each video&apos;s thumbnail unless you apply a shared optimized cover below.
          </p>
        )}
        {shared.art_source === 'youtube' && !youtubePreviewURL && (
          <p className="mt-2 text-xs text-amber-200/90">No YouTube thumbnail available for this item.</p>
        )}
      </div>

      {shared.art_source !== 'none' && (
        <ArtworkAppleOptions
          className="mt-4 border-white/10"
          previewSrc={artworkPreviewSrc}
          optimizedPreviewSrc={optimizedPreviewURL}
          analysis={artworkAnalysis}
          optimizeArtwork={shared.optimize_artwork}
          onOptimizeArtworkChange={(checked) => applyShared({ optimize_artwork: checked })}
          showMp4boxReembed={false}
          onReplace={() => void pickCustomArtwork()}
          onRemove={
            shared.art_source === 'custom'
              ? () => applyShared({ art_source: 'youtube', cover_path: '', optimize_artwork: false })
              : undefined
          }
          onApplyOptimization={artworkPreviewSrc ? () => void applyOptimization() : undefined}
          applyingOptimization={applyingOptimization}
          applyOptimizationLabel="Optimize for Apple Music now"
          disabled={disabled}
        />
      )}

      {artworkError && (
        <p className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-200">
          {artworkError}
        </p>
      )}

      {shared.art_source !== 'none' && (
        <p className="mt-3 text-xs text-white/45">
          Accent colors come from embedded artwork on your device — not Apple&apos;s streaming CDN. Motion album
          art and synced lyrics still require catalog matches.
        </p>
      )}

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
      optimize_artwork: false,
    }
  })
  return map
}

export function metaPayload(metaByTrack, selected) {
  const total = selected.size
  const sharedOptimize = [...selected].some((num) => metaByTrack[num]?.optimize_artwork)
  return [...selected]
    .sort((a, b) => a - b)
    .map((num) => {
      const meta = metaByTrack[num]
      if (!meta) return null
      return {
        ...meta,
        track_total: total,
        track_number: meta.track_number || num,
        optimize_artwork: meta.optimize_artwork ?? sharedOptimize,
      }
    })
    .filter(Boolean)
}
