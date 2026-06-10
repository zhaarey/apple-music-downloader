import { useState } from 'react'
import { RunAppleMusicDeepPurge, OpenApplePurgeLog } from '../wailsjs/go/main/App'
import { useConfirm } from '../hooks/useConfirm'

export default function ApplePurgeSettingsPanel() {
  const { requestConfirm, ConfirmDialogSlot } = useConfirm()
  const [busy, setBusy] = useState('')
  const [result, setResult] = useState(null)

  const runPurge = async (elevated) => {
    setBusy(elevated ? 'admin' : 'purge')
    setResult(null)
    try {
      const res = await RunAppleMusicDeepPurge(elevated)
      setResult(res)
    } catch (e) {
      setResult({
        ok: false,
        summary: 'Deep purge failed',
        message: String(e?.message || e),
      })
    } finally {
      setBusy('')
    }
  }

  const confirmAndRun = async (elevated) => {
    const confirmed = await requestConfirm({
      title: elevated ? 'Run deep purge as administrator?' : 'Deep purge Apple Music PC caches?',
      message: elevated
        ? 'Administrator mode restarts the Apple Mobile Device Service (USB / iPhone sync). UAC will ask for approval. Apple Music will be closed first.'
        : 'This closes Apple Music, clears PC artwork caches (UWP app, library bundle art, and legacy iTunes art folders), and tries to restart the mobile device service. Your downloaded .m4a files are not deleted. iPhone artwork is not cleared — fix that on the device separately.',
      confirmLabel: elevated ? 'Run as administrator' : 'Run deep purge',
    })
    if (!confirmed) return
    await runPurge(elevated)
  }

  return (
    <section className="mt-4 space-y-3 rounded-xl border border-amber-500/20 bg-amber-500/[0.04] p-4">
      <div>
        <h3 className="font-medium text-white/90">Apple Music PC cache purge</h3>
        <p className="mt-1 text-xs leading-relaxed text-white/50">
          Use when Apple Music on this PC keeps showing wrong album art after you fixed tags, or when sync repair’s lighter
          cache clear is not enough. This is stronger than Tag Editor → Clear PC caches.
        </p>
      </div>

      <ul className="list-inside list-disc space-y-1 text-[11px] leading-relaxed text-white/45">
        <li>Quits Apple Music and related PC processes automatically.</li>
        <li>Clears UWP app caches, Apple Music library bundle artwork, and legacy iTunes artwork folders.</li>
        <li>Does not delete your download folders or .m4a files.</li>
        <li>Does not reset iPhone artwork — delete affected albums on the phone, then re-sync from the PC.</li>
        <li>Afterward: re-open Apple Music and re-import your albums so embedded art is picked up fresh.</li>
      </ul>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={!!busy}
          onClick={() => void confirmAndRun(false)}
          className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm font-medium text-amber-100 hover:bg-amber-500/15 disabled:opacity-40"
        >
          {busy === 'purge' ? 'Purging…' : 'Deep purge PC caches'}
        </button>
        <button
          type="button"
          disabled={!!busy}
          onClick={() => void confirmAndRun(true)}
          className="rounded-lg border border-white/15 px-4 py-2 text-sm text-white/80 hover:bg-white/5 disabled:opacity-40"
        >
          {busy === 'admin' ? 'Waiting for UAC…' : 'Run as administrator'}
        </button>
        <button
          type="button"
          onClick={() => OpenApplePurgeLog()}
          className="rounded-lg border border-white/10 px-3 py-2 text-xs text-white/50 hover:bg-white/5"
        >
          Open purge log
        </button>
      </div>

      {result && (
        <div
          className={`rounded-lg border px-3 py-2 text-sm ${
            result.ok
              ? 'border-green-500/30 bg-green-500/10 text-green-100'
              : 'border-yellow-500/30 bg-yellow-500/10 text-yellow-100'
          }`}
        >
          <p className="font-medium">{result.summary || result.message}</p>
          {result.message && result.message !== result.summary && (
            <p className="mt-1 text-xs opacity-90">{result.message}</p>
          )}
          {result.need_elevated && !result.ok && (
            <p className="mt-2 text-xs opacity-90">
              Caches may still have cleared — use Run as administrator if you sync an iPhone over USB.
            </p>
          )}
          {result.manual_steps?.length > 0 && (
            <ol className="mt-2 list-inside list-decimal space-y-0.5 text-xs opacity-90">
              {result.manual_steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          )}
        </div>
      )}

      {ConfirmDialogSlot}
    </section>
  )
}
