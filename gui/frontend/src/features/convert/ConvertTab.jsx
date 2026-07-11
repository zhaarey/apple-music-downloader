import { useState } from 'react'
import {
  TagPickConvertSourceFile,
  TagPickConvertOutputFile,
  TagConvertToAppleAAC,
  RevealInFolder,
} from '../../wailsjs/go/main/App'
import PageShell from '../../components/PageShell'
import StatusToast from '../../components/StatusToast'
import { formatActionError } from '../../lib/formatActionError'
import { reportFrontendError } from '../../lib/errorReporting'

function basename(path) {
  const parts = String(path || '').split(/[/\\]/)
  return parts[parts.length - 1] || path
}

/**
 * Convert common audio formats to AAC-LC 256 kbps M4A for Apple Music import.
 */
export default function ConvertTab({ onOpenInTagEditor }) {
  const [sourcePath, setSourcePath] = useState('')
  const [outputPath, setOutputPath] = useState('')
  const [lastOutput, setLastOutput] = useState('')
  const [converting, setConverting] = useState(false)
  const [error, setError] = useState('')
  const [toast, setToast] = useState('')
  const [toastVariant, setToastVariant] = useState('success')

  const pickSource = async () => {
    setError('')
    try {
      const path = await TagPickConvertSourceFile()
      if (path) {
        setSourcePath(path)
        setOutputPath('')
        setLastOutput('')
      }
    } catch (e) {
      setError(formatActionError(e, 'Pick source file'))
    }
  }

  const pickOutput = async () => {
    if (!sourcePath) {
      setError('Choose a source file first.')
      return
    }
    setError('')
    try {
      const path = await TagPickConvertOutputFile(sourcePath)
      if (path) setOutputPath(path)
    } catch (e) {
      setError(formatActionError(e, 'Pick output path'))
    }
  }

  const convert = async () => {
    if (!sourcePath || converting) return
    setConverting(true)
    setError('')
    try {
      const res = await TagConvertToAppleAAC(sourcePath, outputPath || '')
      if (!res?.output_path) {
        throw new Error('Conversion returned no output path')
      }
      setLastOutput(res.output_path)
      setToastVariant('success')
      setToast(res.summary || 'Converted to AAC 256 kbps.')
    } catch (e) {
      reportFrontendError('ConvertTab.convert', e)
      setError(formatActionError(e, 'Convert to AAC'))
    } finally {
      setConverting(false)
    }
  }

  return (
    <PageShell>
      <StatusToast message={toast} variant={toastVariant} onDismiss={() => setToast('')} duration={4200} />

      <section className="rounded-xl border border-white/10 bg-surface-raised p-4">
        <h2 className="text-xl font-semibold">Convert to AAC</h2>
        <p className="mt-1 text-sm text-white/50">
          Turn MP3, FLAC, WAV, AIFF, OGG, Opus, WMA, and similar files into AAC 256 kbps .m4a so Apple Music can import
          them. Then open the result in Tag Editor to fix title, album, and artwork.
        </p>

        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={pickSource}
            disabled={converting}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium hover:bg-accent-muted disabled:opacity-50"
          >
            Choose source file…
          </button>
          <button
            type="button"
            onClick={pickOutput}
            disabled={converting || !sourcePath}
            className="rounded-lg border border-white/15 px-4 py-2 text-sm text-white/80 hover:bg-white/5 disabled:opacity-50"
          >
            Choose save location…
          </button>
        </div>

        {sourcePath && (
          <div className="mt-4 rounded-lg bg-black/20 px-3 py-2.5 text-sm text-white/70">
            <p className="truncate" title={sourcePath}>
              <span className="text-white/40">Source:</span> {basename(sourcePath)}
            </p>
            <p className="mt-1 truncate text-xs text-white/45" title={outputPath || 'Beside source as .m4a'}>
              <span className="text-white/40">Output:</span>{' '}
              {outputPath ? basename(outputPath) : 'Same folder (auto .m4a name)'}
            </p>
          </div>
        )}

        <button
          type="button"
          onClick={() => void convert()}
          disabled={converting || !sourcePath}
          className="mt-4 w-full rounded-lg border border-accent/40 bg-accent/15 py-3 text-sm font-semibold text-accent hover:bg-accent/25 disabled:opacity-50"
        >
          {converting ? 'Converting to AAC 256 kbps…' : 'Convert to AAC / M4A'}
        </button>

        {error && (
          <p className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">{error}</p>
        )}
      </section>

      {lastOutput && (
        <section className="rounded-xl border border-emerald-500/25 bg-emerald-500/10 p-4">
          <p className="text-sm font-medium text-emerald-100">Conversion complete</p>
          <p className="mt-1 truncate text-xs text-emerald-100/70" title={lastOutput}>
            {lastOutput}
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            {onOpenInTagEditor && (
              <button
                type="button"
                onClick={() => onOpenInTagEditor(lastOutput)}
                className="rounded-lg bg-accent px-3 py-2 text-sm font-medium hover:bg-accent-muted"
              >
                Open in Tag Editor
              </button>
            )}
            <button
              type="button"
              onClick={() => RevealInFolder(lastOutput)}
              className="rounded-lg border border-white/15 px-3 py-2 text-sm hover:bg-white/5"
            >
              Show in folder
            </button>
          </div>
        </section>
      )}

      <section className="rounded-xl border border-dashed border-white/10 p-4 text-sm text-white/45">
        <p className="font-medium text-white/60">Supported inputs</p>
        <p className="mt-1">MP3 · FLAC · WAV · AIFF · OGG · Opus · WMA · AAC · M4A · MP4 (audio)</p>
        <p className="mt-2">Output is always AAC-LC stereo at 256 kbps in an .m4a container.</p>
      </section>
    </PageShell>
  )
}
