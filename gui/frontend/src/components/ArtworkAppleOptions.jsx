import ArtworkPreview from './ArtworkPreview'

/**
 * Artwork panel with Apple Music iOS album-view optimization controls.
 * Guides users to pick an image in the GUI and embed it into the track on save
 * (fixes "folder art only" sync issues — cover.jpg is optional, not required).
 */
export default function ArtworkAppleOptions({
  previewSrc = '',
  optimizedPreviewSrc = '',
  onReplace,
  onRemove,
  onUseFolderCover,
  onEmbedAndSave,
  onClearPending,
  showFolderCover = false,
  folderCoverAvailable = false,
  folderCoverName = 'cover.jpg',
  hasEmbeddedArtwork = false,
  forceMissingArtwork = false,
  pendingCoverPath = '',
  embedding = false,
  disabled = false,
  optimizeArtwork = false,
  onOptimizeArtworkChange,
  mp4boxReembed = false,
  onMp4boxReembedChange,
  showMp4boxReembed = true,
  analysis = null,
  onApplyOptimization,
  applyingOptimization = false,
  applyOptimizationLabel = 'Apply Apple Music optimization',
  className = '',
}) {
  const displaySrc = previewSrc || optimizedPreviewSrc
  const hasArtwork = Boolean(displaySrc)
  const pendingEmbed = Boolean(pendingCoverPath) && !hasEmbeddedArtwork
  const replacing = Boolean(pendingCoverPath) && hasEmbeddedArtwork
  const missingEmbedded =
    Boolean(forceMissingArtwork) || (!hasEmbeddedArtwork && !pendingCoverPath && !hasArtwork)
  const showOptimizedCompare =
    optimizeArtwork && optimizedPreviewSrc && previewSrc && optimizedPreviewSrc !== previewSrc
  const showApplyButton = Boolean(onApplyOptimization && hasArtwork && !analysis?.accent_ready)

  let status = null
  if (pendingEmbed || replacing) {
    status = {
      tone: 'accent',
      title: replacing ? 'New cover ready' : 'Ready to embed',
      detail: 'Click Save to write this image into the file tags. iPhone sync needs art inside the track — folder art alone is not enough.',
    }
  } else if (hasEmbeddedArtwork) {
    status = {
      tone: 'ok',
      title: 'Embedded in file',
      detail: 'Cover is stored in the track tags — ready for Apple Music / iPhone sync.',
    }
  } else if (missingEmbedded) {
    status = {
      tone: 'warn',
      title: forceMissingArtwork ? 'Tracks missing embedded cover' : 'No cover in this file',
      detail: forceMissingArtwork
        ? 'Choose an image below, then Save all to embed it on every track. Folder art alone will not sync to iPhone.'
        : 'PC Apple Music may still show folder art. Choose an image below, then Save to embed it into this track.',
    }
  }

  const toneClass =
    status?.tone === 'ok'
      ? 'border-green-500/25 bg-green-500/10 text-green-100/90'
      : status?.tone === 'warn'
        ? 'border-yellow-500/30 bg-yellow-500/10 text-yellow-100/90'
        : status?.tone === 'accent'
          ? 'border-accent/35 bg-accent/10 text-sky-100/90'
          : ''

  return (
    <section className={`rounded-xl border border-white/10 bg-surface-raised p-4 ${className}`}>
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-medium">Artwork</h3>
          <p className="mt-0.5 text-[11px] leading-relaxed text-white/45">
            Pick an image here and Save — embeds the cover into the song file.
          </p>
        </div>
        {showOptimizedCompare && (
          <span className="rounded bg-accent/15 px-2 py-0.5 text-[10px] font-medium text-accent">Optimize on save</span>
        )}
      </div>

      <div
        className={`mt-3 overflow-hidden rounded-lg ${
          missingEmbedded ? 'ring-1 ring-yellow-500/35 ring-offset-0' : ''
        }`}
      >
        <ArtworkPreview
          src={displaySrc}
          emptyLabel={missingEmbedded ? 'No cover embedded' : 'No artwork'}
          className={`flex aspect-square items-center justify-center bg-black/30 ${
            missingEmbedded ? 'border border-dashed border-yellow-500/30' : ''
          }`}
        />
      </div>

      {status && (
        <div className={`mt-3 rounded-lg border px-3 py-2.5 text-[11px] leading-relaxed ${toneClass}`}>
          <p className="font-medium">{status.title}</p>
          <p className="mt-0.5 opacity-90">{status.detail}</p>
        </div>
      )}

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
        {showApplyButton && (
          <button
            type="button"
            disabled={disabled || applyingOptimization || embedding}
            onClick={onApplyOptimization}
            className="rounded-lg bg-accent px-3 py-2 text-xs font-medium text-white hover:bg-accent-muted disabled:opacity-50"
          >
            {applyingOptimization ? 'Optimizing…' : applyOptimizationLabel}
          </button>
        )}

        {missingEmbedded && (
          <>
            <button
              type="button"
              onClick={onReplace}
              disabled={disabled || embedding}
              className="rounded-lg bg-accent px-3 py-2.5 text-xs font-semibold text-white hover:bg-accent/90 disabled:opacity-50"
            >
              Choose artwork image…
            </button>
            {showFolderCover && onUseFolderCover && folderCoverAvailable && (
              <button
                type="button"
                disabled={disabled || embedding}
                onClick={onUseFolderCover}
                className="rounded-lg border border-white/15 px-3 py-2 text-xs hover:bg-white/5 disabled:opacity-50"
              >
                Use album folder {folderCoverName}
              </button>
            )}
            <p className="text-[11px] leading-relaxed text-white/40">
              Choose any JPG or PNG. On Save it embeds into this file
              {showFolderCover ? ' and writes a folder cover for Apple Music on PC' : ''}.
            </p>
          </>
        )}

        {!missingEmbedded && (
          <>
            <button
              type="button"
              onClick={onReplace}
              disabled={disabled || embedding}
              className="rounded-lg border border-white/15 px-3 py-2 text-xs transition-colors duration-200 ease-apple hover:bg-white/5 disabled:opacity-50"
            >
              {hasArtwork ? 'Replace artwork' : 'Add artwork'}
            </button>
            {(pendingEmbed || replacing) && onEmbedAndSave && (
              <button
                type="button"
                disabled={disabled || embedding}
                onClick={onEmbedAndSave}
                className="rounded-lg bg-accent px-3 py-2.5 text-xs font-semibold text-white hover:bg-accent/90 disabled:opacity-50"
              >
                {embedding ? 'Saving…' : 'Save & embed into file'}
              </button>
            )}
            {(pendingEmbed || replacing) && onClearPending && (
              <button
                type="button"
                disabled={disabled || embedding}
                onClick={onClearPending}
                className="rounded-lg border border-white/15 px-3 py-2 text-xs text-white/70 hover:bg-white/5 disabled:opacity-50"
              >
                Cancel pending artwork
              </button>
            )}
            {hasArtwork && !pendingCoverPath && onRemove && (
              <button
                type="button"
                disabled={disabled || embedding}
                onClick={onRemove}
                className="rounded-lg border border-white/15 px-3 py-2 text-xs text-white/70 transition-colors duration-200 ease-apple hover:bg-white/5 disabled:opacity-50"
              >
                Remove artwork
              </button>
            )}
          </>
        )}
      </div>

      <div className="mt-3 space-y-2 border-t border-white/10 pt-3">
        <label className="flex cursor-pointer items-start gap-2 text-xs text-white/70">
          <input
            type="checkbox"
            className="mt-0.5"
            checked={optimizeArtwork}
            onChange={(e) => onOptimizeArtworkChange?.(e.target.checked)}
            disabled={disabled || embedding}
          />
          <span>
            <span className="font-medium text-white/85">Optimize for Apple Music album view</span>
            <span className="mt-0.5 block text-white/45">
              Optional. Square crop and JPEG re-encode for iOS accent backgrounds — leaves artwork unchanged when off.
            </span>
          </span>
        </label>

        {showMp4boxReembed && (
          <label className="flex cursor-pointer items-start gap-2 text-xs text-white/70">
            <input
              type="checkbox"
              className="mt-0.5"
              checked={mp4boxReembed}
              onChange={(e) => onMp4boxReembedChange?.(e.target.checked)}
              disabled={disabled || embedding}
            />
            <span>
              <span className="font-medium text-white/85">Re-embed via MP4Box after save</span>
              <span className="mt-0.5 block text-white/45">
                Optional. Helps stubborn Windows → iPhone sync; requires MP4Box on PATH or in tools folder.
              </span>
            </span>
          </label>
        )}

        {showFolderCover && onUseFolderCover && folderCoverAvailable && (hasArtwork || pendingCoverPath) && (
          <button
            type="button"
            disabled={disabled || embedding}
            onClick={onUseFolderCover}
            className="w-full rounded-lg border border-white/15 px-3 py-2 text-xs hover:bg-white/5 disabled:opacity-50"
          >
            Use album folder {folderCoverName}
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
