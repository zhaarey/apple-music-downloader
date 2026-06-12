/** @typedef {'separate' | 'apple-music'} YouTubeOutputLocation */

export const YOUTUBE_OUTPUT_LOCATIONS = [
  {
    id: 'separate',
    shortLabel: 'Separate folder',
    desc: 'Keep YouTube videos and audio in their own folder, separate from Apple Music downloads.',
  },
  {
    id: 'apple-music',
    shortLabel: 'Apple Music folder',
    desc: 'Save YouTube downloads to the same folder as your Apple Music library (AAC / default path).',
  },
]

/** @returns {YouTubeOutputLocation} */
export function youtubeOutputLocation(settings) {
  return settings?.['youtube-output-location'] === 'apple-music' ? 'apple-music' : 'separate'
}

export function resolveYouTubeOutputPath(settings) {
  if (youtubeOutputLocation(settings) === 'apple-music') {
    return String(settings?.['aac-save-folder'] || '').trim()
  }
  const separate = String(settings?.['youtube-save-folder'] || '').trim()
  if (separate) return separate
  return String(settings?.['aac-save-folder'] || '').trim()
}

export function youtubeOutputFolderLabel(settings) {
  return youtubeOutputLocation(settings) === 'apple-music'
    ? 'Output folder (Apple Music)'
    : 'YouTube download folder'
}

export function youtubeOutputFolderHint(settings) {
  return youtubeOutputLocation(settings) === 'apple-music'
    ? 'Using your Apple Music download folder. Change the AAC path under Settings → Output folders.'
    : 'Videos and audio save here. Also configurable in Settings → YouTube.'
}
