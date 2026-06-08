export default function QueueTab({ logs, downloading, onCancel, onOpenFolder }) {
  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Download queue</h2>
          <p className="text-sm text-white/50">
            {downloading ? 'Download in progress…' : 'Idle'}
          </p>
        </div>
        <div className="flex gap-2">
          {downloading && (
            <button onClick={onCancel} className="rounded-lg border border-red-500/50 px-4 py-2 text-sm text-red-400">
              Cancel
            </button>
          )}
          <button onClick={onOpenFolder} className="rounded-lg bg-surface-raised px-4 py-2 text-sm hover:bg-surface-hover">
            Open output folder
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto rounded-xl border border-white/10 bg-black/30 p-4 font-mono text-xs">
        {logs.length === 0 ? (
          <p className="text-white/40">Activity log will appear here when downloads run.</p>
        ) : (
          logs.map((l, i) => (
            <div key={i} className="border-b border-white/5 py-1 text-white/80">
              <span className="text-white/40">[{l.time}]</span> {l.msg}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
