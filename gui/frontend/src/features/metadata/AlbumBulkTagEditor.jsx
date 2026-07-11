import { useEffect, useMemo, useRef, useState } from 'react'
import {
  TagPickArtworkFile,
  TagReadAlbumFolder,
  TagWriteAlbumBatch,
  TagLocalFileURL,
  TagAnalyzeArtwork,
  TagAnalyzeEmbeddedArtwork,
  TagFindAlbumCover,
  PickFolder,
  RevealInFolder,
} from '../../wailsjs/go/main/App'
import ArtworkAppleOptions from '../../components/ArtworkAppleOptions'
import {
  loadArtworkAnalysis,
  loadEmbeddedArtworkAnalysis,
  optimizedPreviewFromAnalysis,
} from '../../lib/artworkApple'
import { useConfirm } from '../../hooks/useConfirm'
import { bulkSaveArtworkConfirmDetails } from '../../lib/prepareAlbumConfirm'
import { resolveMediaURL } from '../../lib/resolveMediaURL'
import { formatActionError } from '../../lib/formatActionError'
import { reportFrontendError } from '../../lib/errorReporting'

function basename(path) {
  const parts = String(path || '').split(/[/\\]/)
  return parts[parts.length - 1] || path
}

function trackFromInfo(info, index) {
  return {
    path: info.path,
    title: info.title || '',
    artist: info.artist || '',
    album: info.album || '',
    album_artist: info.album_artist || '',
    genre: info.genre || '',
    year: info.year || '',
    track_number: info.track_number > 0 ? info.track_number : index + 1,
    disc_number: info.disc_number > 0 ? info.disc_number : 1,
    has_artwork: info.has_artwork,
  }
}

function inferAlbumDefaults(tracks) {
  if (!tracks.length) {
    return { album: '', album_artist: '', genre: '', year: '', track_total: null, disc_total: 1 }
  }
  const first = tracks[0]
  const same = (key) => tracks.every((t) => (t[key] || '') === (first[key] || ''))
  return {
    album: same('album') ? first.album || '' : '',
    album_artist: same('album_artist') ? first.album_artist || '' : '',
    genre: same('genre') ? first.genre || '' : '',
    year: same('year') ? first.year || '' : '',
    track_total: tracks.length,
    disc_total: 1,
  }
}

function artworkFromTracks(infos) {
  const withArt = (infos || []).find((i) => i?.artwork_b64 && i?.artwork_mime)
  if (!withArt) return ''
  return `data:${withArt.artwork_mime};base64,${withArt.artwork_b64}`
}

