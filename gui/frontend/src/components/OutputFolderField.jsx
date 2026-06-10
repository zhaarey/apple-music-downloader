import { useState } from 'react'

export default function OutputFolderField({
  label = 'Download folder',
  hint = '',
  value = '',
  disabled = false,
  onBrowse,
  onOpen,
  onSavePath,
}) {
  const [draft, setDraft] = useState('')
  const [editing, setEditing] = useState(false)
  const display = editing ? draft : value

  const startEdit = () => {
    setDraft(value || '')
    setEditing(true)
  }

  const commitEdit = async () => {
    setEditing(false)
    const trimmed = draft.trim()
    if (trimmed && trimmed !== value) {
      await onSavePath?.(trimmed)
    }
  }

  return (
    <div className="rounded-xl border border-white/10 bg-surface-raised/60 p-3">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <label className="text-xs font-medium text-white/60">{label}</label>
          {hint && <p className="mt-0.5 text-[11px] text-white/40">{hint}</p>}
        </div>
        <div className="flex shrink-0 gap-2">
          <button
            type="button"
            disabled={disabled || !onBrowse}
            onClick={onBrowse}
            className="rounded-lg border border-white/15 px-3 py-1.5 text-xs hover:bg-white/5 disabled:opacity-40"
          >
            Browse…
          </button>
          {value && onOpen && (
            <button
              type="button"
              disabled={disabled}
              onClick={onOpen}
              className="rounded-lg border border-white/15 px-3 py-1.5 text-xs hover:bg-white/5 disabled:opacity-40"
            >
              Open
            </button>
          )}
        </div>
      </div>
      <input
        value={display}
        readOnly={!editing}
        disabled={disabled}
        onFocus={startEdit}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => {
          void commitEdit()
        }}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault()
            e.currentTarget.blur()
          }
          if (e.key === 'Escape') {
            setEditing(false)
            setDraft(value || '')
            e.currentTarget.blur()
          }
        }}
        placeholder="Choose a folder — files save as Artist → Album → tracks"
        className={`mt-2 w-full rounded-lg border px-3 py-2 text-sm ${
          editing
            ? 'border-accent/40 bg-surface text-white'
            : 'cursor-pointer border-white/10 bg-black/20 text-white/80'
        }`}
        title={value || 'No folder set'}
      />
    </div>
  )
}
