import { useEffect, useState } from 'react'

export default function SettingsTab({ settings, deps, onSave, onPickFolder, onRefreshDeps, onShowWizard }) {
  const [cfg, setCfg] = useState(settings || {})
  const [saved, setSaved] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)

  useEffect(() => {
    if (settings) setCfg(settings)
  }, [settings])

  const update = (key, val) => setCfg((c) => ({ ...c, [key]: val }))

  const save = async () => {
    await onSave(cfg)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const pickFolder = async (key) => {
    const path = await onPickFolder()
    if (path) update(key, path)
  }

  return (
    <div className="mx-auto h-full max-w-2xl overflow-y-auto">
      <h2 className="text-xl font-semibold">Settings</h2>
      <p className="mt-1 text-sm text-white/50">Config is saved to your AppData folder</p>

      <section className="mt-6 space-y-4 rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">Account</h3>
        <label className="block text-xs text-white/50">Storefront</label>
        <input
          value={cfg.storefront || 'us'}
          onChange={(e) => update('storefront', e.target.value)}
          maxLength={2}
          className="w-24 rounded-lg border border-white/10 bg-surface px-3 py-2 uppercase"
        />
        <label className="block text-xs text-white/50">media-user-token</label>
        <textarea
          value={cfg['media-user-token'] || ''}
          onChange={(e) => update('media-user-token', e.target.value)}
          rows={3}
          className="w-full rounded-lg border border-white/10 bg-surface px-3 py-2 font-mono text-sm"
        />
      </section>

      <section className="mt-4 space-y-3 rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">Output folders</h3>
        {[
          ['aac-save-folder', 'AAC / default'],
          ['alac-save-folder', 'ALAC'],
          ['atmos-save-folder', 'Dolby Atmos'],
          ['mv-save-folder', 'Music videos'],
        ].map(([key, label]) => (
          <div key={key} className="flex gap-2">
            <div className="flex-1">
              <label className="text-xs text-white/50">{label}</label>
              <input
                value={cfg[key] || ''}
                onChange={(e) => update(key, e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm"
              />
            </div>
            <button
              type="button"
              onClick={() => pickFolder(key)}
              className="mt-5 rounded-lg bg-surface px-3 text-sm hover:bg-surface-hover"
            >
              Browse
            </button>
          </div>
        ))}
      </section>

      <section className="mt-4 space-y-2 rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">Metadata</h3>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={!!cfg['embed-lrc']} onChange={(e) => update('embed-lrc', e.target.checked)} />
          Embed lyrics (LRC)
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={!!cfg['embed-cover']} onChange={(e) => update('embed-cover', e.target.checked)} />
          Embed album cover
        </label>
        <label className="block text-xs text-white/50">AAC type</label>
        <select
          value={cfg['aac-type'] || 'aac-lc'}
          onChange={(e) => update('aac-type', e.target.value)}
          className="rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm"
        >
          <option value="aac-lc">aac-lc</option>
          <option value="aac">aac</option>
          <option value="aac-binaural">aac-binaural</option>
          <option value="aac-downmix">aac-downmix</option>
        </select>
      </section>

      <section className="mt-4 rounded-xl border border-white/10 bg-surface-raised p-4">
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex w-full items-center justify-between text-sm font-medium"
        >
          Advanced
          <span>{showAdvanced ? '▲' : '▼'}</span>
        </button>
        {showAdvanced && (
          <div className="mt-4 space-y-3 border-t border-white/10 pt-4">
            <div>
              <label className="text-xs text-white/50">decrypt-m3u8-port</label>
              <input
                value={cfg['decrypt-m3u8-port'] || ''}
                onChange={(e) => update('decrypt-m3u8-port', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="text-xs text-white/50">get-m3u8-port</label>
              <input
                value={cfg['get-m3u8-port'] || ''}
                onChange={(e) => update('get-m3u8-port', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm"
              />
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={!!cfg['convert-after-download']}
                onChange={(e) => update('convert-after-download', e.target.checked)}
              />
              Convert after download (ffmpeg)
            </label>
          </div>
        )}
      </section>

      <section className="mt-4 rounded-xl border border-white/10 bg-surface-raised p-4">
        <div className="flex items-center justify-between">
          <h3 className="font-medium">Dependencies</h3>
          <button onClick={onRefreshDeps} className="text-sm text-accent hover:underline">
            Test
          </button>
        </div>
        <ul className="mt-3 space-y-2">
          {(deps || []).map((d) => (
            <li key={d.name} className="flex justify-between text-sm">
              <span>{d.name}</span>
              <span className={d.ok ? 'text-green-400' : 'text-yellow-400'}>{d.ok ? 'OK' : 'Not found'}</span>
            </li>
          ))}
        </ul>
        <p className="mt-3 text-xs text-white/40">
          ALAC / Atmos require wrapper on ports above. See README-WINDOWS.md for manual setup via WSL.
        </p>
      </section>

      <div className="mt-6 flex gap-3 pb-8">
        <button onClick={save} className="rounded-xl bg-accent px-6 py-2 font-medium">
          {saved ? 'Saved!' : 'Save settings'}
        </button>
        <button onClick={onShowWizard} className="rounded-xl border border-white/20 px-4 py-2 text-sm">
          Re-run setup wizard
        </button>
      </div>
    </div>
  )
}
