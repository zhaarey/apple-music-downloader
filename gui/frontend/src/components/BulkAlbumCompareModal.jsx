import { RevealInFolder } from '../wailsjs/go/main/App'
import { compareRowsForItem, isFullySkippedAlbum, summarizeItemTracks } from '../lib/parseBulkUrls'

function ChoiceToggle({ value, onChange, onDisk, disabled }) {
  const options = onDisk
    ? [
        { id: 'skip', label: 'Skip', hint: 'Keep existing file' },
        { id: 'redownload', label: 'Re-download', hint: 'Replace on disk' },
      ]
    : [
        { id: 'download', label: 'Download', hint: 'Fetch this track' },
        { id: 'skip', label: 'Skip', hint: 'Do not download' },
      ]

  return (
    <div className="flex flex-wrap gap-1">
      {options.map((opt) => (
        <button
          key={opt.id}
          type="button"
          disabled={disabled}
          title={opt.hint}
          onClick={() => onChange(opt.id)}
          className={`rounded-md px-2 py-1 text-[10px] font-medium transition disabled:opacity-40 ${
            value === opt.id
              ? opt.id === 'redownload'
                ? 'bg-amber-500/25 text-amber-200 ring-1 ring-amber-500/40'
                : opt.id === 'skip'
                  ? 'bg-emerald-500/15 text-emerald-300 ring-1 ring-emerald-500/30'
                  : 'bg-accent/25 text-accent ring-1 ring-accent/40'
              : 'bg-white/5 text-white/50 hover:bg-white/10 hover:text-white/80'
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

export default function BulkAlbumCompareModal({ item, open, onClose, onSave, onRemoveAlbum }) {
  if (!open || !item) return null

  const rows = compareRowsForItem(item)
  const summary = summarizeItemTracks(item)
  const fullySkipped = isFullySkippedAlbum(item)

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
    >
      <div className="flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-xl border border-white/15 bg-surface-raised shadow-2xl">
        <div className="flex items-start gap-4 border-b border-white/10 px-5 py-4">
          {item.preview?.art_url ? (
            <img src={item.preview.art_url} alt="" className="h-16 w-16 shrink-0 rounded-lg object-cover" />
          ) : (
            <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-lg bg-surface text-2xl text-white/30">
              ♪
            </div>
          )}
          <div className="min-w-0 flex-1">
            <h2 className="truncate text-lg font-semibold">{item.preview?.title || 'Compare tracks'}</h2>
            <p className="truncate text-sm text-white/50">{item.preview?.subtitle}</p>
            <p className="mt-2 text-xs text-white/45">
              {summary.download} to download · {summary.skip} skipped · {summary.redownload} re-download
            </p>
          </div>
          <button type="button" onClick={onClose} className="text-white/40 hover:text-white">
            ✕
          </button>
        </div>

        {fullySkipped && (
          <p className="border-b border-yellow-500/20 bg-yellow-500/10 px-5 py-2 text-xs text-yellow-100">
            All tracks are set to skip — this album will not be included when you start the download queue.
          </p>
        )}

        <div className="overflow-y-auto px-5 py-3">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="text-white/40">
                <th className="pb-2 pr-2 font-medium">#</th>
                <th className="pb-2 pr-2 font-medium">Track</th>
                <th className="hidden pb-2 pr-2 font-medium sm:table-cell">On disk</th>
                <th className="hidden pb-2 pr-2 font-medium md:table-cell">Would save to</th>
                <th className="pb-2 font-medium">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {rows.map((row) => (
                <tr key={row.num} className={row.onDisk ? 'bg-emerald-500/[0.03]' : ''}>
                  <td className="py-2.5 pr-2 tabular-nums text-white/40">{row.num}</td>
                  <td className="max-w-[140px] py-2.5 pr-2">
                    <p className="truncate font-medium text-white/90">{row.name}</p>
                    {row.onDisk && (
                      <span className="mt-0.5 inline-block rounded bg-emerald-500/15 px-1.5 py-0.5 text-[10px] text-emerald-300">
                        duplicate
                      </span>
                    )}
                  </td>
                  <td className="hidden max-w-[180px] py-2.5 pr-2 sm:table-cell">
                    {row.onDisk ? (
                      <div>
                        <p className="truncate text-emerald-300/90" title={row.existingPath}>
                          {row.existingPath || 'Found on disk'}
                        </p>
                        {row.existingRoot && (
                          <p className="truncate text-[10px] text-white/35">{row.existingRoot}</p>
                        )}
                        {row.existingPath && (
                          <button
                            type="button"
                            onClick={() => RevealInFolder(row.existingPath)}
                            className="mt-0.5 text-[10px] text-accent hover:underline"
                          >
                            Show file
                          </button>
                        )}
                      </div>
                    ) : (
                      <span className="text-white/35">Not found</span>
                    )}
                  </td>
                  <td className="hidden max-w-[180px] py-2.5 pr-2 md:table-cell">
                    <p className="truncate text-white/45" title={row.expectedPath}>
                      {row.expectedPath || '—'}
                    </p>
                  </td>
                  <td className="py-2.5">
                    <ChoiceToggle
                      value={row.choice}
                      onDisk={row.onDisk}
                      onChange={(choice) => onSave(item.id, row.num, choice)}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-white/10 px-5 py-4">
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => onSave(item.id, '_all_dupes', 'skip')}
              className="rounded-lg border border-emerald-500/30 px-3 py-1.5 text-xs text-emerald-300 hover:bg-emerald-500/10"
            >
              Skip all duplicates
            </button>
            <button
              type="button"
              onClick={() => onSave(item.id, '_all_dupes', 'redownload')}
              className="rounded-lg border border-amber-500/30 px-3 py-1.5 text-xs text-amber-200 hover:bg-amber-500/10"
            >
              Re-download all duplicates
            </button>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => onRemoveAlbum(item.id)}
              className="rounded-lg border border-red-500/30 px-3 py-1.5 text-xs text-red-300 hover:bg-red-500/10"
            >
              Remove from queue
            </button>
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg bg-accent px-4 py-1.5 text-xs font-medium hover:bg-accent-muted"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
