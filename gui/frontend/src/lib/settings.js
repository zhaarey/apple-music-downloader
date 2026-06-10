import { SaveSettings } from '../wailsjs/go/main/App'

export function outputFolderKey(quality, youtubeMode) {
  if (youtubeMode) return 'youtube-save-folder'
  if (quality === 'alac') return 'alac-save-folder'
  if (quality === 'atmos') return 'atmos-save-folder'
  return 'aac-save-folder'
}

export function outputFolderLabel(quality, youtubeMode) {
  if (youtubeMode) return 'YouTube download folder'
  if (quality === 'alac') return 'Lossless download folder'
  if (quality === 'atmos') return 'Dolby Atmos download folder'
  return 'Download folder'
}

export function outputFolderPath(settings, quality, youtubeMode) {
  const key = outputFolderKey(quality, youtubeMode)
  const primary = String(settings?.[key] || '').trim()
  if (primary) return primary
  if (!youtubeMode && key !== 'aac-save-folder') {
    return String(settings?.['aac-save-folder'] || '').trim()
  }
  return ''
}

/** Merge patch into current settings and persist via Wails. */
export async function persistSettings(current, patch) {
  const merged = { ...(current || {}), ...(patch || {}) }
  await SaveSettings(merged)
  return merged
}
