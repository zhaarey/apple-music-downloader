export default function AppleDownloadModeSwitch({ mode, onChange, disabled }) {
  return (
    <div className="rounded-xl border border-accent/25 bg-gradient-to-br from-accent/10 to-transparent p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-white">Add links</p>
          <p className="mt-0.5 text-xs text-white/55">
            One album at a time, or paste many URLs and download them as a queue.
          </p>
        </div>
        <div className="flex shrink-0 rounded-lg bg-black/30 p-1 ring-1 ring-white/10">
          <button
            type="button"
            disabled={disabled}
            onClick={() => onChange('single')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition ${
              mode === 'single'
                ? 'bg-accent text-white shadow-sm'
                : 'text-white/60 hover:text-white disabled:opacity-40'
            }`}
          >
            Single link
          </button>
          <button
            type="button"
            disabled={disabled}
            onClick={() => onChange('bulk')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition ${
              mode === 'bulk'
                ? 'bg-accent text-white shadow-sm'
                : 'text-white/60 hover:text-white disabled:opacity-40'
            }`}
          >
            Bulk queue
          </button>
        </div>
      </div>
    </div>
  )
}
