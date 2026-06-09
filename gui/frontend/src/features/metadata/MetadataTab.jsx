import { useCallback, useMemo, useState } from 'react'
import {
  TagPickAudioFile,
  TagPickArtworkFile,
  TagReadFile,
  TagWriteFile,
  TagLocalFileURL,
  OpenFolder,
} from '../../wailsjs/go/main/App'

import ArtworkEditor from '../../components/ArtworkEditor'
import StatusToast from '../../components/StatusToast'
import { resolveMediaURL } from '../splice/useMasterAudio'
import { formatActionError } from '../splice/projectUtils'
import { reportFrontendError } from '../../lib/errorReporting'

const EMPTY = {
  path: '',
  title: '',
  artist: '',
  album: '',
  album_artist: '',
  genre: '',
  year: '',
  track_number: '',
  track_total: '',
  disc_number: '',
  disc_total: '',
}

function artworkSrc(info, coverPreviewURL) {
  if (coverPreviewURL) return coverPreviewURL
  if (info?.artwork_b64 && info?.artwork_mime) {
    return `data:${info.artwork_mime};base64,${info.artwork_b64}`
  }
  return ''
}

function tagsFromInfo(info, path) {
  return {
    path: info.path || path,
    title: info.title || '',
    artist: info.artist || '',
    album: info.album || '',
    album_artist: info.album_artist || '',
    genre: info.genre || '',
    year: info.year || '',
    track_number: info.track_number > 0 ? String(info.track_number) : '',
    track_total: info.track_total > 0 ? String(info.track_total) : '',
    disc_number: info.disc_number > 0 ? String(info.disc_number) : '',
    disc_total: info.disc_total > 0 ? String(info.disc_total) : '',
  }
}

