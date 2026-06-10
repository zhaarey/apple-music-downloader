import { useMemo, useState } from 'react'
import { RevealInFolder } from '../../wailsjs/go/main/App'

function TrackArtwork({ track, className = 'h-11 w-11' }) {
  if (track.art_url) {
    return <img src={track.art_url} alt="" className={`${className} shrink-0 rounded-md object-cover`} />
  }
  return (
    <div
      className={`${className} flex shrink-0 items-center justify-center rounded-md bg-white/5 text-xs text-white/30`}
    >
      ♪
    </div>
  )
}

function TrackMetaLines({ track, compact = false }) {
  const albumLine = [track.album, track.album_artist && track.album_artist !== track.artist ? track.album_artist : '']
    .filter(Boolean)
    .join(' · ')
  if (!albumLine && !track.genre && !track.year) return null
  return (
    <div className={`truncate text-white/40 ${compact ? 'text-[10px]' : 'text-xs'}`}>
      {albumLine && <p className="truncate">{albumLine}</p>}
      {(track.genre || track.year) && (
        <p className="truncate">
          {[track.genre, track.year].filter(Boolean).join(' · ')}
        </p>
      )}
    </div>
  )
}

export default function AppleTrackReviewList({
  tracks = [],
  originalTracks = [],
  selected,
  onToggleTrack,
  onSelectAll,
  onClear,
  allowUrlEdit = false,
  onSaveTrackUrl,
  onDiskByNum = {},
  disabled = false,
  title = 'Tracks',
  maxHeightClass = 'max-h-64 xl:max-h-[min(70vh,36rem)]',
}) {
  const [editingNum, setEditingNum] = useState(null)
  const [urlDraft, setUrlDraft] = useState('')
  const [urlErrors, setUrlErrors] = useState({})
  const [savingNum, setSavingNum] = useState(null)

  const origByNum = useMemo(
    () => Object.fromEntries((originalTracks.length ? originalTracks : tracks).map((t) => [t.num, t.url])),
    [originalTracks, tracks],
  )

  const startEdit = (track) => {
    setEditingNum(track.num)
    setUrlDraft(track.url || '')
    setUrlErrors((prev) => ({ ...prev, [track.num]: '' }))
  }

  const cancelEdit = () => {
    setEditingNum(null)
    setUrlDraft('')
  }

  const saveEdit = async (num) => {
    const trimmed = urlDraft.trim()
    if (!trimmed) {
      setUrlErrors((prev) => ({ ...prev, [num]: 'Enter an Apple Music song link.' }))
      return
    }
    if (!/music\.apple\.com/i.test(trimmed)) {
      setUrlErrors((prev) => ({ ...prev, [num]: 'Use an Apple Music song or music video link.' }))
      return
    }
    setSavingNum(num)
    try {
      const err = await onSaveTrackUrl?.(num, trimmed)
      if (err) {
        setUrlErrors((prev) => ({ ...prev, [num]: err }))
        return
      }
      setEditingNum(null)
      setUrlDraft('')
    } finally {
      setSavingNum(null)
    }
  }

  return (
    <div className="rounded-xl border border-white/10 bg-surface-raised">
      <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
        <span className="text-sm font-medium">{title}</span>
        <div className="flex gap-3 text-xs">
          <button type="button" onClick={onSelectAll} className="text-accent hover:underline" disabled={disabled}>
            Select all
          </button>
          <button type="button" onClick={onClear} className="text-white/50 hover:underline" disabled={disabled}>
            Clear
          </button>
        </div>
      </div>
      <ul className={`${maxHeightClass} divide-y divide-white/5 overflow-y-auto`}>
        {tracks.map((t) => {
          const isEditing = editingNum === t.num
          const urlChanged = allowUrlEdit && t.url && origByNum[t.num] && t.url !== origByNum[t.num]
          return (
            <li key={t.num} className="px-4 py-2.5 text-sm hover:bg-white/[0.02]">
              <div className="flex items-start gap-3">
                <input
                  type="checkbox"
                  checked={selected.has(t.num)}
                  onChange={() => onToggleTrack(t.num)}
                  disabled={disabled}
                  className="mt-2 shrink-0"
                />
                <TrackArtwork track={t} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">{t.name}</p>
                      <p className="truncate text-xs text-white/55">{t.artist}</p>
                      <TrackMetaLines track={t} />
                    </div>
                    <div className="flex shrink-0 flex-col items-end gap-1">
                      <span className="text-xs text-white/40">{t.duration}</span>
                      {t.explicit && <span className="text-[10px] text-white/50">E</span>}
                      {t.is_mv && <span className="text-[10px] text-white/50">MV</span>}
                      {urlChanged && (
                        <span className="rounded-full border border-accent/30 bg-accent/10 px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wide text-accent">
                          Swapped
                        </span>
                      )}
                    </div>
                  </div>

                  {allowUrlEdit && !isEditing && (
                    <div className="mt-1.5 flex flex-wrap items-center gap-2">
                      {t.url && (
                        <p className="min-w-0 flex-1 truncate font-mono text-[10px] text-white/30" title={t.url}>
                          {t.url}
                        </p>
                      )}
                      {!disabled && (
                        <button
                          type="button"
                          onClick={() => startEdit(t)}
                          className="shrink-0 text-[10px] text-accent hover:underline"
                        >
                          Edit link
                        </button>
                      )}
                    </div>
                  )}

                  {allowUrlEdit && isEditing && (
                    <div className="mt-2 space-y-2 rounded-lg border border-white/10 bg-black/20 p-2.5">
                      <label className="text-[10px] font-medium uppercase tracking-wide text-white/45">
                        Song URL
                      </label>
                      <input
                        type="url"
                        value={urlDraft}
                        onChange={(e) => setUrlDraft(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') saveEdit(t.num)
                          if (e.key === 'Escape') cancelEdit()
                        }}
                        placeholder="https://music.apple.com/.../song/..."
                        className="w-full rounded-lg border border-white/15 bg-surface px-2.5 py-1.5 text-xs focus:border-accent focus:outline-none"
                        autoFocus
                        disabled={savingNum === t.num}
                      />
                      {urlErrors[t.num] && <p className="text-[11px] text-red-300">{urlErrors[t.num]}</p>}
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          onClick={() => saveEdit(t.num)}
                          disabled={savingNum === t.num}
                          className="rounded-md bg-accent px-2.5 py-1 text-[11px] font-medium hover:bg-accent-muted disabled:opacity-50"
                        >
                          {savingNum === t.num ? 'Loading…' : 'Save & refresh'}
                        </button>
                        <button
                          type="button"
                          onClick={cancelEdit}
                          className="rounded-md border border-white/15 px-2.5 py-1 text-[11px] text-white/60 hover:bg-white/5"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  )}
                </div>

                {onDiskByNum[t.num] && (
                  <button
                    type="button"
                    title={onDiskByNum[t.num].existing_path || 'Already on disk'}
                    onClick={() =>
                      onDiskByNum[t.num].existing_path && RevealInFolder(onDiskByNum[t.num].existing_path)
                    }
                    className="mt-1 shrink-0 rounded-full border border-emerald-500/40 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-emerald-300 hover:bg-emerald-500/20"
                  >
                    On disk
                  </button>
                )}
              </div>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
