export default function SpotifyTrackNotice({ compact = false }) {
  return (
    <div className="rounded-xl border border-[#1DB954]/20 bg-[#1DB954]/[0.06] px-4 py-3">
      <p className="text-sm font-medium text-emerald-50">Spotify: one track at a time</p>
      <p className="mt-1 text-xs text-emerald-100/70">
        Paste individual <strong className="font-medium text-emerald-100/90">open.spotify.com/track/…</strong> links
        and we&apos;ll search for the match on Apple Music. Playlists and albums are not supported — use an{' '}
        <strong className="font-medium text-emerald-100/90">Apple Music playlist</strong> link to queue many songs
        with track checkboxes.
      </p>
      {!compact && (
        <p className="mt-2 text-[11px] text-emerald-100/55">
          Matching uses the track title and artist from Spotify — no API keys or Spotify account setup required.
        </p>
      )}
    </div>
  )
}
