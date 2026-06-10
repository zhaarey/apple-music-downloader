import { useEffect, useState } from 'react'
import { OpenLogFile, OpenConfigFolder } from '../wailsjs/go/main/App'
import IPhoneArtworkGuide from './IPhoneArtworkGuide'
import FullArtworkResetGuide from './FullArtworkResetGuide'
import PageShell from './PageShell'
import YouTubeOutputLocationSwitch from './YouTubeOutputLocationSwitch'
import { youtubeOutputLocation } from '../lib/youtubeOutput'

const sectionClass = 'space-y-3 rounded-xl border border-white/10 bg-surface-raised p-4'

export default function SettingsTab({
  settings,
  deps,
  platform = 'windows',
  activityLogs = [],
  onSave,
  onPickFolder,
  onRefreshDeps,
  onShowWizard,
  onShowSetupChecklist,
}) {
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
    if (!path) return
    update(key, path)
    await onSave({ [key]: path })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const isMac = platform === 'darwin'
  const configDirHint = isMac
    ? 'Config is saved to ~/Library/Application Support/AuraAudioDownloader'
    : 'Config is saved to your AppData folder'

  return (
    <PageShell wide>
      <div className="mb-6">
        <h2 className="text-xl font-semibold">Settings</h2>
        <p className="mt-1 text-sm text-white/50">{configDirHint}</p>
        <button
          type="button"
          onClick={() => OpenConfigFolder()}
          className="mt-2 rounded-lg border border-white/15 px-3 py-1.5 text-xs text-white/70 hover:bg-white/5"
        >
          Open config folder
        </button>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
      <section className={`${sectionClass} space-y-4`}>
        <h3 className="font-medium">Account</h3>
        <label className="block text-xs text-white/50">Storefront</label>
        <input
          value={cfg.storefront || 'us'}
          onChange={(e) => update('storefront', e.target.value)}
          maxLength={2}
          className="w-24 rounded-lg border border-white/10 bg-surface px-3 py-2 uppercase"
        />
        <label className="block text-xs text-white/50">media-user-token (required for AAC)</label>
        <textarea
          value={cfg['media-user-token'] || ''}
          onChange={(e) => update('media-user-token', e.target.value)}
          rows={3}
          className="w-full rounded-lg border border-white/10 bg-surface px-3 py-2 font-mono text-sm"
        />
      </section>

      <section className={`${sectionClass} lg:col-span-2`}>
        <h3 className="font-medium">Output folders</h3>
        <p className="text-xs text-white/50">
          Downloads save as Artist → Album → track files. Browse saves immediately; typed paths need Save settings.
        </p>
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

      <section className={sectionClass}>
        <h3 className="font-medium">Duplicate detection</h3>
        <p className="text-xs text-white/50">
          Extra folders to scan before download (in addition to the output folder above). Useful if you moved
          your library — matching files are skipped automatically.
        </p>
        {(cfg['duplicate-check-folders'] || []).map((folder, idx) => (
          <div key={`${folder}-${idx}`} className="flex gap-2">
            <input
              value={folder}
              onChange={(e) => {
                const next = [...(cfg['duplicate-check-folders'] || [])]
                next[idx] = e.target.value
                update('duplicate-check-folders', next)
              }}
              className="min-w-0 flex-1 rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm"
            />
            <button
              type="button"
              onClick={() => {
                const next = (cfg['duplicate-check-folders'] || []).filter((_, i) => i !== idx)
                update('duplicate-check-folders', next)
              }}
              className="shrink-0 rounded-lg border border-white/10 px-3 text-sm text-white/60 hover:bg-white/5"
            >
              Remove
            </button>
          </div>
        ))}
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => update('duplicate-check-folders', [...(cfg['duplicate-check-folders'] || []), ''])}
            className="rounded-lg border border-white/10 px-3 py-1.5 text-sm hover:bg-white/5"
          >
            Add folder
          </button>
          <button
            type="button"
            onClick={async () => {
              const path = await onPickFolder()
              if (!path) return
              const list = [...(cfg['duplicate-check-folders'] || [])]
              if (!list.some((p) => p.trim().toLowerCase() === path.trim().toLowerCase())) {
                list.push(path)
              }
              update('duplicate-check-folders', list)
              await onSave({ 'duplicate-check-folders': list.filter(Boolean) })
              setSaved(true)
              setTimeout(() => setSaved(false), 2000)
            }}
            className="rounded-lg bg-surface px-3 py-1.5 text-sm hover:bg-surface-hover"
          >
            Browse to add
          </button>
        </div>
      </section>

      <section className={sectionClass}>
        <h3 className="font-medium">Library organization</h3>
        <p className="text-xs text-white/50">
          Saves as Artist → Album → track files. Embedded tags help Apple Music / iTunes match your library.
        </p>
        <div>
          <label className="text-xs text-white/50">Artist folder</label>
          <input
            value={cfg['artist-folder-format'] || '{ArtistName}'}
            onChange={(e) => update('artist-folder-format', e.target.value)}
            placeholder="{ArtistName}"
            className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm font-mono"
          />
        </div>
        <div>
          <label className="text-xs text-white/50">Album folder</label>
          <input
            value={cfg['album-folder-format'] || '{AlbumName}'}
            onChange={(e) => update('album-folder-format', e.target.value)}
            placeholder="{AlbumName}"
            className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm font-mono"
          />
        </div>
        <div>
          <label className="text-xs text-white/50">Track filename</label>
          <input
            value={cfg['song-file-format'] || '{TrackNumber}. {SongName}'}
            onChange={(e) => update('song-file-format', e.target.value)}
            placeholder="{TrackNumber}. {SongName}"
            className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm font-mono"
          />
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={cfg['tag-sort-order'] !== false}
            onChange={(e) => update('tag-sort-order', e.target.checked)}
          />
          Embed sort tags (Title/Artist/Album sort fields)
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={cfg['tag-itunes-id'] !== false}
            onChange={(e) => update('tag-itunes-id', e.target.checked)}
          />
          Embed iTunes catalog IDs (better Apple Music matching)
        </label>
      </section>

      <section className={`${sectionClass} space-y-2`}>
        <h3 className="font-medium">Cover & lyrics</h3>
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

      <section className={sectionClass}>
        <h3 className="font-medium">Spotify playlist migration</h3>
        <p className="text-xs text-white/50">
          One-time setup to read public Spotify playlists and match songs on Apple Music (ISRC when available).
        </p>
        <ol className="mt-2 space-y-2 text-xs text-white/55">
          <li>
            <span className="font-medium text-white/75">1.</span> Create a free app at{' '}
            <a href="https://developer.spotify.com/dashboard" className="text-accent hover:underline" target="_blank" rel="noreferrer">
              developer.spotify.com
            </a>
          </li>
          <li>
            <span className="font-medium text-white/75">2.</span> Copy Client ID and Client secret below
          </li>
          <li>
            <span className="font-medium text-white/75">3.</span> On Spotify, make each playlist public (⋯ → Make public)
          </li>
          <li>
            <span className="font-medium text-white/75">4.</span> Apple Music tab → Bulk queue → paste playlist link
          </li>
        </ol>
        <div>
          <label className="text-xs text-white/50">Client ID</label>
          <input
            value={cfg['spotify-client-id'] || ''}
            onChange={(e) => update('spotify-client-id', e.target.value)}
            className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm font-mono"
          />
        </div>
        <div>
          <label className="text-xs text-white/50">Client secret</label>
          <input
            type="password"
            value={cfg['spotify-client-secret'] || ''}
            onChange={(e) => update('spotify-client-secret', e.target.value)}
            className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm font-mono"
          />
        </div>
      </section>

      <section className={sectionClass}>
        <h3 className="font-medium">YouTube downloads</h3>
        <p className="text-xs text-white/50">
          Used on the YouTube tab for audio, video, or both. Requires yt-dlp and ffmpeg.
        </p>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={!!cfg['youtube-mode']}
            onChange={(e) => update('youtube-mode', e.target.checked)}
          />
          Default to YouTube mode on launch
        </label>
        <YouTubeOutputLocationSwitch
          value={youtubeOutputLocation(cfg)}
          onChange={async (mode) => {
            update('youtube-output-location', mode)
            await onSave({ 'youtube-output-location': mode })
            setSaved(true)
            setTimeout(() => setSaved(false), 2000)
          }}
        />
        {youtubeOutputLocation(cfg) === 'separate' ? (
          <div className="flex gap-2">
            <div className="flex-1">
              <label className="text-xs text-white/50">Separate YouTube folder</label>
              <input
                value={cfg['youtube-save-folder'] || ''}
                onChange={(e) => update('youtube-save-folder', e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm"
              />
            </div>
            <button
              type="button"
              onClick={() => pickFolder('youtube-save-folder')}
              className="mt-5 rounded-lg bg-surface px-3 text-sm hover:bg-surface-hover"
            >
              Browse
            </button>
          </div>
        ) : (
          <div className="rounded-lg border border-white/10 bg-black/20 px-3 py-2.5 text-sm text-white/60">
            <p className="text-xs text-white/45">Apple Music download folder (AAC / default)</p>
            <p className="mt-1 truncate font-mono text-white/80" title={cfg['aac-save-folder'] || ''}>
              {cfg['aac-save-folder'] || 'Not set — configure under Output folders above'}
            </p>
          </div>
        )}
        <div>
          <label className="text-xs text-white/50">yt-dlp path (optional)</label>
          <input
            value={cfg['yt-dlp-path'] || ''}
            onChange={(e) => update('yt-dlp-path', e.target.value)}
            placeholder="yt-dlp"
            className="mt-1 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm font-mono"
          />
        </div>
      </section>

      <section className={sectionClass}>
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

      <section className={sectionClass}>
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
          {isMac
            ? 'Install MP4Box (gpac), ffmpeg, and yt-dlp via Homebrew if not bundled. See README-macOS.md.'
            : 'ALAC / Atmos require wrapper on ports above. See README-WINDOWS.md for manual setup via WSL.'}
        </p>
      </section>

      <section className={sectionClass}>
        <h3 className="font-medium">Diagnostics</h3>
        <p className="mt-1 text-xs text-white/50">
          If the app crashes or a download fails, open the log file for full details.
        </p>
        <button onClick={OpenLogFile} className="mt-3 rounded-lg bg-surface px-4 py-2 text-sm hover:bg-surface-hover">
          Open log file
        </button>
      </section>

      {activityLogs.length > 0 && (
        <section className={`${sectionClass} lg:col-span-2`}>
          <h3 className="font-medium">Recent activity</h3>
          <p className="mt-1 text-xs text-white/50">Latest download and engine messages from this session.</p>
          <ul className="mt-3 max-h-48 space-y-1 overflow-y-auto font-mono text-xs text-white/60">
            {activityLogs
              .slice(-40)
              .reverse()
              .map((entry, i) => (
                <li key={`${entry.time}-${i}`} className="truncate">
                  <span className="text-white/35">{entry.time}</span> {entry.msg}
                </li>
              ))}
          </ul>
        </section>
      )}

      <div className="flex flex-col gap-4 lg:col-span-2">
        <IPhoneArtworkGuide platform={platform} />
        <FullArtworkResetGuide platform={platform} />
      </div>

      <div className="flex flex-wrap gap-3 lg:col-span-2">
        <button onClick={save} className="rounded-xl bg-accent px-6 py-2 font-medium">
          {saved ? 'Saved!' : 'Save settings'}
        </button>
        {onShowSetupChecklist && (
          <button onClick={onShowSetupChecklist} className="rounded-xl border border-white/20 px-4 py-2 text-sm">
            Re-run setup checklist
          </button>
        )}
        {onShowWizard && (
          <button onClick={onShowWizard} className="rounded-xl border border-white/20 px-4 py-2 text-sm">
            Re-run setup wizard
          </button>
        )}
      </div>
      </div>
    </PageShell>
  )
}
