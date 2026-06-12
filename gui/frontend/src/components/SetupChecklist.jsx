import { useState } from 'react'

const STOREFRONTS = ['us', 'jp', 'gb', 'ca', 'de', 'fr', 'au', 'kr', 'cn', 'tw', 'hk', 'sg', 'in', 'br', 'mx']

export default function SetupChecklist({ settings, deps, onComplete, onRefreshDeps, onPickFolder }) {
  const [cfg, setCfg] = useState(
    settings || {
      'media-user-token': '',
      storefront: 'us',
      'aac-save-folder': '',
    },
  )

  const update = (key, val) => setCfg((c) => ({ ...c, [key]: val }))

  const mp4boxOk = deps?.some((d) => d.name === 'MP4Box' && d.ok)
  const ffmpegOk = deps?.some((d) => d.name === 'ffmpeg' && d.ok)
  const ytDlpOk = deps?.some((d) => d.name === 'yt-dlp' && d.ok)
  const hasToken = (cfg['media-user-token'] || '').length > 50
  const hasFolder = Boolean((cfg['aac-save-folder'] || '').trim())

  const finish = () => onComplete(cfg)

  return (
    <div className="flex h-screen items-center justify-center bg-surface p-6">
      <div className="w-full max-w-lg rounded-2xl border border-white/10 bg-surface-raised p-6 shadow-2xl sm:p-8">
        <h2 className="text-2xl font-bold">Quick setup</h2>
        <p className="mt-2 text-sm text-white/60">
          A few one-time steps so Apple Music and YouTube downloads work. You can change everything later in Settings.
        </p>

        <section className="mt-6 space-y-3 rounded-xl border border-white/10 bg-surface p-4">
          <h3 className="text-sm font-medium">Apple Music (AAC)</h3>
          <label className="block text-xs text-white/50">Storefront</label>
          <select
            value={cfg.storefront || 'us'}
            onChange={(e) => update('storefront', e.target.value)}
            className="w-full rounded-lg border border-white/10 bg-surface-raised px-3 py-2 text-sm"
          >
            {STOREFRONTS.map((s) => (
              <option key={s} value={s}>
                {s.toUpperCase()}
              </option>
            ))}
          </select>
          <label className="block text-xs text-white/50">media-user-token</label>
          <textarea
            value={cfg['media-user-token'] || ''}
            onChange={(e) => update('media-user-token', e.target.value)}
            placeholder="From music.apple.com cookies (DevTools → Application)"
            rows={3}
            className="w-full rounded-lg border border-white/10 bg-surface-raised px-3 py-2 font-mono text-xs"
          />
          <p className={`text-xs ${hasToken ? 'text-green-400' : 'text-yellow-300/90'}`}>
            {hasToken ? 'Token looks configured.' : 'Required for Apple Music AAC — skip if you only use YouTube.'}
          </p>
        </section>

        <section className="mt-4 space-y-2 rounded-xl border border-white/10 bg-surface p-4">
          <h3 className="text-sm font-medium">Output folder</h3>
          <div className="flex gap-2">
            <input
              value={cfg['aac-save-folder'] || ''}
              onChange={(e) => update('aac-save-folder', e.target.value)}
              placeholder="~/Music/Aura Downloads"
              className="min-w-0 flex-1 rounded-lg border border-white/10 bg-surface-raised px-3 py-2 text-sm"
            />
            <button
              type="button"
              onClick={async () => {
                const path = await onPickFolder?.()
                if (path) update('aac-save-folder', path)
              }}
              className="shrink-0 rounded-lg border border-white/15 px-3 text-sm hover:bg-white/5"
            >
              Browse
            </button>
          </div>
          <p className={`text-xs ${hasFolder ? 'text-green-400' : 'text-yellow-300/90'}`}>
            {hasFolder ? 'Download folder set.' : 'Pick where AAC and YouTube files should be saved.'}
          </p>
        </section>

        <section className="mt-4 rounded-xl border border-white/10 bg-surface p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">Tools</h3>
            <button type="button" onClick={onRefreshDeps} className="text-xs text-accent hover:underline">
              Refresh
            </button>
          </div>
          <ul className="mt-3 space-y-2 text-sm">
            <li className="flex justify-between">
              <span>MP4Box (tagging)</span>
              <span className={mp4boxOk ? 'text-green-400' : 'text-yellow-400'}>{mp4boxOk ? 'OK' : 'Missing'}</span>
            </li>
            <li className="flex justify-between">
              <span>ffmpeg (YouTube)</span>
              <span className={ffmpegOk ? 'text-green-400' : 'text-yellow-400'}>{ffmpegOk ? 'OK' : 'Missing'}</span>
            </li>
            <li className="flex justify-between">
              <span>yt-dlp (YouTube)</span>
              <span className={ytDlpOk ? 'text-green-400' : 'text-yellow-400'}>{ytDlpOk ? 'OK' : 'Missing'}</span>
            </li>
          </ul>
          <p className="mt-2 text-xs text-white/40">Install via Homebrew if not bundled: brew install gpac ffmpeg yt-dlp</p>
        </section>

        <button
          type="button"
          onClick={finish}
          className="mt-6 w-full rounded-xl bg-accent py-3 text-sm font-semibold hover:bg-accent-muted"
        >
          Get started
        </button>
      </div>
    </div>
  )
}
