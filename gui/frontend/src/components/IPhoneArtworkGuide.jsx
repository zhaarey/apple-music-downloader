import { useEffect, useState } from 'react'
import {
  ValidateIPhoneSyncFolder,
  PickFolder,
  PrepareAlbumForSync,
  PreviewPrepareAlbumForSync,
  IsAppleMusicRunning,
} from '../wailsjs/go/main/App'
import { useConfirm } from '../hooks/useConfirm'
import { confirmAndPrepareAlbum } from '../lib/prepareAlbumConfirm'
import { FolderSyncValidationPanel } from '../features/metadata/SyncValidationPanel'

const STEP_LABELS = ['Symptom', 'Check files', 'Fix PC', 'Fix iPhone', 'Re-sync']

const IPHONE_CHECKLIST = [
  'Open the Music app on your iPhone.',
  'Find the affected album in your library or Recently Added.',
  'Long-press the album → Delete from Library (not just Remove Download).',
  'If Sync Library is on: Settings → Music → turn off Sync Library briefly, delete the album, then turn it back on.',
  'Force-close Music (app switcher) if the wrong thumbnail still appears before re-syncing.',
]

function platformCopy(platform) {
  const isMac = platform === 'darwin'
  return {
    musicApp: isMac ? 'Music.app' : 'Apple Music',
    syncApp: isMac ? 'Finder' : 'Apple Devices',
    importHint: isMac
      ? 'In Music: File → Import, select the album folder on disk.'
      : 'In Apple Music: File → Import, select the album folder on disk.',
    removeHint: isMac
      ? 'In Music: right-click the album → Delete from Library, then re-import the folder.'
      : 'In Apple Music: right-click the album → Delete from Library, then re-import the folder.',
    syncHint: isMac
      ? 'Connect your iPhone → Finder → select device → Music → sync only the affected album first.'
      : 'Open Apple Devices → your iPhone → Music → sync only the affected album first.',
  }
}

function StepProgress({ step }) {
  return (
    <div className="mb-5 flex gap-1.5">
      {STEP_LABELS.map((label, i) => (
        <div key={label} className="flex min-w-0 flex-1 flex-col gap-1">
          <div className={`h-1 rounded-full ${i <= step ? 'bg-accent' : 'bg-white/10'}`} />
          <span className={`truncate text-[10px] ${i === step ? 'text-white/70' : 'text-white/30'}`}>{label}</span>
        </div>
      ))}
    </div>
  )
}

function ChecklistItem({ checked, onChange, children }) {
  return (
    <label className="flex cursor-pointer gap-2.5 rounded-lg px-2 py-1.5 text-sm text-white/75 hover:bg-white/[0.03]">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 shrink-0 rounded border-white/20"
      />
      <span className={checked ? 'text-white/45 line-through' : ''}>{children}</span>
    </label>
  )
}

