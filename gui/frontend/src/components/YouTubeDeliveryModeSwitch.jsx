import { YOUTUBE_DELIVERY_MODES } from '../lib/youtubeDelivery'

export default function YouTubeDeliveryModeSwitch({ value, onChange, disabled }) {
  const active = YOUTUBE_DELIVERY_MODES.find((m) => m.id === value) || YOUTUBE_DELIVERY_MODES[0]

  return (
    <div className="rounded-xl border border-white/10 bg-surface-raised p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-medium text-white/90">Save as</p>
          <p className="mt-0.5 text-xs text-white/50">{active.desc}</p>
        </div>
        <div
          className="flex w-full shrink-0 rounded-lg bg-black/30 p-1 ring-1 ring-white/10 sm:w-auto"
          role="radiogroup"
          aria-label="YouTube download format"
        >
          {YOUTUBE_DELIVERY_MODES.map((mode) => {
            const isActive = value === mode.id
            return (
              <button
                key={mode.id}
                type="button"
                role="radio"
                aria-checked={isActive}
                title={mode.desc}
                disabled={disabled}
                onClick={() => onChange(mode.id)}
                className={`min-w-0 flex-1 rounded-md px-3 py-2.5 text-sm font-medium transition sm:flex-none sm:min-w-[5.5rem] sm:px-4 ${
                  isActive
                    ? 'bg-accent text-white shadow-sm'
                    : 'text-white/60 hover:text-white disabled:opacity-40'
                }`}
              >
                <span className="block">{mode.shortLabel}</span>
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
