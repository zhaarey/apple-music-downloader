const PHASES = {
  single: [
    { id: 'link', label: 'Link' },
    { id: 'review', label: 'Review' },
    { id: 'downloading', label: 'Downloading' },
    { id: 'done', label: 'Done' },
  ],
  bulk: [
    { id: 'link', label: 'Paste & add' },
    { id: 'review', label: 'Review queue' },
    { id: 'downloading', label: 'Downloading' },
    { id: 'done', label: 'Done' },
  ],
  youtube: [
    { id: 'link', label: 'Link' },
    { id: 'review', label: 'Review' },
    { id: 'downloading', label: 'Downloading' },
    { id: 'done', label: 'Done' },
  ],
}

function phaseIndex(phases, current) {
  return phases.findIndex((p) => p.id === current)
}

export default function DownloadPhaseStepper({ phase, variant = 'single' }) {
  const phases = PHASES[variant] || PHASES.single
  const currentIdx = phaseIndex(phases, phase)

  return (
    <nav aria-label="Download progress" className="flex items-center gap-1 sm:gap-2">
      {phases.map((step, i) => {
        const done = i < currentIdx
        const active = i === currentIdx
        return (
          <div key={step.id} className="flex min-w-0 items-center gap-1 sm:gap-2">
            {i > 0 && (
              <span
                className={`hidden h-px w-4 shrink-0 sm:block sm:w-6 ${done || active ? 'bg-accent/50' : 'bg-white/10'}`}
                aria-hidden
              />
            )}
            <div
              className={`flex min-w-0 items-center gap-1.5 rounded-full px-2 py-1 text-xs font-medium sm:px-3 ${
                active
                  ? 'bg-accent/20 text-accent ring-1 ring-accent/40'
                  : done
                    ? 'text-green-400/90'
                    : 'text-white/35'
              }`}
            >
              <span
                className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[10px] ${
                  active
                    ? 'bg-accent text-white'
                    : done
                      ? 'bg-green-500/20 text-green-400'
                      : 'bg-white/5 text-white/40'
                }`}
                aria-hidden
              >
                {done ? '✓' : i + 1}
              </span>
              <span className="truncate">{step.label}</span>
            </div>
          </div>
        )
      })}
    </nav>
  )
}