export default function IPhoneArtworkGuide({ platform = 'windows' }) {
  const copy = platformCopy(platform)
  const { requestConfirm, ConfirmDialogSlot } = useConfirm()
  const [open, setOpen] = useState(false)
  const [step, setStep] = useState(0)
  const [albumFolder, setAlbumFolder] = useState('')
  const [validation, setValidation] = useState(null)
  const [busy, setBusy] = useState('')
  const [feedback, setFeedback] = useState(null)
  const [musicRunning, setMusicRunning] = useState(false)
  const [iphoneChecks, setIphoneChecks] = useState(() => IPHONE_CHECKLIST.map(() => false))
  const [symptomMatch, setSymptomMatch] = useState(null)

  const refreshMusicRunning = () => {
    IsAppleMusicRunning().then(setMusicRunning).catch(() => setMusicRunning(false))
  }

  useEffect(() => {
    if (open) refreshMusicRunning()
  }, [open, step])

  const folderName = albumFolder ? albumFolder.split(/[/\\]/).pop() : ''

  const pickAlbumFolder = async () => {
    const folder = await PickFolder()
    if (folder) {
      setAlbumFolder(folder)
      setValidation(null)
      setFeedback(null)
    }
  }

  const checkFolder = async () => {
    if (!albumFolder) return
    setBusy('check')
    setFeedback(null)
    try {
      const res = await ValidateIPhoneSyncFolder(albumFolder)
      setValidation(res)
      setFeedback({
        variant: res.ready ? 'success' : 'info',
        message: res.summary,
      })
    } catch (e) {
      setFeedback({ variant: 'error', message: String(e?.message || e) })
    } finally {
      setBusy('')
    }
  }

  const prepareArtwork = async () => {
    if (!albumFolder) return
    setBusy('prepare')
    setFeedback(null)
    try {
      const outcome = await confirmAndPrepareAlbum({
        requestConfirm,
        folder: albumFolder,
        PreviewPrepareAlbumForSync,
        PrepareAlbumForSync,
      })
      if (outcome?.cancelled) return
      if (outcome?.error) {
        setFeedback({ variant: 'error', message: outcome.error })
        return
      }
      const res = await ValidateIPhoneSyncFolder(albumFolder)
      setValidation(res)
      setFeedback({ variant: 'success', message: outcome.result?.summary || 'Artwork updated.' })
    } catch (e) {
      setFeedback({ variant: 'error', message: String(e?.message || e) })
    } finally {
      setBusy('')
    }
  }

  const resetGuide = () => {
    setStep(0)
    setAlbumFolder('')
    setValidation(null)
    setFeedback(null)
    setSymptomMatch(null)
    setIphoneChecks(IPHONE_CHECKLIST.map(() => false))
  }

  const filesLookGood = validation?.ready === true
  const iphoneDone = iphoneChecks.every(Boolean)

  const feedbackClass =
    feedback?.variant === 'success'
      ? 'border-green-500/30 bg-green-500/10 text-green-100'
      : feedback?.variant === 'error'
        ? 'border-red-500/30 bg-red-500/10 text-red-100'
        : 'border-sky-500/30 bg-sky-500/10 text-sky-100'

  return (
    <section className="rounded-xl border border-sky-500/25 bg-sky-500/[0.04] p-4">
      {ConfirmDialogSlot}

      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="font-medium text-white/90">Wrong album art on iPhone?</h3>
          <p className="mt-1 text-xs leading-relaxed text-white/50">
            Guided fix when library grid or Recently Added shows the wrong cover, but opening the album or playing a track
            shows the correct artwork.
          </p>
        </div>
        <button
          type="button"
          onClick={() => {
            if (open) resetGuide()
            setOpen((v) => !v)
          }}
          className="shrink-0 rounded-lg border border-white/15 px-3 py-1.5 text-xs text-white/70 hover:bg-white/5"
        >
          {open ? 'Close guide' : 'Start guide'}
        </button>
      </div>

      {open && (
        <div className="mt-4 border-t border-white/10 pt-4">
          <StepProgress step={step} />

          {step === 0 && (
            <div className="space-y-4">
              <p className="text-sm leading-relaxed text-white/70">
                This usually means your <strong className="font-medium text-white/90">audio files are fine</strong> — embedded
                artwork is correct — but iPhone&apos;s <strong className="font-medium text-white/90">album grid thumbnail cache</strong>{' '}
                is stale or wrong. Aura can fix the PC side; you must delete the album on the phone before re-syncing.
              </p>
              <div className="rounded-lg border border-white/10 bg-black/20 p-3 text-sm text-white/65">
                <p className="font-medium text-white/85">Does this match what you see?</p>
                <ul className="mt-2 list-inside list-disc space-y-1 text-xs leading-relaxed">
                  <li>Wrong cover on the library shelf or Recently Added</li>
                  <li>Correct cover when you open the album or play a song</li>
                </ul>
              </div>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setSymptomMatch(true)
                    setStep(1)
                  }}
                  className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white"
                >
                  Yes — that&apos;s my issue
                </button>
                <button
                  type="button"
                  onClick={() => setSymptomMatch(false)}
                  className="rounded-lg border border-white/15 px-4 py-2 text-sm text-white/70 hover:bg-white/5"
                >
                  No — art is wrong everywhere
                </button>
              </div>
              {symptomMatch === false && (
                <p className="rounded-lg border border-yellow-500/25 bg-yellow-500/10 px-3 py-2 text-xs text-yellow-100/90">
                  If artwork is missing or wrong on <strong>both PC and iPhone</strong> but files still look correct in
                  properties, use <strong>Full artwork reset</strong> below this guide — library indexes likely drifted from
                  your files. If only embedded art is bad, fix it in <strong>Tag Editor</strong> first, then run this guide.
                </p>
              )}
            </div>
          )}

          {step === 1 && (
            <div className="space-y-4">
              <p className="text-sm text-white/70">
                Pick the album folder on your PC (the folder that contains the <code className="text-white/80">.m4a</code>{' '}
                files). We&apos;ll check whether embedded artwork is consistent — the usual cause of grid-only glitches is
                good files + bad cache.
              </p>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  disabled={!!busy}
                  onClick={() => void pickAlbumFolder()}
                  className="rounded-lg border border-white/15 px-4 py-2 text-sm hover:bg-white/5 disabled:opacity-50"
                >
                  {albumFolder ? `Change folder (${folderName})` : 'Choose album folder…'}
                </button>
                {albumFolder && (
                  <button
                    type="button"
                    disabled={!!busy}
                    onClick={() => void checkFolder()}
                    className="rounded-lg bg-accent px-4 py-2 text-sm font-medium disabled:opacity-50"
                  >
                    {busy === 'check' ? 'Checking…' : 'Check artwork'}
                  </button>
                )}
              </div>
              {albumFolder && (
                <p className="truncate text-xs text-white/40" title={albumFolder}>
                  {albumFolder}
                </p>
              )}
              {validation && (
                <div className="space-y-3">
                  <FolderSyncValidationPanel result={validation} />
                  {!filesLookGood && albumFolder && (
                    <button
                      type="button"
                      disabled={!!busy || musicRunning}
                      onClick={() => void prepareArtwork()}
                      className="rounded-lg border border-red-500/35 bg-red-500/10 px-4 py-2 text-sm text-red-100 hover:bg-red-500/20 disabled:opacity-50"
                    >
                      {busy === 'prepare' ? 'Updating…' : 'Fix embedded artwork in this folder'}
                    </button>
                  )}
                  {filesLookGood && (
                    <p className="rounded-lg border border-green-500/25 bg-green-500/10 px-3 py-2 text-xs text-green-100/90">
                      Files look good — the problem is almost certainly PC or iPhone cache. Continue to fix the PC side.
                    </p>
                  )}
                </div>
              )}
            </div>
          )}

          {step === 2 && (
            <div className="space-y-4">
              <p className="text-sm text-white/70">
                Quit <strong className="text-white/90">{copy.musicApp}</strong> on this PC, then use Tag Editor sync
                repair to clear the artwork cache and re-import the album so sync does not reuse a stale thumbnail index.
              </p>
              {musicRunning && (
                <p className="rounded-lg border border-yellow-500/25 bg-yellow-500/10 px-3 py-2 text-xs text-yellow-100/90">
                  {copy.musicApp} is still running — quit it before clearing caches.
                </p>
              )}
              <ol className="list-decimal space-y-2 pl-5 text-sm text-white/65">
                <li>Quit {copy.musicApp} completely.</li>
                <li>
                  Open <strong className="text-white/80">Tag Editor</strong> → expand Sync repair tools →{' '}
                  <strong className="text-white/80">Clear Apple Music art cache</strong> (safe for your .m4a files).
                </li>
                <li>{copy.removeHint}</li>
                <li>{copy.importHint}</li>
              </ol>
            </div>
          )}

          {step === 3 && (
            <div className="space-y-4">
              <p className="text-sm text-white/70">
                Aura cannot clear iPhone thumbnails remotely. Delete the album on your phone so the next sync rebuilds
                the grid artwork from your corrected files.
              </p>
              <div className="rounded-lg border border-white/10 bg-black/20 p-2">
                {IPHONE_CHECKLIST.map((text, i) => (
                  <ChecklistItem
                    key={text}
                    checked={iphoneChecks[i]}
                    onChange={(checked) =>
                      setIphoneChecks((prev) => {
                        const next = [...prev]
                        next[i] = checked
                        return next
                      })
                    }
                  >
                    {text}
                  </ChecklistItem>
                ))}
              </div>
              {iphoneDone && (
                <p className="text-xs text-green-300/90">iPhone steps marked done — continue to re-sync from your PC.</p>
              )}
            </div>
          )}

          {step === 4 && (
            <div className="space-y-4">
              <p className="text-sm text-white/70">
                Re-sync <strong className="text-white/90">only the affected album first</strong> — confirm the shelf
                thumbnail before syncing your full library.
              </p>
              <ol className="list-decimal space-y-2 pl-5 text-sm text-white/65">
                <li>{copy.syncHint}</li>
                <li>On iPhone: check Recently Added and the library grid — the cover should match the album page.</li>
                <li>If it is still wrong: delete the album on iPhone again, force-close Music, and sync that album once more.</li>
              </ol>
              <p className="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-xs text-white/50">
                Tip: Syncing one album at a time avoids re-copying a bad thumbnail index for the whole library.
              </p>
            </div>
          )}

          {feedback && (
            <p className={`mt-4 rounded-lg border px-3 py-2 text-xs ${feedbackClass}`}>{feedback.message}</p>
          )}

          <div className="mt-5 flex flex-wrap items-center justify-between gap-2 border-t border-white/10 pt-4">
            <button
              type="button"
              disabled={step === 0}
              onClick={() => setStep((s) => Math.max(0, s - 1))}
              className="rounded-lg border border-white/15 px-4 py-2 text-sm text-white/70 hover:bg-white/5 disabled:opacity-40"
            >
              Back
            </button>
            <div className="flex gap-2">
              {step < STEP_LABELS.length - 1 ? (
                <button
                  type="button"
                  onClick={() => setStep((s) => s + 1)}
                  disabled={step === 1 && !albumFolder}
                  className="rounded-lg bg-accent px-4 py-2 text-sm font-medium disabled:opacity-40"
                >
                  {step === 0 ? 'Skip intro' : 'Continue'}
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => {
                    resetGuide()
                    setOpen(false)
                  }}
                  className="rounded-lg bg-accent px-4 py-2 text-sm font-medium"
                >
                  Done
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </section>
  )
}
