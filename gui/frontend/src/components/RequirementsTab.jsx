const TOOLS = [
  {
    name: 'Go 1.23+',
    url: 'https://go.dev/dl/',
    requiredFor: 'Building the app from source (CLI + GUI)',
    withoutIt: 'You cannot build from source. Installer users do not need it.',
  },
  {
    name: 'Node.js 18+',
    url: 'https://nodejs.org/en/download',
    requiredFor: 'Building the React frontend',
    withoutIt: 'Frontend build and Wails build will fail.',
  },
  {
    name: 'Wails CLI',
    url: 'https://wails.io/docs/gettingstarted/installation',
    command: 'go install github.com/wailsapp/wails/v2/cmd/wails@latest',
    requiredFor: 'Building AuraAudioDownloader.exe (GUI)',
    withoutIt: 'You can still build the CLI only (`amd.exe`).',
  },
  {
    name: 'MP4Box.exe',
    url: 'https://gpac.io/downloads/gpac-nightly-builds/',
    requiredFor: 'Tagging and post-processing tracks',
    withoutIt: 'Core track processing/tagging will fail.',
  },
  {
    name: 'ffmpeg.exe',
    url: 'https://ffmpeg.org/download.html',
    requiredFor: 'Optional conversion, animated artwork, and YouTube audio extraction',
    withoutIt: 'Apple Music downloads still work; conversion/YouTube mode need it.',
  },
  {
    name: 'yt-dlp',
    url: 'https://github.com/yt-dlp/yt-dlp/releases',
    requiredFor: 'YouTube audio downloads (Download tab → YouTube Audio mode)',
    withoutIt: 'YouTube mode is unavailable. Apple Music downloads are unaffected.',
  },
  {
    name: 'mp4decrypt.exe',
    url: 'https://www.bento4.com/downloads/',
    requiredFor: 'Music video downloads',
    withoutIt: 'MV downloads are skipped/unavailable.',
  },
  {
    name: 'wrapper (WSL/Linux)',
    url: 'https://github.com/WorldObservationLog/wrapper/releases',
    requiredFor: 'ALAC and Dolby Atmos decryption',
    withoutIt: 'AAC still works. ALAC/Atmos are unavailable.',
  },
]

const FEATURES = [
  { feature: 'AAC downloads (256 kbps)', worksWithoutTools: true, notes: 'Needs Apple Music subscription + media-user-token in Settings. MP4Box on PATH or in app tools folder for tagging. No wrapper required.' },
  { feature: 'ALAC downloads', worksWithoutTools: false, notes: 'Needs wrapper running on configured ports (WSL2 on Windows).' },
  { feature: 'Dolby Atmos downloads', worksWithoutTools: false, notes: 'Needs wrapper running on configured ports (WSL2 on Windows).' },
  { feature: 'Lyrics (LRC)', worksWithoutTools: true, notes: 'Needs valid media-user-token.' },
  { feature: 'Music video downloads', worksWithoutTools: false, notes: 'Needs mp4decrypt + media-user-token.' },
  { feature: 'Post-download conversion', worksWithoutTools: false, notes: 'Needs ffmpeg.' },
  { feature: 'YouTube audio (DJ sets, mixes)', worksWithoutTools: false, notes: 'Enable YouTube Audio on Download tab. Needs yt-dlp + ffmpeg. No Apple account.' },
]

export default function RequirementsTab({ deps, onRefreshDeps }) {
  return (
    <div className="mx-auto h-full max-w-content space-y-4 overflow-y-auto pb-8">
      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h2 className="text-xl font-semibold">Build and tool requirements</h2>
        <p className="mt-1 text-sm text-white/60">
          Use this page to see what to install, where to download it, and what features work with or without each tool.
        </p>
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <div className="flex items-center justify-between">
          <h3 className="font-medium">Live dependency status</h3>
          <button onClick={onRefreshDeps} className="text-sm text-accent hover:underline">
            Refresh
          </button>
        </div>
        <div className="mt-3 grid gap-2 md:grid-cols-2">
          {(deps || []).map((d) => (
            <div key={d.name} className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2 text-sm">
              <span>{d.name}</span>
              <span className={d.ok ? 'text-green-400' : 'text-yellow-300'}>{d.ok ? 'Detected' : 'Not found'}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">What to install</h3>
        <div className="mt-3 space-y-3">
          {TOOLS.map((tool) => (
            <div key={tool.name} className="rounded-lg border border-white/10 p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="font-medium">{tool.name}</p>
                <a className="text-sm text-accent hover:underline" href={tool.url} target="_blank" rel="noreferrer">
                  Download
                </a>
              </div>
              <p className="mt-1 text-sm text-white/70">
                <span className="text-white/90">Needed for:</span> {tool.requiredFor}
              </p>
              <p className="text-sm text-white/55">
                <span className="text-white/90">Without it:</span> {tool.withoutIt}
              </p>
              {tool.command && (
                <pre className="mt-2 overflow-x-auto rounded bg-black/40 px-3 py-2 text-xs text-white/80">{tool.command}</pre>
              )}
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">AAC troubleshooting</h3>
        <ul className="mt-2 list-inside list-disc space-y-1 text-sm text-white/70">
          <li><span className="text-white/90">License failed</span> — refresh media-user-token from music.apple.com (DevTools → Application → Cookies).</li>
          <li><span className="text-white/90">Download incomplete</span> — check network, retry; Apple CDN may have timed out.</li>
          <li><span className="text-white/90">Decrypt / tagging failed</span> — ensure MP4Box is on PATH; open the log file from the Queue tab.</li>
          <li><span className="text-white/90">Lossless (ALAC)</span> — not AAC; requires the wrapper service (see above). Use Quality: AAC in the Download tab for no-wrapper downloads.</li>
        </ul>
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">Feature availability</h3>
        <div className="mt-3 space-y-2">
          {FEATURES.map((item) => (
            <div key={item.feature} className="rounded-lg border border-white/10 px-3 py-2">
              <div className="flex items-center justify-between text-sm">
                <span>{item.feature}</span>
                <span className={item.worksWithoutTools ? 'text-green-400' : 'text-yellow-300'}>
                  {item.worksWithoutTools ? 'Works by default' : 'Needs extra tools'}
                </span>
              </div>
              <p className="mt-1 text-xs text-white/55">{item.notes}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h3 className="font-medium">Build commands (Windows)</h3>
        <p className="mt-1 text-sm text-white/60">Run these from the project root in PowerShell.</p>
        <pre className="mt-3 overflow-x-auto rounded bg-black/40 px-3 py-2 text-xs text-white/80">
{`# Install Wails CLI once
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Build GUI exe + CLI + (optionally) installer
.\\scripts\\build-windows.ps1

# Build exe only, skip installer
.\\scripts\\build-windows.ps1 -SkipInstaller`}
        </pre>
      </section>
    </div>
  )
}
