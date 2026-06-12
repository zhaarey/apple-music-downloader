export function prepareAlbumConfirmDetails(preview) {
  if (!preview) return []
  const lines = [
    `${preview.track_count ?? 0} track(s) may be updated`,
    preview.recursive
      ? 'Includes .m4a files in this folder and all subfolders'
      : 'Only .m4a files directly in this folder',
    'Title, artist, and track numbers are not changed — artwork only',
    'Tracks that already have matching JPEG art are skipped',
  ]
  if (preview.cover_source) {
    lines.push(`Artwork source: ${preview.cover_source}`)
  }
  lines.push('Every updated track gets one shared normalized JPEG cover')
  return lines
}

export function syncRepairConfirmDetails(preview) {
  if (!preview) return []
  const folders = preview.folders || []
  const lines = [
    `${preview.track_count ?? 0} track(s) across ${folders.length} library folder(s) may be updated`,
    'Artwork only — text metadata is preserved; matching tracks are skipped',
    'PC Apple Music artwork caches are cleared afterward (not your iPhone)',
    'Remove and re-import affected albums in Apple Music; delete them on iPhone before re-syncing',
  ]
  if (folders.length > 0 && folders.length <= 4) {
    folders.forEach((f) => lines.push(f))
  } else if (folders.length > 4) {
    lines.push(`${folders[0]} … and ${folders.length - 1} more folder(s)`)
  }
  return lines
}

export function bulkSaveArtworkConfirmDetails(trackCount, hasNewCover, clearArtwork) {
  const lines = [`${trackCount} track(s) will be saved`]
  if (clearArtwork) {
    lines.push('Embedded artwork will be removed from every track')
  } else if (hasNewCover) {
    lines.push('The selected artwork will be embedded on every track')
  }
  return lines
}

/** Runs preview + confirm + PrepareAlbumForSync for a folder. Returns result or null if cancelled. */
export async function confirmAndPrepareAlbum({ requestConfirm, folder, PreviewPrepareAlbumForSync, PrepareAlbumForSync }) {
  const preview = await PreviewPrepareAlbumForSync(folder)
  if (!preview?.track_count) {
    return { cancelled: false, error: 'No .m4a files found directly in that folder.', folder }
  }
  const confirmed = await requestConfirm({
    title: 'Update artwork on album tracks?',
    message:
      preview.warning ||
      'Only embedded artwork will change. Title, artist, and track numbers stay as they are.',
    details: [...prepareAlbumConfirmDetails(preview), folder],
    confirmLabel: `Update artwork on ${preview.track_count} track(s)`,
  })
  if (!confirmed) {
    return { cancelled: true }
  }
  const result = await PrepareAlbumForSync(folder)
  return { cancelled: false, result, folder, preview }
}
