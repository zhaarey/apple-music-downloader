import ArtworkPreview from './ArtworkPreview'

/**
 * Artwork panel with Apple Music iOS album-view optimization controls.
 */
export default function ArtworkAppleOptions({
  previewSrc = '',
  optimizedPreviewSrc = '',
  onReplace,
  onRemove,
  onUseFolderCover,
  showFolderCover = false,
  disabled = false,
  optimizeArtwork = true,
  onOptimizeArtworkChange,
  mp4boxReembed = false,
  onMp4boxReembedChange,
  analysis = null,
  className = '',
}) {
  const displaySrc = previewSrc || optimizedPreviewSrc
  const hasArtwork = Boolean(displaySrc)
  const showOptimizedCompare =
    optimizeArtwork && optimizedPreviewSrc && previewSrc && optimizedPreviewSrc !== previewSrc

  return (
    <section className={`rounded-xl border border-white/10 bg-surface-raised p-4 ${className}`}>
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-medium">Artwork</h3>
        {showOptimizedCompare && (
          <span className="rounded bg-accent/15 px-2 py-0.5 text-[10px] font-medium text-accent">Optimize on save</span>
        )}
      </div>

      <ArtworkPreview
        src={displaySrc}
        className="mt-3 flex aspect-square items-center justify-center rounded-lg bg-black/30"
      />

      {showOptimizedCompare && (
        <div className="mt-3 rounded-lg border border-white/10 bg-black/20 p-2">
          <p className="text-[10px] font-medium uppercase tracking-wide text-white/45">After optimize (save preview)</p>
          <ArtworkPreview
            src={optimizedPreviewSrc}
            className="mt-2 flex aspect-square max-h-28 items-center justify-center rounded-md bg-black/30"
          />
        </div>
      )}

      <div className="mt-3 flex flex-col gap-2">
        <button
          type="button"
          onClick={onReplace}
          disabled={disabled}
          className="rounded-lg border border-white/15 px-3 py-2 text-xs transition-colors duration-200 ease-apple hover:bg-white/5 disabled:opacity-50"
        >
          {hasArtwork ? 'Replace artwork' : 'Add artwork'}
        </button>
        {hasArtwork && onRemove && (
          <button
            type="button"
            disabled={disabled}
            onClick={onRemove}
            className="rounded-lg border border-white/15 px-3 py-2 text-xs text-white/70 transition-colors duration-200 ease-apple hover:bg-white/5 disabled:opacity-50"
          >
            Remove artwork
          </button>
        )}
      </div>

      <div className="mt-3 space-y-2 border-t border-white/10 pt-3">
        <label className="flex cursor-pointer items-start gap-2 text-xs text-white/70">
          <input
            type="checkbox"
            className="mt-0.5"
            checked={optimizeArtwork}
            onChange={(e) => onOptimizeArtworkChange?.(e.target.checked)}
            disabled={disabled}
          />
          <span>
            <span className="font-medium text-white/85">Optimize for Apple Music album view</span>
            <span className="mt-0.5 block text-white/45">
              Square crop, trim letterbox bars, mild saturation boost, baseline JPEG — improves iOS accent backgrounds.
            </span>
          </span>
        </label>

        <label className="flex cursor-pointer items-start gap-2 text-xs text-white/70">
          <input
            type="checkbox"
            className="mt-0.5"
            checked={mp4boxReembed}
            onChange={(e) => onMp4boxReembedChange?.(e.target.checked)}
            disabled={disabled}
          />
          <span>
            <span className="font-medium text-white/85">Re-embed via MP4Box after save</span>
            <span className="mt-0.5 block text-white/45">
              Optional. Helps stubborn Windows → iPhone sync; requires MP4Box on PATH or in tools folder.
            </span>
          </span>
        </label>

        {showFolderCover && onUseFolderCover && (
          <button
            type="button"
            disabled={disabled}
            onClick={onUseFolderCover}
            className="w-full rounded-lg border border-white/15 px-3 py-2 text-xs hover:bg-white/5 disabled:opacity-50"
          >
            Use album folder cover (cover.jpg)
          </button>
        )}
      </div>

      {analysis?.warnings?.length > 0 && (
        <ul className="mt-3 space-y-1 rounded-lg border border-yellow-500/25 bg-yellow-500/10 px-3 py-2 text-[11px] leading-relaxed text-yellow-100/90">
          {analysis.warnings.map((w) => (
            <li key={w}>• {w}</li>
          ))}
        </ul>
      )}
      {analysis?.accent_ready && (
        <p className="mt-3 rounded-lg border border-green-500/25 bg-green-500/10 px-3 py-2 text-[11px] text-green-100/90">
          {analysis.summary}
        </p>
      )}
    </section>
  )
}
