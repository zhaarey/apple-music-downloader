import DownloadPhaseStepper from './DownloadPhaseStepper'

export default function DownloadFlowLayout({
  title,
  subtitle,
  phase,
  stepperVariant = 'single',
  backAction,
  children,
  footer,
  footerNote,
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="shrink-0 space-y-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            {backAction && (
              <div className="mb-2">{backAction}</div>
            )}
            {title && <h2 className="text-xl font-semibold">{title}</h2>}
            {subtitle && <p className="mt-1 text-sm text-white/50">{subtitle}</p>}
          </div>
          <div className="w-full sm:w-auto sm:max-w-[min(100%,36rem)]">
            <DownloadPhaseStepper phase={phase} variant={stepperVariant} />
          </div>
        </div>
      </div>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto">{children}</div>

      {footer && (
        <div className="sticky bottom-0 shrink-0 border-t border-white/10 bg-surface pt-4">
          {footer}
          {footerNote && <p className="mt-2 text-center text-xs text-white/40">{footerNote}</p>}
        </div>
      )}
    </div>
  )
}
