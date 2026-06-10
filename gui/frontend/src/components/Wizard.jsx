import { useState } from 'react'

const STOREFRONTS = ['us', 'jp', 'gb', 'ca', 'de', 'fr', 'au', 'kr', 'cn', 'tw', 'hk', 'sg', 'in', 'br', 'mx']

export default function Wizard({ settings, deps, onComplete, onRefreshDeps }) {
  const [step, setStep] = useState(0)
  const [cfg, setCfg] = useState(
    settings || {
      'media-user-token': '',
      storefront: 'us',
      'aac-save-folder': '',
      'embed-lrc': true,
      'embed-cover': true,
    },
  )

  const steps = ['Welcome', 'Account', 'Output', 'Dependencies']

  const update = (key, val) => setCfg((c) => ({ ...c, [key]: val }))

  const next = () => {
    if (step < steps.length - 1) setStep(step + 1)
    else onComplete(cfg)
  }

  return (
    <div className="flex h-screen items-center justify-center bg-surface p-8">
      <div className="w-full max-w-lg rounded-2xl border border-white/10 bg-surface-raised p-8 shadow-2xl">
        <div className="mb-6 flex gap-2">
          {steps.map((s, i) => (
            <div
              key={s}
              className={`h-1 flex-1 rounded-full ${i <= step ? 'bg-accent' : 'bg-white/10'}`}
            />
          ))}
        </div>

        {step === 0 && (
          <div className="space-y-4">
            <h2 className="text-2xl font-bold">Welcome to Aura</h2>
            <p className="text-white/70 leading-relaxed">
              Aura downloads Apple Music as AAC, pulls YouTube DJ sets, and includes a tag editor for your library.
            </p>
            <p className="text-sm text-white/50">Apple Music downloads require an active subscription.</p>
          </div>
        )}

        {step === 1 && (
          <div className="space-y-4">
            <h2 className="text-2xl font-bold">Apple Music account</h2>
            <label className="block text-sm text-white/60">Storefront (2-letter country code)</label>
            <select
              value={cfg.storefront || 'us'}
              onChange={(e) => update('storefront', e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-surface px-3 py-2"
            >
              {STOREFRONTS.map((s) => (
                <option key={s} value={s}>
                  {s.toUpperCase()}
                </option>
              ))}
            </select>
            <label className="block text-sm text-white/60">media-user-token (required for AAC downloads)</label>
            <textarea
              value={cfg['media-user-token'] || ''}
              onChange={(e) => update('media-user-token', e.target.value)}
              placeholder="Paste from music.apple.com cookies (DevTools → Application → Cookies)"
              rows={4}
              className="w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm font-mono"
            />
            <p className="text-xs text-white/40">
              Open music.apple.com → F12 → Application → Cookies → copy <code className="text-accent">media-user-token</code>
            </p>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-4">
            <h2 className="text-2xl font-bold">Output folder</h2>
            <label className="block text-sm text-white/60">AAC / default downloads</label>
            <input
              value={cfg['aac-save-folder'] || ''}
              onChange={(e) => update('aac-save-folder', e.target.value)}
              placeholder="C:\Users\You\Music\Apple Music Downloads"
              className="w-full rounded-lg border border-white/10 bg-surface px-3 py-2"
            />
          </div>
        )}

        {step === 3 && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-2xl font-bold">Dependencies</h2>
              <button onClick={onRefreshDeps} className="text-sm text-accent hover:underline">
                Refresh
              </button>
            </div>
            <ul className="space-y-2">
              {(deps || []).map((d) => (
                <li key={d.name} className="flex items-center justify-between rounded-lg bg-surface px-3 py-2">
                  <span>{d.name}</span>
                  <span className={d.ok ? 'text-green-400' : d.required ? 'text-red-400' : 'text-yellow-400'}>
                    {d.ok ? '✓ Ready' : d.required ? '✗ Missing' : 'Optional'}
                  </span>
                </li>
              ))}
            </ul>
            <p className="text-xs text-white/40">
              MP4Box is bundled with the installer for tagging AAC downloads.
            </p>
          </div>
        )}

        <div className="mt-8 flex justify-between">
          <button
            onClick={() => (step > 0 ? setStep(step - 1) : null)}
            disabled={step === 0}
            className="rounded-lg px-4 py-2 text-white/60 disabled:opacity-30 hover:text-white"
          >
            Back
          </button>
          <button onClick={next} className="rounded-lg bg-accent px-6 py-2 font-medium hover:bg-accent-muted">
            {step === steps.length - 1 ? 'Get started' : 'Continue'}
          </button>
        </div>
      </div>
    </div>
  )
}
