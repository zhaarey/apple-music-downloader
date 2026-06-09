import ArtworkPreview from './ArtworkPreview'

/**
 * Shared artwork panel — preview, replace, and remove (Tag Editor & Splice Suite).
 */
export default function ArtworkEditor({
  previewSrc = '',
  onReplace,
  onRemove,
  disabled = false,
  replaceLabel = 'Replace artwork',
  removeLabel = 'Remove artwork',
  title = 'Artwork',
  className = '',
}) {
  const hasArtwork = Boolean(previewSrc)

  return (
    <section className={`rounded-xl border border-white/10 bg-surface-raised p-4 ${className}`}>
      <h3 className="text-sm font-medium">{title}</h3>
      <ArtworkPreview
        src={previewSrc}
        className="mt-3 flex aspect-square items-center justify-center rounded-lg bg-black/30"
      />
      <div className="mt-3 flex flex-col gap-2">
        <button
          type="button"
          onClick={onReplace}
          disabled={disabled}
          className="rounded-lg border border-white/15 px-3 py-2 text-xs transition-colors duration-200 ease-apple hover:bg-white/5 disabled:opacity-50"
        >
          {hasArtwork ? replaceLabel : 'Add artwork'}
        </button>
        {hasArtwork && onRemove && (
          <button
            type="button"
            disabled={disabled}
            onClick={onRemove}
            className="rounded-lg border border-white/15 px-3 py-2 text-xs text-white/70 transition-colors duration-200 ease-apple hover:bg-white/5 disabled:opacity-50"
          >
            {removeLabel}
          </button>
        )}
      </div>
    </section>
  )
}
