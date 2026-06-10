function formatCodec(name) {
  if (!name) return '—'
  const n = String(name).toLowerCase()
  if (n === 'h264' || n.startsWith('avc')) return 'H.264'
  if (n === 'aac') return 'AAC'
  return name
}

function StatusPill({ ok, label }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${
        ok ? 'bg-green-500/15 text-green-200' : 'bg-amber-500/15 text-amber-100'
      }`}
    >
      {label}
    </span>
  )
}

export default function VideoTagInfoPanel({ fileInfo }) {
  if (!fileInfo || fileInfo.media_kind !== 'video') return null

  const resolution =
    fileInfo.video_width > 0 && fileInfo.video_height > 0
      ? `${fileInfo.video_width}×${fileInfo.video_height}`
      : '—'

  return (
    <section className="rounded-xl border border-violet-500/20 bg-violet-500/5 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-medium">Music video</h3>
            <StatusPill
              ok={fileInfo.apple_video_ready}
              label={fileInfo.apple_video_ready ? 'Apple Music ready' : 'Needs conversion'}
            />
          </div>
          <p className="mt-1 text-xs text-white/50">
            MP4 with H.264 video and AAC stereo audio · plays offline in the iPhone Music app as a music video
          </p>
        </div>
      </div>
      <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <dt className="text-[11px] uppercase tracking-wide text-white/40">Video</dt>
          <dd className="mt-0.5 font-medium text-white/90">{formatCodec(fileInfo.video_codec)}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wide text-white/40">Audio</dt>
          <dd className="mt-0.5 font-medium text-white/90">{formatCodec(fileInfo.audio_codec)}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wide text-white/40">Resolution</dt>
          <dd className="mt-0.5 font-medium text-white/90">{resolution}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wide text-white/40">Duration</dt>
          <dd className="mt-0.5 font-medium text-white/90">{fileInfo.duration_label || '—'}</dd>
        </div>
      </dl>
      {!fileInfo.apple_video_ready && fileInfo.apple_video_detail && (
        <p className="mt-3 rounded-lg border border-amber-500/25 bg-amber-500/10 px-3 py-2 text-xs text-amber-100">
          {fileInfo.apple_video_detail}
        </p>
      )}
    </section>
  )
}