function SaveButtonIcon({ saving, saved }) {
  if (saved) {
    return (
      <svg className="h-4 w-4" viewBox="0 0 16 16" fill="none" aria-hidden>
        <path
          d="M3.5 8.5 6.5 11.5 12.5 4.5"
          stroke="currentColor"
          strokeWidth="1.75"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    )
  }
  if (saving) {
    return (
      <svg className="h-4 w-4 animate-spin" viewBox="0 0 16 16" fill="none" aria-hidden>
        <circle cx="8" cy="8" r="5.5" stroke="currentColor" strokeOpacity="0.25" strokeWidth="1.5" />
        <path d="M8 2.5a5.5 5.5 0 0 1 5.5 5.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    )
  }
  return null
}

export default function MetadataTab() {
  const [tags, setTags] = useState(EMPTY)
  const [fileInfo, setFileInfo] = useState(null)
  const [coverPath, setCoverPath] = useState('')
  const [coverPreviewURL, setCoverPreviewURL] = useState('')
  const [clearArtwork, setClearArtwork] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [toast, setToast] = useState('')
  const [toastVariant, setToastVariant] = useState('success')

  const dismissToast = useCallback(() => setToast(''), [])

  const showToast = useCallback((message, variant = 'success') => {
    setToastVariant(variant)
    setToast(message)
  }, [])

  const artworkPreview = useMemo(
    () => (clearArtwork ? '' : artworkSrc(fileInfo, coverPreviewURL)),
    [fileInfo, coverPreviewURL, clearArtwork],
  )

  const loadFile = async (path) => {
    if (!path) return
    setLoading(true)
    setSaved(false)
    setToast('')
    setCoverPath('')
    setCoverPreviewURL('')
    setClearArtwork(false)
    try {
      const info = await TagReadFile(path)
      setFileInfo(info)
      setTags(tagsFromInfo(info, path))
      if (info.artwork_count > 1) {
        showToast(`Found ${info.artwork_count} embedded covers — save once to keep a single artwork for Apple Music.`, 'info')
      } else {
        showToast(info.summary ? `Loaded ${info.summary}` : 'Loaded file metadata')
      }
    } catch (e) {
      reportFrontendError('MetadataTab.loadFile', e)
      showToast(formatActionError(e, 'Read tags'), 'error')
      setFileInfo(null)
    } finally {
      setLoading(false)
    }
  }

  const pickFile = async () => {
    const path = await TagPickAudioFile()
    if (path) await loadFile(path)
  }

  const pickArtwork = async () => {
    const path = await TagPickArtworkFile()
    if (path) {
      setCoverPath(path)
      setClearArtwork(false)
      setSaved(false)
      const url = await Promise.resolve(TagLocalFileURL(path))
      setCoverPreviewURL(typeof url === 'string' ? resolveMediaURL(url) : '')
    }
  }

  const update = (key, value) => {
    setSaved(false)
    setTags((t) => ({ ...t, [key]: value }))
  }

  const toInt16 = (value) => {
    const n = Number(value)
    return Number.isFinite(n) && n > 0 ? Math.round(n) : 0
  }

  const save = async () => {
    if (!tags.path) {
      showToast('Open an audio file first.', 'error')
      return
    }
    setSaving(true)
    setSaved(false)
    try {
      const updated = await TagWriteFile({
        path: tags.path,
        title: tags.title,
        artist: tags.artist,
        album: tags.album,
        album_artist: tags.album_artist,
        genre: tags.genre,
        year: tags.year,
        track_number: toInt16(tags.track_number),
        track_total: toInt16(tags.track_total),
        disc_number: toInt16(tags.disc_number),
        disc_total: toInt16(tags.disc_total),
        cover_path: coverPath,
        clear_artwork: clearArtwork,
        sort_tags: true,
      })
      setFileInfo(updated)
      setTags(tagsFromInfo(updated, tags.path))
      setCoverPath('')
      setCoverPreviewURL('')
      setClearArtwork(false)
      setSaved(true)
      showToast('Tags saved successfully.')
      setTimeout(() => setSaved(false), 2200)
    } catch (e) {
      reportFrontendError('MetadataTab.save', e)
      showToast(formatActionError(e, 'Save tags'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const saveButtonLabel = saved ? 'Saved' : saving ? 'Saving…' : 'Save tags'

  return (
    <div className="relative mx-auto flex h-full max-w-4xl flex-col gap-4 overflow-y-auto pb-4">
      <StatusToast message={toast} variant={toastVariant} onDismiss={dismissToast} />

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h2 className="text-xl font-semibold">Tag Editor</h2>
        <p className="mt-1 text-sm text-white/50">
          Edit title, album, track numbers, and artwork on local M4A files before syncing to Apple Music.
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={pickFile}
            disabled={loading || saving}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium transition-all duration-200 ease-apple disabled:opacity-50"
          >
            {loading ? 'Loading…' : 'Open audio file'}
          </button>
          {tags.path && (
            <button
              type="button"
              onClick={() => {
                const i = Math.max(tags.path.lastIndexOf('\\'), tags.path.lastIndexOf('/'))
                if (i > 0) OpenFolder(tags.path.slice(0, i))
              }}
              className="rounded-lg border border-white/15 px-4 py-2 text-sm transition-colors duration-200 ease-apple hover:bg-white/5"
            >
              Show in folder
            </button>
          )}
        </div>
        {tags.path && (
          <p className="mt-2 truncate text-xs text-white/45" title={tags.path}>
            {tags.path}
          </p>
        )}
      </section>

      {!tags.path ? (
        <div className="flex flex-1 items-center justify-center rounded-xl border border-dashed border-white/10 p-8 text-center text-sm text-white/45">
          Open an .m4a file to view and edit its tags.
        </div>
      ) : (
        <div
          className={`grid gap-4 transition-opacity duration-300 ease-apple lg:grid-cols-[12rem_1fr] ${
            saving ? 'pointer-events-none opacity-70' : 'opacity-100'
          }`}
        >
          <ArtworkEditor
            previewSrc={artworkPreview}
            onReplace={pickArtwork}
            onRemove={() => {
              setClearArtwork(true)
              setCoverPath('')
              setCoverPreviewURL('')
              setSaved(false)
            }}
            disabled={saving}
          />

          <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
            <h3 className="text-sm font-medium">Metadata</h3>
            <div className="mt-3 grid gap-3 sm:grid-cols-2">
              {[
                ['title', 'Title'],
                ['artist', 'Artist'],
                ['album', 'Album'],
                ['album_artist', 'Album artist'],
                ['genre', 'Genre'],
                ['year', 'Year'],
              ].map(([key, label]) => (
                <div key={key} className={key === 'title' ? 'sm:col-span-2' : ''}>
                  <label className="text-xs text-white/50">{label}</label>
                  <input
                    value={tags[key]}
                    onChange={(e) => update(key, e.target.value)}
                    disabled={saving}
                    className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-sm transition-colors duration-200 ease-apple focus:border-accent/40 focus:outline-none disabled:opacity-60"
                  />
                </div>
              ))}
              {[
                ['track_number', 'Track #'],
                ['track_total', 'Track total'],
                ['disc_number', 'Disc #'],
                ['disc_total', 'Disc total'],
              ].map(([key, label]) => (
                <div key={key}>
                  <label className="text-xs text-white/50">{label}</label>
                  <input
                    type="number"
                    min={1}
                    value={tags[key]}
                    onChange={(e) => update(key, e.target.value)}
                    disabled={saving}
                    className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-sm transition-colors duration-200 ease-apple focus:border-accent/40 focus:outline-none disabled:opacity-60"
                  />
                </div>
              ))}
            </div>
            <button
              type="button"
              onClick={save}
              disabled={saving}
              aria-busy={saving}
              className={`mt-4 flex w-full items-center justify-center gap-2 rounded-lg py-2.5 text-sm font-semibold transition-all duration-300 ease-apple disabled:cursor-wait ${
                saved
                  ? 'bg-green-600 text-white shadow-sm shadow-green-900/30'
                  : 'bg-accent text-white hover:bg-accent/90 disabled:opacity-80'
              }`}
            >
              <SaveButtonIcon saving={saving} saved={saved} />
              {saveButtonLabel}
            </button>
          </section>
        </div>
      )}
    </div>
  )
}
