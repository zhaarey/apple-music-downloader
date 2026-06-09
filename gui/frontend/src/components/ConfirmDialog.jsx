export default function ConfirmDialog({
  open,
  title,
  message,
  details = [],
  confirmLabel = 'Continue',
  cancelLabel = 'Cancel',
  destructive = true,
  busy = false,
  onConfirm,
  onCancel,
}) {
  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
    >
      <div className="w-full max-w-md rounded-xl border border-white/15 bg-surface-raised p-5 shadow-2xl">
        <h2 id="confirm-dialog-title" className="text-lg font-semibold text-white">
          {title}
        </h2>
        {message ? <p className="mt-2 text-sm text-white/75">{message}</p> : null}
        {details.length > 0 && (
          <ul className="mt-3 space-y-1 rounded-lg border border-amber-500/25 bg-amber-500/10 px-3 py-2 text-sm text-amber-100">
            {details.map((line) => (
              <li key={line} className="break-words">
                {line}
              </li>
            ))}
          </ul>
        )}
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={onCancel}
            className="rounded-lg border border-white/15 px-4 py-2 text-sm hover:bg-white/5 disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={onConfirm}
            className={`rounded-lg px-4 py-2 text-sm font-medium disabled:opacity-50 ${
              destructive
                ? 'bg-red-600 text-white hover:bg-red-500'
                : 'bg-accent text-white hover:bg-accent/90'
            }`}
          >
            {busy ? 'Working…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
