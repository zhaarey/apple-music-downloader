import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  TagPickAudioFile,
  TagPickArtworkFile,
  TagReadFile,
  TagWriteFile,
  TagLocalFileURL,
  ValidateIPhoneSync,
  RevealInFolder,
  GetRecentFiles,
} from '../../wailsjs/go/main/App'

import ArtworkEditor from '../../components/ArtworkEditor'
import StatusToast from '../../components/StatusToast'
import SyncValidationPanel, { SyncTroubleshootingPanel } from './SyncValidationPanel'
import AlbumBulkTagEditor from './AlbumBulkTagEditor'
import { useTagEditorDrop } from '../../hooks/useTagEditorDrop'
import { resolveMediaURL } from '../../lib/resolveMediaURL'
import { formatActionError } from '../../lib/formatActionError'
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

function basename(path) {
  const parts = String(path || '').split(/[/\\]/)
  return parts[parts.length - 1] || path
}

function albumFolder(path) {
  const parts = String(path || '').split(/[/\\]/).filter(Boolean)
  if (parts.length < 2) return ''
  parts.pop()
  return parts.join('\\')
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

export default function MetadataTab({ platform = 'windows' }) {
  const [editorMode, setEditorMode] = useState('single')
  const [bulkAlbumFolder, setBulkAlbumFolder] = useState('')
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
  const [syncResult, setSyncResult] = useState(null)
  const [folderSyncResult, setFolderSyncResult] = useState(null)
  const [validating, setValidating] = useState(false)
  const [recentFiles, setRecentFiles] = useState([])
  const [albumLoadRequest, setAlbumLoadRequest] = useState(null)
  const [albumBusy, setAlbumBusy] = useState(false)

  useEffect(() => {
    GetRecentFiles()
      .then((paths) => setRecentFiles(Array.isArray(paths) ? paths : []))
      .catch(() => setRecentFiles([]))
  }, [])

  const dismissToast = useCallback(() => setToast(''), [])

  const showToast = useCallback((message, variant = 'success') => {
    setToastVariant(variant)
    setToast(message)
  }, [])

  const artworkPreview = useMemo(
    () => (clearArtwork ? '' : artworkSrc(fileInfo, coverPreviewURL)),
    [fileInfo, coverPreviewURL, clearArtwork],
  )

  const clearFile = useCallback(() => {
    if (saving) return
    setTags(EMPTY)
    setFileInfo(null)
    setCoverPath('')
    setCoverPreviewURL('')
    setClearArtwork(false)
    setSaved(false)
    setSyncResult(null)
    setValidating(false)
    setToast('')
  }, [saving])

  const loadFile = async (path) => {
    if (!path) return
    setEditorMode('single')
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
      setSyncResult(null)
      setRecentFiles((prev) => [path, ...prev.filter((p) => p !== path)].slice(0, 15))
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

  const handleDropSingleFile = useCallback(
    async (path, message) => {
      await loadFile(path)
      if (message) showToast(message, 'info')
    },
    [showToast],
  )

  const handleDropAlbumFolder = useCallback(
    async (folder, message) => {
      setEditorMode('album')
      setAlbumLoadRequest({ folder, key: Date.now() })
      if (message) showToast(message, 'info')
    },
    [showToast],
  )

  const handleDropError = useCallback(
    (message) => {
      showToast(message, 'error')
    },
    [showToast],
  )

  const dropDisabled = loading || saving || albumBusy

  const { dragOver, dropZoneProps } = useTagEditorDrop({
    onSingleFile: handleDropSingleFile,
    onAlbumFolder: handleDropAlbumFolder,
    onError: handleDropError,
    disabled: dropDisabled,
  })

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

  const validateSync = async () => {
    if (!tags.path) {
      showToast('Open an audio file first.', 'error')
      return
    }
    setValidating(true)
    try {
      const res = await ValidateIPhoneSync(tags.path)
      setSyncResult(res)
      showToast(res.summary, res.ready ? 'success' : 'info')
    } catch (e) {
      reportFrontendError('MetadataTab.validateSync', e)
      showToast(formatActionError(e, 'Validate sync'), 'error')
    } finally {
      setValidating(false)
    }
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
      const validation = await ValidateIPhoneSync(tags.path)
      setSyncResult(validation)
      showToast(validation.ready ? 'Tags saved — ready for iPhone sync.' : 'Tags saved — review sync checklist below.')
      setTimeout(() => setSaved(false), 2200)
    } catch (e) {
      reportFrontendError('MetadataTab.save', e)
      showToast(formatActionError(e, 'Save tags'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const saveButtonLabel = saved ? 'Saved' : saving ? 'Saving…' : 'Save tags'

  const hintAlbumFolder =
    editorMode === 'single' && tags.path ? albumFolder(tags.path) : editorMode === 'album' ? bulkAlbumFolder : ''

  return (
    <div
      className={`relative mx-auto flex h-full max-w-content flex-col gap-4 overflow-y-auto pb-4 transition-colors duration-200 ${
        dragOver ? 'rounded-xl ring-2 ring-accent/40 ring-inset' : ''
      }`}
      {...dropZoneProps}
    >
      <StatusToast message={toast} variant={toastVariant} onDismiss={dismissToast} duration={4200} />

      {dragOver && (
        <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center rounded-xl border-2 border-dashed border-accent/50 bg-accent/10 backdrop-blur-[1px]">
          <div className="rounded-xl border border-accent/30 bg-surface-raised/95 px-6 py-4 text-center shadow-lg">
            <p className="text-sm font-medium text-white">Drop to open in Tag Editor</p>
            <p className="mt-1 text-xs text-white/55">One .m4a → single file · Folder or multiple tracks → album bulk</p>
          </div>
        </div>
      )}

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h2 className="text-xl font-semibold">Tag Editor</h2>
        <p className="mt-1 text-sm text-white/50">
          Edit title, album, track numbers, and artwork on local M4A files before syncing to Apple Music. Drag and drop a
          file or folder anywhere here.
        </p>
        <div className="mt-4 inline-flex rounded-lg border border-white/10 bg-black/20 p-0.5">
          <button
            type="button"
            onClick={() => setEditorMode('single')}
            className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
              editorMode === 'single' ? 'bg-accent text-white' : 'text-white/60 hover:text-white'
            }`}
          >
            Single file
          </button>
          <button
            type="button"
            onClick={() => setEditorMode('album')}
            className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
              editorMode === 'album' ? 'bg-accent text-white' : 'text-white/60 hover:text-white'
            }`}
          >
            Album bulk
          </button>
        </div>
        {editorMode === 'single' && (
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
              onClick={validateSync}
              disabled={loading || saving || validating}
              className="rounded-lg border border-white/15 px-4 py-2 text-sm transition-colors duration-200 ease-apple hover:bg-white/5 disabled:opacity-50"
            >
              {validating ? 'Checking…' : 'Validate iPhone sync'}
            </button>
          )}
          {tags.path && (
            <button
              type="button"
              onClick={() => RevealInFolder(tags.path)}
              className="rounded-lg border border-white/15 px-4 py-2 text-sm transition-colors duration-200 ease-apple hover:bg-white/5"
            >
              Show in folder
            </button>
          )}
          {tags.path && (
            <button
              type="button"
              onClick={clearFile}
              disabled={loading || saving}
              className="rounded-lg border border-white/15 px-4 py-2 text-sm text-white/70 transition-colors duration-200 ease-apple hover:border-white/25 hover:bg-white/5 hover:text-white disabled:opacity-50"
            >
              Close file
            </button>
          )}
        </div>
        )}
        {editorMode === 'single' && tags.path && (
          <p className="mt-2 truncate text-xs text-white/45" title={tags.path}>
            {tags.path}
          </p>
        )}
        {editorMode === 'single' && tags.path && albumFolder(tags.path) && (
          <p className="mt-1 truncate text-xs text-white/35" title={albumFolder(tags.path)}>
            Album folder: {albumFolder(tags.path)}
          </p>
        )}
        {editorMode === 'single' && syncResult && tags.path && (
          <div className="mt-4">
            <SyncValidationPanel result={syncResult} />
          </div>
        )}
      </section>

      {!tags.path && editorMode === 'single' ? (
        <div className="flex flex-1 flex-col gap-4">
          {recentFiles.length > 0 && (
            <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
              <h3 className="text-sm font-medium">Recent files</h3>
              <ul className="mt-3 space-y-1">
                {recentFiles.map((path) => (
                  <li key={path}>
                    <button
                      type="button"
                      onClick={() => loadFile(path)}
                      disabled={loading}
                      className="flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-white/5 disabled:opacity-50"
                      title={path}
                    >
                      <span className="truncate">{basename(path)}</span>
                      <span className="shrink-0 text-xs text-white/40">Open</span>
                    </button>
                  </li>
                ))}
              </ul>
            </section>
          )}
          <div className="flex flex-1 items-center justify-center rounded-xl border border-dashed border-white/10 p-8 text-center text-sm text-white/45">
            Open an .m4a file or drag one here. Drop a folder (or several tracks from the same album) for bulk edit.
          </div>
        </div>
      ) : editorMode === 'album' ? (
        <AlbumBulkTagEditor
          onStatus={(message, variant) => showToast(message, variant)}
          onFolderChange={setBulkAlbumFolder}
          loadRequest={albumLoadRequest}
          onBusyChange={setAlbumBusy}
        />
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

      <SyncTroubleshootingPanel
        platform={platform}
        hintAlbumFolder={hintAlbumFolder}
        onStatus={(message, variant) => showToast(message, variant)}
        folderResult={folderSyncResult}
        onFolderResult={setFolderSyncResult}
      />
    </div>
  )
}
