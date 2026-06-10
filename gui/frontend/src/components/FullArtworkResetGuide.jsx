import { useState } from 'react'
import ApplePurgePanel from './ApplePurgePanel'

const PC_STEPS = [
  'Quit Apple Music completely (check Task Manager for AppleMusic.exe).',
  'In Apple Music: select all affected albums → right-click → Delete from Library → Keep Files. This removes broken library index entries but keeps your .m4a downloads on disk.',
  'Run Deep purge PC caches below (or Tag Editor → Clear Apple Music art cache).',
  'Tag Editor → Sync repair tools → Update album artwork on each album folder (embeds one normalized JPEG per track).',
  'Re-import albums: File → Import, or drag each album folder into Apple Music. Confirm Get Info shows artwork on PC before syncing.',
]

const IPHONE_STEPS = [
  'On iPhone: Settings → Music → turn Sync Library OFF (if it was on). Wait 1–2 minutes.',
  'Open Music → Library → Albums. Tap each affected album → … → Delete from Library.',
  'Optional nuclear step: Settings → General → iPhone Storage → Music → review downloaded content; remove stale library data if albums still show blank art.',
  'Force-close Music (app switcher), then reopen.',
  'Reconnect USB → Apple Devices (Windows) or Finder (Mac) → sync ONE album first and verify art on the phone grid before syncing everything.',
  'Turn Sync Library back ON only after a test album looks correct.',
]

const WHY_BULLETS = [
  {
    title: 'Three separate artwork stores',
    body: 'Your .m4a files hold embedded art (covr). Apple Music on PC keeps its own artwork index (Album Artwork cache, .itc-style thumbnails). iPhone keeps a third grid thumbnail cache. They can drift apart.',
  },
  {
    title: 'PC “properties” ≠ what sync sends',
    body: 'Windows file properties and Get Info can read embedded tags from the file, while the library grid reads Apple’s pre-built thumbnail database. After cache clears or partial repairs, the library can show blank art even when files still have covers.',
  },
  {
    title: 'Repair attempts can worsen drift',
    body: 'Clearing PC caches without removing library entries leaves Apple Music pointing at deleted thumbnail files. Syncing to iPhone then copies that broken index — both sides show no art while files on disk remain correct.',
  },
  {
    title: 'Format and count matter',
    body: 'iPhone sync is picky: one JPEG embedded per track, ≤3000px, consistent across the album. PNG, multiple covers, or sidecar-only folder.jpg causes unpredictable results.',
  },
]

function Checklist({ steps, checks, onToggle }) {
  return (
    <ol className="mt-2 space-y-1">
      {steps.map((text, i) => (
        <li key={text}>
          <label className="flex cursor-pointer gap-2.5 rounded-lg px-2 py-1.5 text-sm text-white/75 hover:bg-white/[0.03]">
            <input
              type="checkbox"
              checked={checks[i]}
              onChange={(e) => onToggle(i, e.target.checked)}
              className="mt-0.5 shrink-0 rounded border-white/20"
            />
            <span className={checks[i] ? 'text-white/40 line-through' : ''}>
              <span className="mr-1.5 font-medium text-white/35">{i + 1}.</span>
              {text}
            </span>
          </label>
        </li>
      ))}
    </ol>
  )
}

export default function FullArtworkResetGuide({ platform = 'windows' }) {
  const [open, setOpen] = useState(false)
  const [pcChecks, setPcChecks] = useState(() => PC_STEPS.map(() => false))
  const [iphoneChecks, setIphoneChecks] = useState(() => IPHONE_STEPS.map(() => false))
  const isMac = platform === 'darwin'

  const toggle = (setter) => (index, checked) => {
    setter((prev) => {
      const next = [...prev]
      next[index] = checked
      return next
    })
  }

  return (
    <section className="mt-4 rounded-xl border border-red-500/20 bg-red-500/[0.03] p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="font-medium text-white/90">Full artwork reset (nuclear option)</h3>
          <p className="mt-1 text-xs leading-relaxed text-white/50">
            Use when artwork is missing or wrong on <strong className="font-medium text-white/65">both PC and iPhone</strong>,
            but your downloaded files still show correct art in properties or Tag Editor.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="shrink-0 rounded-lg border border-white/15 px-3 py-1.5 text-xs text-white/70 hover:bg-white/5"
        >
          {open ? 'Hide' : 'Show guide'}
        </button>
      </div>

      {open && (
        <div className="mt-4 space-y-5 border-t border-white/10 pt-4">
          <div>
            <h4 className="text-sm font-medium text-white/85">Why this is so unpredictable</h4>
            <ul className="mt-3 space-y-3">
              {WHY_BULLETS.map((item) => (
                <li key={item.title} className="rounded-lg border border-white/10 bg-black/20 px-3 py-2.5">
                  <p className="text-xs font-medium text-white/80">{item.title}</p>
                  <p className="mt-1 text-xs leading-relaxed text-white/50">{item.body}</p>
                </li>
              ))}
            </ul>
          </div>

          <div className="rounded-lg border border-amber-500/25 bg-amber-500/[0.06] px-3 py-2.5 text-xs leading-relaxed text-amber-100/90">
            <strong className="font-medium">Aura never deletes your download folders.</strong> A full reset clears Apple’s
            caches and library <em>references</em> — you re-import the same .m4a files afterward. Your files on disk stay put.
          </div>

          <div>
            <h4 className="text-sm font-medium text-white/85">Phase 1 — Reset PC library index</h4>
            <p className="mt-1 text-xs text-white/45">
              Do these in order. Check off each step as you go.
            </p>
            <Checklist steps={PC_STEPS} checks={pcChecks} onToggle={toggle(setPcChecks)} />
            {!isMac && (
              <div className="mt-4">
                <ApplePurgePanel compact />
              </div>
            )}
            {isMac && (
              <p className="mt-3 text-xs text-white/45">
                On Mac: quit Music.app, delete ~/Library/Caches/com.apple.Music/Artwork, then re-import albums.
              </p>
            )}
          </div>

          <div>
            <h4 className="text-sm font-medium text-white/85">Phase 2 — Reset iPhone library entries</h4>
            <p className="mt-1 text-xs text-white/45">
              No PC app can wipe iPhone artwork caches. You must remove library entries on the device.
            </p>
            <Checklist steps={IPHONE_STEPS} checks={iphoneChecks} onToggle={toggle(setIphoneChecks)} />
          </div>

          <div className="rounded-lg border border-white/10 bg-black/20 px-3 py-2.5 text-xs leading-relaxed text-white/50">
            <p className="font-medium text-white/70">After reset</p>
            <p className="mt-1">
              Sync one album at a time. If art appears inside the album but not on the grid, use{' '}
              <strong className="text-white/60">Wrong album art on iPhone?</strong> above for the lighter grid-cache fix.
            </p>
          </div>
        </div>
      )}
    </section>
  )
}
