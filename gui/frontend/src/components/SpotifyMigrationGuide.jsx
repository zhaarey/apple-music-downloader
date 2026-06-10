import { hasSpotifyCredentials, MIGRATION_STEPS } from '../lib/spotifyMigration'

export default function SpotifyMigrationGuide({ settings, onOpenSettings, compact = false }) {
  const ready = hasSpotifyCredentials(settings)

  return (
    <div className="rounded-xl border border-[#1DB954]/20 bg-[#1DB954]/[0.06] p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-emerald-50">Migrate a Spotify playlist</p>
          <p className="mt-0.5 text-xs text-emerald-100/70">
            Public playlist → Apple Music matches → download to your library folder.
          </p>
        </div>
        <span
          className={`shrink-0 rounded-full border px-2.5 py-0.5 text-[10px] font-medium uppercase tracking-wide ${
            ready
              ? 'border-emerald-500/40 bg-emerald-500/15 text-emerald-200'
              : 'border-amber-500/40 bg-amber-500/10 text-amber-200'
          }`}
        >
          {ready ? 'API ready' : 'Setup needed'}
        </span>
      </div>

      {!compact && (
        <ol className="mt-3 space-y-2.5">
          {MIGRATION_STEPS.map((step, i) => (
            <li key={step.title} className="flex gap-3 text-xs">
              <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-black/25 text-[10px] font-semibold text-emerald-200">
                {i + 1}
              </span>
              <div>
                <p className="font-medium text-emerald-50/95">{step.title}</p>
                <p className="mt-0.5 text-emerald-100/60">{step.detail}</p>
              </div>
            </li>
          ))}
        </ol>
      )}

      {!ready && onOpenSettings && (
        <button
          type="button"
          onClick={onOpenSettings}
          className="mt-3 w-full rounded-lg border border-[#1DB954]/30 bg-black/20 py-2 text-xs font-medium text-emerald-100 hover:bg-black/30"
        >
          Open Settings → Spotify matching
        </button>
      )}
    </div>
  )
}
