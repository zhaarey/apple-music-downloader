export const YOUTUBE_DELIVERY_MODES = [
  {
    id: 'audio',
    label: 'Audio only',
    shortLabel: 'Audio',
    desc: 'AAC 256 kbps · tagged for Apple Music',
  },
  {
    id: 'both',
    label: 'Audio + video',
    shortLabel: 'Both',
    desc: 'AAC plus H.264 MP4 · plays in iPhone Music app',
  },
  {
    id: 'video',
    label: 'Video only',
    shortLabel: 'Video',
    desc: 'H.264 MP4 with embedded AAC stereo',
  },
]

export function youtubeDownloadButtonLabel(mode, count) {
  const n = count ?? 0
  const suffix = n > 0 ? ` (${n})` : ''
  switch (mode) {
    case 'both':
      return `Download audio + MP4${suffix}`
    case 'video':
      return `Download MP4 video${suffix}`
    default:
      return `Download audio${suffix}`
  }
}

export function youtubeDeliveryIncludesAudio(mode) {
  return mode === 'audio' || mode === 'both'
}

export function youtubeDeliveryIncludesVideo(mode) {
  return mode === 'video' || mode === 'both'
}
