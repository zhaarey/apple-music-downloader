import { SaveSettings } from '../wailsjs/go/main/App'
import { resolveYouTubeOutputPath, youtubeOutputLocation } from './youtubeOutput'

export { youtubeOutputLocation, resolveYouTubeOutputPath }

export function outputFolderKey(quality, youtubeMode, settings) {
  if (youtubeMode) {
    return youtubeOutputLocation(settings) === 'apple-music' ? 'aac-save-folder' : 'youtube-save-folder'
  }
  if (quality === 'alac') return 'alac-save-folder'
  if (quality === 'atmos') return 'atmos-save-folder'
  return 'aac-save-folder'
}

export function outputFolderLabel(quality, youtubeMode, settings) {
  if (youtubeMode) {
    return youtubeOutputLocation(settings) === 'apple-music'
      ? 'Output folder (Apple Music)'
      : 'YouTube download folder'
  }
  if (quality === 'alac') return 'Lossless download folder'
  if (quality === 'atmos') return 'Dolby Atmos download folder'
  return 'Download folder'
}

export function outputFolderPath(settings, quality, youtubeMode) {
  if (youtubeMode) {
    return resolveYouTubeOutputPath(settings)
  }
  const key = outputFolderKey(quality, youtubeMode, settings)
  const primary = String(settings?.[key] || '').trim()
  if (primary) return primary
  if (key !== 'aac-save-folder') {
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
