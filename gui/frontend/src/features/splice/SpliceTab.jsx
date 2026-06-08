import { useEffect, useMemo, useRef, useState } from 'react'
import {
  SpliceProbeMaster,
  SpliceDistributeDrift,
  SpliceSetBoundary,
  SpliceSetTrackStart,
  SpliceSetTrackDuration,
  SpliceGetPeaks,
  SpliceStartExport,
  SpliceCancelExport,
  SpliceIsExporting,
  SplicePickMasterFile,
  SplicePickArtwork,
  SplicePickOutputDir,
  SpliceSaveProject,
  SpliceMasterAudioURL,
  OpenFolder,
  EventsOn,
} from '../../wailsjs/go/main/App'
import WaveformEditor from './WaveformEditor'
import TrackPreview from './TrackPreview'
import { computeTrackSegment, formatMsPrecise, parseTimeInput } from './spliceTime'
import { formatActionError, normalizeProject } from './projectUtils'
import { reportFrontendError } from '../../lib/errorReporting'

const TRACKLIST_TIMESTAMP_RE = /^(\d{1,2}:\d{2}(?::\d{2})?(?:\.\d{1,3})?)\s+(.+)$/

function emptyProject(handoff) {
  return {
    master_path: handoff?.master_path || '',
    output_dir: handoff?.output_dir || '',
    album: {
      album: handoff?.album || '',
      album_artist: handoff?.album_artist || handoff?.artist || '',
      artist: handoff?.artist || '',
      year: handoff?.year || '',
      genre: handoff?.genre || 'DJ Mix',
      artwork_path: handoff?.artwork_path || null,
      total_tracks: null,
    },
    tracks: [],
    master_duration_ms: 0,
  }
}

function parseTrackMeta(text) {
  const name = String(text || '').trim()
  if (!name) return { title: '', artist: '' }
  const splitIdx = name.indexOf(' - ')
  if (splitIdx > 0) {
    return {
      artist: name.slice(0, splitIdx).trim(),
      title: name.slice(splitIdx + 3).trim(),
    }
  }
  return { title: name, artist: '' }
}

