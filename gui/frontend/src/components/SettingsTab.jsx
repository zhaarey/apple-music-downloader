import { useEffect, useState } from 'react'
import { OpenLogFile, OpenConfigFolder } from '../wailsjs/go/main/App'
import PageShell from './PageShell'
import YouTubeOutputLocationSwitch from './YouTubeOutputLocationSwitch'
import AppleSyncResetPanel from './AppleSyncResetPanel'
import { youtubeOutputLocation } from '../lib/youtubeOutput'

const sectionClass = 'rounded-xl border border-white/10 bg-surface-raised p-5'
const labelClass = 'block text-xs font-medium text-white/50'
const inputClass =
  'mt-1.5 w-full rounded-lg border border-white/10 bg-surface px-3 py-2 text-sm outline-none transition focus:border-accent/40'

function SettingsSection({ title, description, children, className = '' }) {
  return (
    <section className={`${sectionClass} ${className}`}>
      <div className="mb-4 border-b border-white/10 pb-3">
        <h3 className="text-sm font-semibold tracking-wide text-white">{title}</h3>
        {description && <p className="mt-1 text-xs leading-relaxed text-white/50">{description}</p>}
      </div>
      <div className="space-y-4">{children}</div>
    </section>
  )
}

function FolderRow({ label, value, onChange, onBrowse }) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
      <div className="min-w-0 flex-1">
        <label className={labelClass}>{label}</label>
        <input value={value || ''} onChange={onChange} className={inputClass} />
      </div>
      <button
        type="button"
        onClick={onBrowse}
        className="shrink-0 rounded-lg border border-white/15 bg-surface px-4 py-2 text-sm hover:bg-surface-hover"
      >
        Browse
      </button>
    </div>
  )
}

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
    ? '~/Library/Application Support/AuraAudioDownloader'
    : '%APPDATA%\\AuraAudioDownloader'

  return (
    <PageShell wide>
      <header className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold">Settings</h2>
          <p className="mt-1 text-sm text-white/50">AAC downloads, library folders, and optional YouTube tools.</p>
          <p className="mt-1 text-xs text-white/40">Config: {configDirHint}</p>
        </div>
        <button
          type="button"
          onClick={() => OpenConfigFolder()}
          className="shrink-0 self-start rounded-lg border border-white/15 px-3 py-1.5 text-xs text-white/70 hover:bg-white/5"
        >
          Open config folder
        </button>
      </header>

      <div className="grid gap-5 xl:grid-cols-2">
        <SettingsSection
          title="Apple Music account"
          description="Required for AAC downloads. Copy media-user-token from music.apple.com while signed in (DevTools → Application → Cookies)."
          className="xl:col-span-2"
        >
          <div className="grid gap-4 sm:grid-cols-[auto_1fr]">
            <div>
              <label className={labelClass}>Storefront</label>
              <input
                value={cfg.storefront || 'us'}
                onChange={(e) => update('storefront', e.target.value)}
                maxLength={2}
                className={`${inputClass} w-24 uppercase`}
              />
            </div>
            <div className="sm:col-span-1">
              <label className={labelClass}>media-user-token</label>
              <textarea
                value={cfg['media-user-token'] || ''}
                onChange={(e) => update('media-user-token', e.target.value)}
                rows={3}
                className={`${inputClass} font-mono text-xs`}
                placeholder="Paste token from music.apple.com cookies"
              />
            </div>
          </div>
        </SettingsSection>

        <SettingsSection
          title="Download folder"
          description="Apple Music saves as Artist → Album → numbered tracks (.m4a AAC). Browse saves immediately."
          className="xl:col-span-2"
        >
          <FolderRow
            label="Apple Music downloads (AAC)"
            value={cfg['aac-save-folder']}
            onChange={(e) => update('aac-save-folder', e.target.value)}
            onBrowse={() => pickFolder('aac-save-folder')}
          />
        </SettingsSection>

        <SettingsSection
          title="File names & tags"
          description="Controls folder layout and embedded metadata on downloaded tracks."
        >
          <div>
            <label className={labelClass}>Artist folder</label>
            <input
              value={cfg['artist-folder-format'] || '{ArtistName}'}
              onChange={(e) => update('artist-folder-format', e.target.value)}
              placeholder="{ArtistName}"
              className={`${inputClass} font-mono text-xs`}
            />
          </div>
          <div>
            <label className={labelClass}>Album folder</label>
            <input
              value={cfg['album-folder-format'] || '{AlbumName}'}
              onChange={(e) => update('album-folder-format', e.target.value)}
              placeholder="{AlbumName}"
              className={`${inputClass} font-mono text-xs`}
            />
          </div>
          <div>
            <label className={labelClass}>Track filename</label>
            <input
              value={cfg['song-file-format'] || '{TrackNumber}. {SongName}'}
              onChange={(e) => update('song-file-format', e.target.value)}
              placeholder="{TrackNumber}. {SongName}"
              className={`${inputClass} font-mono text-xs`}
            />
          </div>
          <label className="flex items-center gap-2.5 text-sm text-white/80">
            <input
              type="checkbox"
              checked={cfg['embed-cover'] !== false}
              onChange={(e) => update('embed-cover', e.target.checked)}
            />
            Embed album artwork in each track
          </label>
          <label className="flex items-center gap-2.5 text-sm text-white/80">
            <input
              type="checkbox"
              checked={cfg['tag-sort-order'] !== false}
              onChange={(e) => update('tag-sort-order', e.target.checked)}
            />
            Embed sort tags (helps Apple Music library order)
          </label>
          <label className="flex items-center gap-2.5 text-sm text-white/80">
            <input
              type="checkbox"
              checked={cfg['tag-itunes-id'] !== false}
              onChange={(e) => update('tag-itunes-id', e.target.checked)}
            />
            Embed iTunes catalog IDs
          </label>
        </SettingsSection>

        <SettingsSection
          title="Duplicate detection"
          description="Extra folders to scan before download. Matching tracks are skipped automatically."
        >
          {(cfg['duplicate-check-folders'] || []).length === 0 && (
            <p className="text-xs text-white/40">No extra folders — only your download folder is checked.</p>
          )}
          {(cfg['duplicate-check-folders'] || []).map((folder, idx) => (
            <div key={`${folder}-${idx}`} className="flex gap-2">
              <input
                value={folder}
                onChange={(e) => {
                  const next = [...(cfg['duplicate-check-folders'] || [])]
                  next[idx] = e.target.value
                  update('duplicate-check-folders', next)
                }}
                className={`${inputClass} mt-0`}
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
          <div className="flex flex-wrap gap-2 pt-1">
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
        </SettingsSection>

        <SettingsSection
          title="YouTube downloads"
          description="Optional — used on the YouTube tab. Requires yt-dlp and ffmpeg (see Requirements tab)."
          className="xl:col-span-2"
        >
          <label className="flex items-center gap-2.5 text-sm text-white/80">
            <input
              type="checkbox"
              checked={!!cfg['youtube-mode']}
              onChange={(e) => update('youtube-mode', e.target.checked)}
            />
            Open on YouTube tab by default
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
            <FolderRow
              label="YouTube download folder"
              value={cfg['youtube-save-folder']}
              onChange={(e) => update('youtube-save-folder', e.target.value)}
              onBrowse={() => pickFolder('youtube-save-folder')}
            />
          ) : (
            <div className="rounded-lg border border-white/10 bg-black/20 px-3 py-2.5 text-sm text-white/60">
              <p className="text-xs text-white/45">Using Apple Music download folder</p>
              <p className="mt-1 truncate font-mono text-white/80" title={cfg['aac-save-folder'] || ''}>
                {cfg['aac-save-folder'] || 'Not set — configure above'}
              </p>
            </div>
          )}
          <div>
            <label className={labelClass}>yt-dlp path (optional)</label>
            <input
              value={cfg['yt-dlp-path'] || ''}
              onChange={(e) => update('yt-dlp-path', e.target.value)}
              placeholder="yt-dlp"
              className={`${inputClass} font-mono text-xs`}
            />
          </div>
        </SettingsSection>

        <SettingsSection title="Tools & diagnostics">
          <div className="flex items-center justify-between gap-2">
            <p className="text-xs text-white/50">MP4Box and bundled tools for tagging AAC files.</p>
            <button type="button" onClick={onRefreshDeps} className="text-sm text-accent hover:underline">
              Test
            </button>
          </div>
          <ul className="space-y-1.5 rounded-lg border border-white/10 bg-black/20 px-3 py-2">
            {(deps || []).map((d) => (
              <li key={d.name} className="flex justify-between text-sm">
                <span className="text-white/80">{d.name}</span>
                <span className={d.ok ? 'text-green-400' : 'text-amber-400'}>{d.ok ? 'OK' : 'Missing'}</span>
              </li>
            ))}
          </ul>
          <button
            type="button"
            onClick={OpenLogFile}
            className="rounded-lg border border-white/15 px-4 py-2 text-sm hover:bg-white/5"
          >
            Open log file
          </button>
        </SettingsSection>

        {!isMac && (
          <SettingsSection
            title="Reset Apple sync"
            description="If iPhone artwork only updates after rebooting Windows, run this after syncing — same as canceling a restart, without deleting caches."
            className="xl:col-span-2"
          >
            <AppleSyncResetPanel />
          </SettingsSection>
        )}

        {activityLogs.length > 0 && (
          <SettingsSection title="Recent activity" className="xl:col-span-2">
            <ul className="max-h-40 space-y-1 overflow-y-auto rounded-lg border border-white/10 bg-black/20 px-3 py-2 font-mono text-xs text-white/60">
              {activityLogs
                .slice(-40)
                .reverse()
                .map((entry, i) => (
                  <li key={`${entry.time}-${i}`} className="truncate">
                    <span className="text-white/35">{entry.time}</span> {entry.msg}
                  </li>
                ))}
            </ul>
          </SettingsSection>
        )}
      </div>

      <div className="mt-6 flex flex-wrap gap-3 border-t border-white/10 pt-6">
        <button type="button" onClick={save} className="rounded-xl bg-accent px-6 py-2.5 text-sm font-medium">
          {saved ? 'Saved!' : 'Save settings'}
        </button>
        {onShowSetupChecklist && (
          <button
            type="button"
            onClick={onShowSetupChecklist}
            className="rounded-xl border border-white/20 px-4 py-2.5 text-sm hover:bg-white/5"
          >
            Re-run setup checklist
          </button>
        )}
        {onShowWizard && (
          <button
            type="button"
            onClick={onShowWizard}
            className="rounded-xl border border-white/20 px-4 py-2.5 text-sm hover:bg-white/5"
          >
            Re-run setup wizard
          </button>
        )}
      </div>
    </PageShell>
  )
}