export default function AlbumBulkTagEditor({ onStatus, onFolderChange, loadRequest, onBusyChange }) {
  const { requestConfirm, ConfirmDialogSlot } = useConfirm()
  const [folder, setFolder] = useState('')
  const [album, setAlbum] = useState({
    album: '',
    album_artist: '',
    genre: '',
    year: '',
    track_total: null,
    disc_total: 1,
  })
  const [tracks, setTracks] = useState([])
  const [fileInfos, setFileInfos] = useState([])
  const [selectedTrackIdx, setSelectedTrackIdx] = useState(-1)
  const [coverPath, setCoverPath] = useState('')
  const [coverPreviewURL, setCoverPreviewURL] = useState('')
  const [clearArtwork, setClearArtwork] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savedFlash, setSavedFlash] = useState(false)
  const [albumDefaultsApplied, setAlbumDefaultsApplied] = useState(false)
  const [optimizeArtwork, setOptimizeArtwork] = useState(false)
  const [mp4boxReembed, setMp4boxReembed] = useState(false)
  const [artworkAnalysis, setArtworkAnalysis] = useState(null)
  const [optimizedPreviewURL, setOptimizedPreviewURL] = useState('')
  const [folderCoverPath, setFolderCoverPath] = useState('')
  const albumDefaultsTimerRef = useRef(null)

  useEffect(() => {
    return () => {
      if (albumDefaultsTimerRef.current) clearTimeout(albumDefaultsTimerRef.current)
    }
  }, [])

  useEffect(() => {
    onBusyChange?.(loading || saving)
  }, [loading, saving, onBusyChange])

  const selectedTrack = useMemo(() => {
    if (selectedTrackIdx < 0 || selectedTrackIdx >= tracks.length) return null
    return tracks[selectedTrackIdx]
  }, [tracks, selectedTrackIdx])

  const artworkPreview = useMemo(() => {
    if (clearArtwork) return ''
    if (coverPreviewURL) return coverPreviewURL
    return artworkFromTracks(fileInfos)
  }, [clearArtwork, coverPreviewURL, fileInfos])

  const tracksMissingArt = useMemo(
    () => tracks.filter((t) => !t.has_artwork).length,
    [tracks],
  )

  const hasAnyEmbeddedArt = useMemo(
    () => tracks.some((t) => t.has_artwork) && !clearArtwork,
    [tracks, clearArtwork],
  )

  const updateAlbum = (key, value) => {
    setAlbum((a) => ({ ...a, [key]: value }))
    setSavedFlash(false)
  }

  const updateTrack = (idx, key, value) => {
    setTracks((list) => {
      const next = [...list]
      next[idx] = { ...next[idx], [key]: value }
      return next
    })
    setSavedFlash(false)
  }

  const updateSelectedTrack = (key, value) => {
    if (selectedTrackIdx < 0) return
    updateTrack(selectedTrackIdx, key, value)
  }

  const loadFolder = async (path) => {
    if (!path) return
    setLoading(true)
    setCoverPath('')
    setCoverPreviewURL('')
    setOptimizedPreviewURL('')
    setArtworkAnalysis(null)
    setFolderCoverPath('')
    setClearArtwork(false)
    setSavedFlash(false)
    try {
      const folderResult = await TagReadAlbumFolder(path)
      const infos = Array.isArray(folderResult?.tracks)
        ? folderResult.tracks
        : Array.isArray(folderResult)
          ? folderResult
          : []
      const skipped = Array.isArray(folderResult?.skipped) ? folderResult.skipped : []
      if (infos.length === 0) {
        onStatus?.('No .m4a, .m4b, or .mp4 audio files found in that folder (including subfolders).', 'error')
        return
      }
      const nextTracks = infos.map((info, i) => trackFromInfo(info, i))
      const defaults = inferAlbumDefaults(nextTracks)
      setFolder(path)
      onFolderChange?.(path)
      setFileInfos(infos)
      setTracks(nextTracks)
      setAlbum(defaults)
      setSelectedTrackIdx(0)

      let sidecar = ''
      try {
        sidecar = await TagFindAlbumCover(path)
        setFolderCoverPath(sidecar)
        const analysis = await loadArtworkAnalysis(sidecar, TagAnalyzeArtwork)
        setArtworkAnalysis(analysis)
        setOptimizedPreviewURL(optimizeArtwork ? optimizedPreviewFromAnalysis(analysis) : '')
      } catch {
        setFolderCoverPath('')
        const first = infos.find((i) => i.has_artwork)?.path || infos[0]?.path
        if (first) {
          const analysis = await loadEmbeddedArtworkAnalysis(first, TagAnalyzeEmbeddedArtwork)
          setArtworkAnalysis(analysis)
          setOptimizedPreviewURL(optimizeArtwork ? optimizedPreviewFromAnalysis(analysis) : '')
        }
      }

      const missingArt = nextTracks.filter((t) => !t.has_artwork).length
      if (skipped.length > 0) {
        const preview = skipped.slice(0, 2).join(' · ')
        const more = skipped.length > 2 ? ` · +${skipped.length - 2} more` : ''
        onStatus?.(
          `Loaded ${infos.length} track(s). ${skipped.length} had unreadable tags (using filename): ${preview}${more}`,
          'info',
        )
      } else if (missingArt > 0) {
        onStatus?.(
          `${missingArt} of ${infos.length} track(s) have no embedded cover — choose an image, then Save all to embed it.`,
          'info',
        )
      } else {
        onStatus?.(`Loaded ${infos.length} track(s) from album folder.`)
      }
    } catch (e) {
      reportFrontendError('AlbumBulkTagEditor.loadFolder', e)
      onStatus?.(formatActionError(e, 'Load album folder'), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (loadRequest?.folder) {
      loadFolder(loadRequest.folder)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- loadRequest.key triggers external drop/parent requests
  }, [loadRequest?.key])

  const pickFolder = async () => {
    const path = await PickFolder()
    if (path) await loadFolder(path)
  }

  const refreshArtworkAnalysis = async (path) => {
    if (!path) {
      setArtworkAnalysis(null)
      setOptimizedPreviewURL('')
      return
    }
    const analysis = await loadArtworkAnalysis(path, TagAnalyzeArtwork)
    setArtworkAnalysis(analysis)
    setOptimizedPreviewURL(optimizeArtwork ? optimizedPreviewFromAnalysis(analysis) : '')
  }

  const pickArtwork = async () => {
    const path = await TagPickArtworkFile()
    if (!path) return
    setCoverPath(path)
    setClearArtwork(false)
    setSavedFlash(false)
    const url = await Promise.resolve(TagLocalFileURL(path))
    setCoverPreviewURL(typeof url === 'string' ? resolveMediaURL(url) : '')
    await refreshArtworkAnalysis(path)
    onStatus?.('Artwork selected — click Save all to embed it on every track.', 'success')
  }

  const useFolderCover = async () => {
    if (!folder) return
    try {
      const path = folderCoverPath || (await TagFindAlbumCover(folder))
      setFolderCoverPath(path)
      setCoverPath(path)
      setClearArtwork(false)
      setSavedFlash(false)
      const url = await Promise.resolve(TagLocalFileURL(path))
      setCoverPreviewURL(typeof url === 'string' ? resolveMediaURL(url) : '')
      await refreshArtworkAnalysis(path)
      onStatus?.(`Using ${basename(path)} — click Save all to embed it on every track.`, 'success')
    } catch (e) {
      onStatus?.(formatActionError(e, 'Find folder cover'), 'error')
    }
  }

  const clearPendingArtwork = () => {
    setCoverPath('')
    setCoverPreviewURL('')
    setOptimizedPreviewURL('')
    setClearArtwork(false)
    setSavedFlash(false)
    setArtworkAnalysis(null)
  }

  useEffect(() => {
    if (!coverPath || clearArtwork) return
    if (optimizeArtwork) {
      void refreshArtworkAnalysis(coverPath)
    } else {
      setOptimizedPreviewURL('')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [optimizeArtwork, coverPath, clearArtwork])

  const applyAlbumDefaultsToTracks = () => {
    if (!tracks.length) {
      onStatus?.('Load an album folder first.', 'error')
      return
    }
    const total = Number(album.track_total) > 0 ? Number(album.track_total) : tracks.length
    setTracks((list) =>
      list.map((t, i) => ({
        ...t,
        album: album.album || t.album || '',
        album_artist: album.album_artist || t.album_artist || '',
        genre: album.genre || t.genre || '',
        year: album.year || t.year || '',
        track_number: t.track_number || i + 1,
        disc_number: t.disc_number || album.disc_total || 1,
      })),
    )
    setAlbum((a) => ({ ...a, track_total: total }))
    setAlbumDefaultsApplied(true)
    onStatus?.(`Applied album defaults to ${tracks.length} track(s).`)
    if (albumDefaultsTimerRef.current) clearTimeout(albumDefaultsTimerRef.current)
    albumDefaultsTimerRef.current = setTimeout(() => setAlbumDefaultsApplied(false), 2500)
  }

  const saveAll = async () => {
    if (!tracks.length) {
      onStatus?.('Load an album folder first.', 'error')
      return
    }
    if (coverPath || clearArtwork) {
      const confirmed = await requestConfirm({
        title: clearArtwork ? 'Remove artwork from all tracks?' : 'Apply artwork to all tracks?',
        message: clearArtwork
          ? 'Embedded artwork will be removed from every track in this album.'
          : 'The selected artwork will be embedded on every track in this album.',
        details: bulkSaveArtworkConfirmDetails(tracks.length, !!coverPath, clearArtwork),
        confirmLabel: clearArtwork ? `Remove art from ${tracks.length} tracks` : `Save ${tracks.length} tracks`,
      })
      if (!confirmed) {
        onStatus?.('Save cancelled.', 'info')
        return
      }
    }
    setSaving(true)
    setSavedFlash(false)
    try {
      const trackTotal = Number(album.track_total) > 0 ? Number(album.track_total) : tracks.length
      const res = await TagWriteAlbumBatch({
        folder,
        album: album.album,
        album_artist: album.album_artist,
        genre: album.genre,
        year: album.year,
        track_total: trackTotal,
        disc_total: Number(album.disc_total) > 0 ? Number(album.disc_total) : 1,
        cover_path: coverPath,
        clear_artwork: clearArtwork,
        sort_tags: true,
        optimize_artwork: optimizeArtwork,
        write_cover_sidecar: true,
        mp4box_reembed: mp4boxReembed,
        tracks: tracks.map((t) => ({
          path: t.path,
          title: t.title,
          artist: t.artist,
          track_number: Number(t.track_number) > 0 ? Number(t.track_number) : 0,
          disc_number: Number(t.disc_number) > 0 ? Number(t.disc_number) : 1,
        })),
      })
      const folderResult = await TagReadAlbumFolder(folder)
      const infos = Array.isArray(folderResult?.tracks) ? folderResult.tracks : []
      setFileInfos(infos)
      setTracks(infos.map((info, i) => trackFromInfo(info, i)))
      setCoverPath('')
      setCoverPreviewURL('')
      setOptimizedPreviewURL('')
      setArtworkAnalysis(null)
      setClearArtwork(false)
      setSavedFlash(true)
      setTimeout(() => setSavedFlash(false), 2200)
      onStatus?.(res.summary, res.errors?.length ? 'error' : 'success')
      if (res.errors?.length) {
        reportFrontendError('AlbumBulkTagEditor.saveAll', new Error(res.errors.join('; ')))
      }
    } catch (e) {
      reportFrontendError('AlbumBulkTagEditor.saveAll', e)
      onStatus?.(formatActionError(e, 'Save album tags'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const closeAlbum = () => {
    if (saving) return
    setFolder('')
    onFolderChange?.('')
    setTracks([])
    setFileInfos([])
    setSelectedTrackIdx(-1)
    setCoverPath('')
    setCoverPreviewURL('')
    setFolderCoverPath('')
    setClearArtwork(false)
  }

  if (!folder) {
    return (
      <section className="rounded-xl border border-dashed border-white/10 bg-surface-raised p-8 text-center">
        <h3 className="text-sm font-medium">Album bulk edit</h3>
        <p className="mt-2 text-sm text-white/50">
          Open a folder of tracks from the same album — edit shared metadata once, then tweak individual titles and track
          numbers. Or drag a folder here.
        </p>
        <button
          type="button"
          onClick={pickFolder}
          disabled={loading}
          className="mt-4 rounded-lg bg-accent px-4 py-2 text-sm font-medium disabled:opacity-50"
        >
          {loading ? 'Loading…' : 'Open album folder'}
        </button>
      </section>
    )
  }

  return (
    <div className="space-y-4">
      {ConfirmDialogSlot}
      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <h3 className="text-sm font-medium">Album bulk edit</h3>
            <p className="mt-1 truncate text-xs text-white/45" title={folder}>
              {folder}
            </p>
            <p className="mt-0.5 text-xs text-white/55">{tracks.length} track(s)</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={pickFolder}
              disabled={loading || saving}
              className="rounded-lg border border-white/15 px-3 py-2 text-xs hover:bg-white/5 disabled:opacity-50"
            >
              Change folder
            </button>
            <button
              type="button"
              onClick={() => RevealInFolder(folder)}
              className="rounded-lg border border-white/15 px-3 py-2 text-xs hover:bg-white/5"
            >
              Show folder
            </button>
            <button
              type="button"
              onClick={closeAlbum}
              disabled={saving}
              className="rounded-lg border border-white/15 px-3 py-2 text-xs text-white/70 hover:bg-white/5 disabled:opacity-50"
            >
              Close
            </button>
          </div>
        </div>
        <p className="mt-3 text-xs text-white/40">
          For sync checks or bulk artwork rewrite, expand <strong className="font-medium text-white/55">Sync repair tools</strong> below.
        </p>
      </section>

      <section
        className={`grid gap-4 lg:grid-cols-3 ${saving ? 'pointer-events-none opacity-70' : ''}`}
      >
        <div className="space-y-4">
          <ArtworkAppleOptions
            previewSrc={artworkPreview}
            optimizedPreviewSrc={optimizedPreviewURL}
            analysis={artworkAnalysis}
            optimizeArtwork={optimizeArtwork}
            onOptimizeArtworkChange={setOptimizeArtwork}
            mp4boxReembed={mp4boxReembed}
            onMp4boxReembedChange={setMp4boxReembed}
            showFolderCover={Boolean(folder)}
            folderCoverAvailable={Boolean(folderCoverPath)}
            folderCoverName={folderCoverPath ? basename(folderCoverPath) : 'cover.jpg'}
            hasEmbeddedArtwork={tracksMissingArt === 0 && hasAnyEmbeddedArt}
            forceMissingArtwork={tracksMissingArt > 0 && !coverPath && !clearArtwork}
            pendingCoverPath={coverPath}
            embedding={saving}
            onUseFolderCover={useFolderCover}
            onEmbedAndSave={coverPath ? saveAll : undefined}
            onClearPending={clearPendingArtwork}
            onReplace={pickArtwork}
            onRemove={() => {
              setClearArtwork(true)
              setCoverPath('')
              setCoverPreviewURL('')
              setOptimizedPreviewURL('')
              setArtworkAnalysis(null)
              setSavedFlash(false)
            }}
            disabled={saving}
          />
          <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
            <h3 className="font-medium">Album metadata (mass apply)</h3>
            <p className="mt-1 text-xs text-white/50">Shared fields applied to every track when you save.</p>
            <div className="mt-3 grid gap-2 text-sm">
              {[
                ['album', 'Album', 'text'],
                ['album_artist', 'Album artist', 'text'],
                ['genre', 'Genre', 'text'],
                ['year', 'Year', 'text'],
                ['track_total', 'Track total', 'number'],
                ['disc_total', 'Disc total', 'number'],
              ].map(([key, label, type]) => (
                <div key={key}>
                  <label className="text-xs text-white/50">{label}</label>
                  <input
                    type={type}
                    min={type === 'number' ? 1 : undefined}
                    value={album[key] ?? ''}
                    onChange={(e) => {
                      if (type === 'number') {
                        const n = Number(e.target.value)
                        updateAlbum(key, Number.isFinite(n) && n > 0 ? n : null)
                        return
                      }
                      updateAlbum(key, e.target.value)
                    }}
                    className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
                  />
                </div>
              ))}
              {coverPath && (
                <p className="truncate text-xs text-white/40" title={coverPath}>
                  New artwork: {basename(coverPath)}
                </p>
              )}
              <button
                type="button"
                onClick={applyAlbumDefaultsToTracks}
                disabled={!tracks.length}
                className={`mt-2 w-full rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200 ${
                  albumDefaultsApplied
                    ? 'bg-emerald-600 text-white ring-2 ring-emerald-400/50'
                    : 'bg-accent text-white hover:bg-accent/90 disabled:cursor-not-allowed disabled:opacity-40'
                }`}
              >
                {albumDefaultsApplied
                  ? `Applied to ${tracks.length} track${tracks.length === 1 ? '' : 's'}`
                  : 'Apply album defaults to all tracks'}
              </button>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-white/10 bg-surface-raised p-4 lg:col-span-2">
          <h3 className="font-medium">Tracks ({tracks.length})</h3>
          <div className="mt-2 max-h-72 overflow-auto">
            <table className="w-full text-left text-sm">
              <thead className="sticky top-0 bg-surface-raised text-xs text-white/45">
                <tr>
                  <th className="py-1 pr-2">#</th>
                  <th className="py-1 pr-2">Art</th>
                  <th className="py-1 pr-2">Title</th>
                  <th className="py-1 pr-2">Artist</th>
                  <th className="py-1">Trk</th>
                </tr>
              </thead>
              <tbody>
                {tracks.map((t, i) => (
                  <tr
                    key={t.path}
                    className={`cursor-pointer border-t border-white/5 ${selectedTrackIdx === i ? 'bg-accent/10' : ''}`}
                    onClick={() => setSelectedTrackIdx(i)}
                  >
                    <td className="py-1.5 pr-2 text-white/50">{i + 1}</td>
                    <td className="py-1.5 pr-2">
                      {coverPath && !clearArtwork ? (
                        <span className="text-[10px] font-medium text-accent" title="Will embed on save">
                          →
                        </span>
                      ) : clearArtwork ? (
                        <span className="text-[10px] text-red-300" title="Will remove on save">
                          ×
                        </span>
                      ) : t.has_artwork ? (
                        <span className="text-[10px] text-green-400" title="Embedded">
                          ✓
                        </span>
                      ) : (
                        <span className="text-[10px] text-yellow-300" title="No embedded cover">
                          !
                        </span>
                      )}
                    </td>
                    <td className="py-1.5 pr-2">
                      <input
                        value={t.title || ''}
                        onChange={(e) => updateTrack(i, 'title', e.target.value)}
                        onClick={(e) => e.stopPropagation()}
                        className="w-full rounded bg-black/20 px-2 py-1"
                      />
                    </td>
                    <td className="py-1.5 pr-2">
                      <input
                        value={t.artist || ''}
                        onChange={(e) => updateTrack(i, 'artist', e.target.value)}
                        onClick={(e) => e.stopPropagation()}
                        className="w-full rounded bg-black/20 px-2 py-1"
                      />
                    </td>
                    <td className="py-1.5">
                      <input
                        type="number"
                        min={1}
                        value={t.track_number ?? ''}
                        onChange={(e) => {
                          const n = Number(e.target.value)
                          updateTrack(i, 'track_number', Number.isFinite(n) && n > 0 ? n : '')
                        }}
                        onClick={(e) => e.stopPropagation()}
                        className="w-16 rounded bg-black/20 px-2 py-1"
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">Selected track</h3>
        {!selectedTrack ? (
          <p className="mt-2 text-sm text-white/55">Select a track row to edit per-track details.</p>
        ) : (
          <div className="mt-3 grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-3">
            {[
              ['title', 'Title', 'text'],
              ['artist', 'Artist', 'text'],
              ['track_number', 'Track number', 'number'],
              ['disc_number', 'Disc number', 'number'],
            ].map(([key, label, type]) => (
              <div key={key} className={key === 'title' ? 'sm:col-span-2 lg:col-span-3' : ''}>
                <label className="text-xs text-white/50">{label}</label>
                <input
                  type={type}
                  min={type === 'number' ? 1 : undefined}
                  value={selectedTrack[key] ?? ''}
                  onChange={(e) => {
                    if (type === 'number') {
                      const n = Number(e.target.value)
                      updateSelectedTrack(key, Number.isFinite(n) && n > 0 ? n : '')
                      return
                    }
                    updateSelectedTrack(key, e.target.value)
                  }}
                  className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
                />
              </div>
            ))}
            <div className="sm:col-span-2 lg:col-span-3">
              <p className="truncate text-xs text-white/40" title={selectedTrack.path}>
                {selectedTrack.path}
              </p>
            </div>
          </div>
        )}
      </section>

      <button
        type="button"
        onClick={saveAll}
        disabled={saving || !tracks.length}
        className={`flex w-full items-center justify-center gap-2 rounded-lg py-3 text-sm font-semibold transition-all duration-300 ease-apple disabled:cursor-wait disabled:opacity-80 ${
          savedFlash
            ? 'bg-green-600 text-white shadow-sm shadow-green-900/30'
            : 'bg-accent text-white hover:bg-accent/90'
        }`}
      >
        {savedFlash ? 'Saved all tracks' : saving ? 'Saving all tracks…' : coverPath ? `Save all ${tracks.length} tracks & embed artwork` : `Save all ${tracks.length} tracks`}
      </button>
    </div>
  )
}