function parseTracklistPaste(raw) {
  const lines = String(raw || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
  if (lines.length === 0) return []

  const out = []
  for (const line of lines) {
    const match = line.match(TRACKLIST_TIMESTAMP_RE)
    if (match) {
      const startMs = parseTimeInput(match[1])
      if (startMs == null) continue
      const { title, artist } = parseTrackMeta(match[2])
      out.push({
        title,
        artist,
        start_ms: startMs,
        duration_ms: 60000,
        track_number: out.length + 1,
      })
      continue
    }

    // Continuation lines (common in DJ tracklists), e.g. "w/ ..."
    if (out.length > 0) {
      const prev = out[out.length - 1]
      prev.title = `${prev.title} ${line}`.trim()
    }
  }

  for (let i = 0; i < out.length - 1; i++) {
    const dur = Math.max(1000, out[i + 1].start_ms - out[i].start_ms)
    out[i].duration_ms = dur
  }
  return out
}

export default function SpliceTab({ handoff, onHandoffConsumed }) {
  const [project, setProject] = useState(() => emptyProject(handoff))
  const [probe, setProbe] = useState(null)
  const [peaks, setPeaks] = useState(null)
  const [selectedBoundary, setSelectedBoundary] = useState(1)
  const [exporting, setExporting] = useState(false)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [bulkCount, setBulkCount] = useState(5)
  const [tracklistInput, setTracklistInput] = useState('')
  const [selectedTrackIdx, setSelectedTrackIdx] = useState(-1)
  const [masterAudioURL, setMasterAudioURL] = useState('')
  const [albumDefaultsApplied, setAlbumDefaultsApplied] = useState(false)
  const [importingTracklist, setImportingTracklist] = useState(false)
  const albumDefaultsTimerRef = useRef(null)

  useEffect(() => {
    return () => {
      if (albumDefaultsTimerRef.current) clearTimeout(albumDefaultsTimerRef.current)
    }
  }, [])

  useEffect(() => {
    if (!project.master_path) {
      setMasterAudioURL('')
      return undefined
    }
    let cancelled = false
    ;(async () => {
      try {
        const url = await Promise.resolve(SpliceMasterAudioURL(project.master_path))
        if (!cancelled) {
          setMasterAudioURL(typeof url === 'string' ? url : '')
        }
      } catch {
        if (!cancelled) setMasterAudioURL('')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [project.master_path])

  useEffect(() => {
    if (!handoff?.master_path) return
    const next = emptyProject(handoff)
    setProject(next)
    loadMaster(handoff.master_path, next)
    onHandoffConsumed?.()
  }, [handoff])

  useEffect(() => {
    const off = EventsOn('splice:event', (ev) => {
      if (ev?.message) setStatus(ev.message)
      if (ev?.type === 'splice_complete') {
        setExporting(false)
        setStatus(ev.message)
      }
      if (ev?.type === 'splice_error') {
        setExporting(false)
        setError(ev.message)
      }
      if (ev?.type === 'splice_progress') {
        setExporting(true)
        setError('')
      }
    })
    return () => off?.()
  }, [])

  const boundaries = useMemo(() => {
    if (!Array.isArray(project.tracks) || project.tracks.length === 0) return [0]
    return project.tracks.map((t, i) => (i === 0 ? (t.start_ms ?? 0) : (t.start_ms ?? 0)))
  }, [project.tracks])

  const selectedTrack = useMemo(() => {
    if (selectedTrackIdx < 0 || selectedTrackIdx >= project.tracks.length) return null
    return project.tracks[selectedTrackIdx]
  }, [project.tracks, selectedTrackIdx])

  const selectedSegment = useMemo(
    () => computeTrackSegment(project.tracks, project.master_duration_ms, selectedTrackIdx),
    [project.tracks, project.master_duration_ms, selectedTrackIdx],
  )

  const safeSetProject = (next, fallback = project) => {
    setProject(normalizeProject(next, fallback))
  }

  const loadMaster = async (path, baseProject = project) => {
    if (!path) return
    setError('')
    try {
      const info = await SpliceProbeMaster(path)
      const peakData = await SpliceGetPeaks(path, 3000)
      setProbe(info)
      setPeaks(peakData)
      const updated = normalizeProject({
        ...baseProject,
        master_path: path,
        master_duration_ms: info.duration_ms,
      }, baseProject)
      if (updated.tracks.length > 0) {
        const drifted = await SpliceDistributeDrift(updated)
        safeSetProject(drifted, updated)
      } else {
        safeSetProject(updated, baseProject)
      }
    } catch (e) {
      const msg = formatActionError(e, 'Load master file')
      reportFrontendError('SpliceTab.loadMaster', e)
      setError(msg)
    }
  }

  const pickMaster = async () => {
    const path = await SplicePickMasterFile()
    if (path) await loadMaster(path)
  }

  const updateAlbum = (key, value) => {
    setProject((p) => ({ ...p, album: { ...p.album, [key]: value } }))
  }

  const updateTrack = (idx, key, value) => {
    setProject((p) => {
      const tracks = [...p.tracks]
      tracks[idx] = { ...tracks[idx], [key]: value }
      return { ...p, tracks }
    })
  }

  const updateSelectedTrack = (key, value) => {
    if (selectedTrackIdx < 0) return
    updateTrack(selectedTrackIdx, key, value)
  }

  const addTrack = () => {
    setProject((p) => ({
      ...p,
      tracks: [...p.tracks, { title: `Track ${p.tracks.length + 1}`, duration_ms: 60000 }],
    }))
  }

  const addTracksBulk = () => {
    const count = Number.isFinite(Number(bulkCount)) ? Math.max(1, Math.min(200, Number(bulkCount))) : 1
    setProject((p) => {
      const start = p.tracks.length
      const additions = Array.from({ length: count }, (_, i) => ({
        title: `Track ${start + i + 1}`,
        duration_ms: 60000,
      }))
      return { ...p, tracks: [...p.tracks, ...additions] }
    })
  }

  const handleBoundaryChange = async (boundaryIndex, positionMs) => {
    try {
      const updated = await SpliceSetBoundary(project, boundaryIndex, positionMs)
      safeSetProject(updated, project)
    } catch (e) {
      reportFrontendError('SpliceTab.handleBoundaryChange', e)
      setError(formatActionError(e, 'Move boundary'))
    }
  }

  const handleStartCommit = async (row, raw) => {
    const ms = parseTimeInput(raw)
    if (ms == null) {
      setError('Invalid start format. Use m:ss.ms, h:mm:ss.ms, or seconds.')
      return
    }
    setError('')
    try {
      const updated = await SpliceSetTrackStart(project, row, ms)
      safeSetProject(updated, project)
    } catch (e) {
      reportFrontendError('SpliceTab.handleStartCommit', e)
      setError(formatActionError(e, 'Update start time'))
    }
  }

  const handleDurationCommit = async (row, raw) => {
    const ms = parseTimeInput(raw)
    if (ms == null) {
      setError('Invalid duration format. Use m:ss.ms, h:mm:ss.ms, or seconds.')
      return
    }
    setError('')
    try {
      const updated = await SpliceSetTrackDuration(project, row, ms)
      safeSetProject(updated, project)
    } catch (e) {
      reportFrontendError('SpliceTab.handleDurationCommit', e)
      setError(formatActionError(e, 'Update duration'))
    }
  }

  const handleDistribute = async () => {
    if (!project.tracks.length) {
      setError('Add tracks before fitting durations to the master file.')
      return
    }
    if (!project.master_duration_ms) {
      setError('Open a master file first so durations can be fit to its length.')
      return
    }
    setError('')
    try {
      const updated = await SpliceDistributeDrift(project)
      safeSetProject(updated, project)
      setStatus('Fitted track durations to master length.')
    } catch (e) {
      reportFrontendError('SpliceTab.handleDistribute', e)
      setError(formatActionError(e, 'Fit durations to master'))
    }
  }

  const handleExport = async () => {
    setError('')
    setExporting(true)
    try {
      await SpliceStartExport(project)
      const running = await SpliceIsExporting()
      if (!running) setExporting(false)
    } catch (e) {
      setExporting(false)
      reportFrontendError('SpliceTab.handleExport', e)
      setError(formatActionError(e, 'Export tracks'))
    }
  }

  const handleSave = async () => {
    const path = await SpliceSaveProject(project)
    setStatus(`Saved project to ${path}`)
  }

  const handleParseTracklist = async () => {
    setImportingTracklist(true)
    setError('')
    try {
      const parsed = parseTracklistPaste(tracklistInput)
      if (parsed.length === 0) {
        setError('No timestamped tracks found. Paste lines like "0:12 Artist - Title".')
        return
      }

      let next = normalizeProject({
        ...project,
        tracks: parsed,
        album: {
          ...project.album,
          total_tracks: parsed.length,
        },
      }, project)

      if (project.master_duration_ms > 0) {
        next = normalizeProject(await SpliceDistributeDrift(next), next)
      } else if (project.master_path) {
        setStatus(`Imported ${parsed.length} tracks. Open or reload the master file to fit durations.`)
      }

      safeSetProject(next, project)
      setSelectedTrackIdx(0)
      setSelectedBoundary(Math.min(1, Math.max(0, next.tracks.length - 1)))
      setStatus(`Imported ${parsed.length} tracks from pasted tracklist.`)
    } catch (e) {
      reportFrontendError('SpliceTab.handleParseTracklist', e, tracklistInput.slice(0, 500))
      setError(formatActionError(e, 'Build tracks from paste'))
    } finally {
      setImportingTracklist(false)
    }
  }

  const applyAlbumDefaultsToTracks = () => {
    if (!project.tracks.length) {
      setError('Add tracks before applying album defaults.')
      return
    }
    const trackCount = project.tracks.length
    setProject((p) => {
      const tracks = p.tracks.map((t, i) => ({
        ...t,
        album: p.album.album || t.album || '',
        album_artist: p.album.album_artist || t.album_artist || '',
        genre: p.album.genre || t.genre || '',
        year: p.album.year || t.year || '',
        track_number: t.track_number || i + 1,
      }))
      return { ...p, tracks }
    })
    setError('')
    setStatus(`Applied album defaults to ${trackCount} track${trackCount === 1 ? '' : 's'}.`)
    setAlbumDefaultsApplied(true)
    if (albumDefaultsTimerRef.current) clearTimeout(albumDefaultsTimerRef.current)
    albumDefaultsTimerRef.current = setTimeout(() => setAlbumDefaultsApplied(false), 2500)
  }

  return (
    <div className="flex h-full flex-col gap-4 overflow-auto">
      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h2 className="text-xl font-semibold">Split mix</h2>
        <p className="mt-1 text-sm text-white/50">
          Slice a long DJ set or live recording into tagged AAC tracks for your Apple Music library.
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          <button type="button" onClick={pickMaster} className="rounded-lg bg-accent px-4 py-2 text-sm font-medium">
            Open master file
          </button>
          <button
            type="button"
            onClick={async () => {
              const dir = await SplicePickOutputDir()
              if (dir) setProject((p) => ({ ...p, output_dir: dir }))
            }}
            className="rounded-lg border border-white/15 px-4 py-2 text-sm"
          >
            Output folder
          </button>
          {project.output_dir && (
            <button type="button" onClick={() => OpenFolder(project.output_dir)} className="rounded-lg border border-white/15 px-4 py-2 text-sm">
              Open output
            </button>
          )}
        </div>
        {project.master_path && (
          <p className="mt-2 truncate text-xs text-white/45">{project.master_path}</p>
        )}
        {probe?.summary && <p className="mt-1 text-xs text-white/60">{probe.summary}</p>}
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="mb-2 font-medium">Paste tracklist</h3>
        <p className="mb-2 text-xs text-white/55">
          Paste DJ set timestamps like <code>0:12 Artist - Track</code>. Lines without timestamps are appended to the previous title.
        </p>
        <textarea
          value={tracklistInput}
          onChange={(e) => setTracklistInput(e.target.value)}
          className="h-28 w-full rounded-lg border border-white/10 bg-black/20 p-3 text-sm"
          placeholder={`0:00 Intro\n0:12 Artist - Track Name\n1:49 Artist - Track Name\nw/ Acapella`}
        />
        <div className="mt-2 flex gap-2">
          <button
            type="button"
            disabled={importingTracklist}
            onClick={handleParseTracklist}
            className="rounded-lg bg-accent px-3 py-2 text-sm font-medium disabled:opacity-50"
          >
            {importingTracklist ? 'Building tracks…' : 'Build tracks from paste'}
          </button>
          <button type="button" onClick={() => setTracklistInput('')} className="rounded-lg border border-white/15 px-3 py-2 text-sm">
            Clear
          </button>
        </div>
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <div className="mb-3 flex flex-wrap items-end gap-2">
          <button type="button" onClick={handleDistribute} className="rounded-lg border border-white/15 px-3 py-2 text-sm">
            Fit durations to master
          </button>
          <button type="button" onClick={addTrack} className="rounded-lg border border-white/15 px-3 py-2 text-sm">
            Add track
          </button>
          <div className="flex items-center gap-2">
            <input
              type="number"
              min={1}
              max={200}
              value={bulkCount}
              onChange={(e) => setBulkCount(e.target.value)}
              className="w-20 rounded-lg border border-white/10 bg-black/20 px-2 py-2 text-sm"
            />
            <button type="button" onClick={addTracksBulk} className="rounded-lg border border-white/15 px-3 py-2 text-sm">
              Bulk add
            </button>
          </div>
        </div>

        <WaveformEditor
          peaks={peaks}
          durationMs={project.master_duration_ms}
          boundaries={boundaries}
          selectedBoundary={selectedBoundary}
          onSelectBoundary={setSelectedBoundary}
          onBoundaryChange={handleBoundaryChange}
        />
      </section>

      <section className="grid gap-4 lg:grid-cols-3">
        <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
          <h3 className="font-medium">Album metadata (mass apply)</h3>
          <div className="mt-3 grid gap-2 text-sm">
            {['album', 'album_artist', 'artist', 'year', 'genre', 'total_tracks'].map((key) => (
              <div key={key}>
                <label className="text-xs capitalize text-white/50">{key.replace('_', ' ')}</label>
                <input
                  type={key === 'total_tracks' ? 'number' : 'text'}
                  min={key === 'total_tracks' ? 1 : undefined}
                  value={project.album?.[key] ?? ''}
                  onChange={(e) => {
                    if (key === 'total_tracks') {
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
            <button
              type="button"
              onClick={async () => {
                const path = await SplicePickArtwork()
                if (path) updateAlbum('artwork_path', path)
              }}
              className="mt-1 rounded-lg border border-white/15 px-3 py-2 text-left text-xs"
            >
              Artwork: {project.album?.artwork_path || 'None selected'}
            </button>
            <button
              type="button"
              onClick={applyAlbumDefaultsToTracks}
              disabled={!project.tracks.length}
              title={project.tracks.length ? 'Copy album metadata to every track' : 'Add tracks first'}
              className={`mt-2 w-full rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200 ${
                albumDefaultsApplied
                  ? 'bg-emerald-600 text-white ring-2 ring-emerald-400/50'
                  : 'bg-accent text-white hover:bg-accent-muted disabled:cursor-not-allowed disabled:opacity-40'
              }`}
            >
              {albumDefaultsApplied ? (
                <span className="flex items-center justify-center gap-2">
                  <span aria-hidden className="text-base">✓</span>
                  Applied to {project.tracks.length} track{project.tracks.length === 1 ? '' : 's'}
                </span>
              ) : (
                'Apply album defaults to all tracks'
              )}
            </button>
            {albumDefaultsApplied && (
              <p className="mt-2 rounded-lg border border-emerald-400/30 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-200">
                Album, album artist, genre, year, and track numbers updated on every track.
              </p>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-white/10 bg-surface-raised p-4 lg:col-span-2">
          <h3 className="font-medium">Tracks ({(project.tracks || []).length})</h3>
          <div className="mt-2 max-h-64 overflow-auto">
            <table className="w-full text-left text-sm">
              <thead className="text-xs text-white/45">
                <tr>
                  <th className="py-1 pr-2">#</th>
                  <th className="py-1 pr-2">Title</th>
                  <th className="py-1 pr-2">Start</th>
                  <th className="py-1">Duration</th>
                </tr>
              </thead>
              <tbody>
                {(project.tracks || []).map((t, i) => (
                  <tr key={i} className={`cursor-pointer border-t border-white/5 ${selectedTrackIdx === i ? 'bg-accent/10' : ''}`} onClick={() => setSelectedTrackIdx(i)}>
                    <td className="py-1 pr-2 text-white/50">{i + 1}</td>
                    <td className="py-1 pr-2">
                      <input
                        value={t.title || ''}
                        onChange={(e) => updateTrack(i, 'title', e.target.value)}
                        className="w-full rounded bg-black/20 px-2 py-1"
                      />
                    </td>
                    <td className="py-1 pr-2">
                      {i === 0 && !(t.start_ms > 0) ? (
                        <span className="text-xs text-white/60">{formatMsPrecise(0)}</span>
                      ) : (
                        <input
                          key={`start-${i}-${t.start_ms ?? boundaries[i] ?? 0}`}
                          defaultValue={formatMsPrecise(t.start_ms ?? boundaries[i] ?? 0, (t.start_ms ?? boundaries[i] ?? 0) >= 3600000)}
                          onBlur={(e) => handleStartCommit(i, e.target.value)}
                          className="w-28 rounded bg-black/20 px-2 py-1 text-xs text-white/80"
                          title="m:ss.ms or h:mm:ss.ms"
                        />
                      )}
                    </td>
                    <td className="py-1">
                      <input
                        key={`dur-${i}-${t.duration_ms}`}
                        defaultValue={formatMsPrecise(t.duration_ms ?? 0, (t.duration_ms ?? 0) >= 3600000)}
                        onBlur={(e) => handleDurationCommit(i, e.target.value)}
                        className="w-28 rounded bg-black/20 px-2 py-1 text-xs text-white/80"
                        title="m:ss.ms or h:mm:ss.ms"
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
        <h3 className="font-medium">Selected track metadata</h3>
        {!selectedTrack ? (
          <p className="mt-2 text-sm text-white/55">Select a track row to edit per-track metadata overrides.</p>
        ) : (
          <div className="mt-3 space-y-4">
            <TrackPreview
              audioURL={masterAudioURL}
              masterDurationMs={project.master_duration_ms}
              trackNumber={selectedTrackIdx + 1}
              title={selectedTrack.title}
              startMs={selectedSegment?.startMs ?? 0}
              endMs={selectedSegment?.endMs ?? 0}
            />

            <div className="grid gap-3 text-sm lg:grid-cols-3">
            <div>
              <label className="text-xs text-white/50">Start time</label>
              {selectedTrackIdx === 0 ? (
                <p className="mt-1 rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white/60">
                  {formatMsPrecise(0)}
                </p>
              ) : (
                <input
                  key={`sel-start-${selectedTrackIdx}-${selectedTrack.start_ms ?? selectedSegment?.startMs ?? 0}`}
                  defaultValue={formatMsPrecise(
                    selectedTrack.start_ms ?? selectedSegment?.startMs ?? 0,
                    (selectedTrack.start_ms ?? selectedSegment?.startMs ?? 0) >= 3600000,
                  )}
                  onBlur={(e) => handleStartCommit(selectedTrackIdx, e.target.value)}
                  className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
                  title="m:ss.ms or h:mm:ss.ms"
                />
              )}
            </div>
            <div>
              <label className="text-xs text-white/50">Duration</label>
              <input
                key={`sel-dur-${selectedTrackIdx}-${selectedTrack.duration_ms}`}
                defaultValue={formatMsPrecise(selectedTrack.duration_ms ?? selectedSegment?.durationMs ?? 0)}
                onBlur={(e) => handleDurationCommit(selectedTrackIdx, e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
                title="m:ss.ms or h:mm:ss.ms"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Title</label>
              <input
                value={selectedTrack.title || ''}
                onChange={(e) => updateSelectedTrack('title', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Artist (track performer)</label>
              <input
                value={selectedTrack.artist || ''}
                onChange={(e) => updateSelectedTrack('artist', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Track number</label>
              <input
                type="number"
                min={1}
                value={selectedTrack.track_number ?? ''}
                onChange={(e) => {
                  const n = Number(e.target.value)
                  updateSelectedTrack('track_number', Number.isFinite(n) && n > 0 ? n : null)
                }}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Album override</label>
              <input
                value={selectedTrack.album || ''}
                onChange={(e) => updateSelectedTrack('album', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Album artist override</label>
              <input
                value={selectedTrack.album_artist || ''}
                onChange={(e) => updateSelectedTrack('album_artist', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Genre override</label>
              <input
                value={selectedTrack.genre || ''}
                onChange={(e) => updateSelectedTrack('genre', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Year override</label>
              <input
                value={selectedTrack.year || ''}
                onChange={(e) => updateSelectedTrack('year', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Disc number</label>
              <input
                type="number"
                min={1}
                value={selectedTrack.disc_number ?? ''}
                onChange={(e) => {
                  const n = Number(e.target.value)
                  updateSelectedTrack('disc_number', Number.isFinite(n) && n > 0 ? n : null)
                }}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">Disc total</label>
              <input
                type="number"
                min={1}
                value={selectedTrack.disc_total ?? ''}
                onChange={(e) => {
                  const n = Number(e.target.value)
                  updateSelectedTrack('disc_total', Number.isFinite(n) && n > 0 ? n : null)
                }}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2"
              />
            </div>
            </div>
          </div>
        )}
      </section>

      <section className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={exporting || !project.master_path || project.tracks.length === 0}
          onClick={handleExport}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium disabled:opacity-40"
        >
          {exporting ? 'Exporting…' : 'Export tracks'}
        </button>
        <button type="button" onClick={handleSave} className="rounded-lg border border-white/15 px-4 py-2 text-sm">
          Save project
        </button>
        {exporting && (
          <button type="button" onClick={SpliceCancelExport} className="rounded-lg border border-red-400/40 px-4 py-2 text-sm text-red-200">
            Cancel export
          </button>
        )}
      </section>

      {status && <p className="text-sm text-green-200/90">{status}</p>}
      {error && (
        <div className="rounded-lg border border-red-400/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          <p>{error}</p>
          <p className="mt-1 text-xs text-red-200/70">See Activity → log file or %APPDATA%\AuraAudioDownloader\logs\app.log for details.</p>
        </div>
      )}
    </div>
  )
}
