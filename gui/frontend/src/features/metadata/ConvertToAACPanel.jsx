import { useState } from 'react'
import {
  TagPickConvertSourceFile,
  TagPickConvertOutputFile,
  TagConvertToAppleAAC,
} from '../../wailsjs/go/main/App'
import { formatActionError } from '../../lib/formatActionError'
import { reportFrontendError } from '../../lib/errorReporting'

function basename(path) {
  const parts = String(path || '').split(/[/\\]/)
  return parts[parts.length - 1] || path
}

/**
 * Collapsible Tag Editor section: convert common audio formats to AAC-LC 256 kbps M4A
 * so files can be imported into Apple Music.
 */
export default function ConvertToAACPanel({ open, onToggle, disabled, onConverted }) {
  const [sourcePath, setSourcePath] = useState('')
  const [outputPath, setOutputPath] = useState('')
  const [converting, setConverting] = useState(false)
  const [error, setError] = useState('')

  const pickSource = async () => {
    setError('')
    try {
      const path = await TagPickConvertSourceFile()
      if (path) {
        setSourcePath(path)
        setOutputPath('')
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
    if (!sourcePath || converting || disabled) return
    setConverting(true)
    setError('')
    try {
      const res = await TagConvertToAppleAAC(sourcePath, outputPath || '')
      if (!res?.output_path) {
        throw new Error('Conversion returned no output path')
      }
      await onConverted?.(res)
      setSourcePath('')
      setOutputPath('')
    } catch (e) {
      reportFrontendError('ConvertToAACPanel.convert', e)
      setError(formatActionError(e, 'Convert to AAC'))
    } finally {
      setConverting(false)
    }
  }

  return (
    <section className="rounded-xl border border-white/10 bg-surface-raised">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left"
        aria-expanded={open}
      >
        <div className="min-w-0">
          <p className="text-sm font-medium">Convert to AAC for Apple Music</p>
          <p className="mt-0.5 text-xs text-white/45">
            MP3, FLAC, WAV, and other formats → AAC 256 kbps .m4a
          </p>
        </div>
        <span className={`shrink-0 text-white/40 transition-transform ${open ? 'rotate-180' : ''}`} aria-hidden>
          ▾
        </span>
      </button>

      {open && (
        <div className="space-y-3 border-t border-white/10 px-4 py-4">
          <p className="text-xs leading-relaxed text-white/50">
            Apple Music on desktop and iPhone imports AAC (.m4a) most reliably. Convert a local file here, then edit
            tags and artwork in the editor below.
          </p>

          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={pickSource}
              disabled={disabled || converting}
              className="rounded-lg bg-accent px-3 py-2 text-sm font-medium hover:bg-accent-muted disabled:opacity-50"
            >
              Choose source file…
            </button>
            <button
              type="button"
              onClick={pickOutput}
              disabled={disabled || converting || !sourcePath}
              className="rounded-lg border border-white/15 px-3 py-2 text-sm text-white/80 hover:bg-white/5 disabled:opacity-50"
            >
              Choose save location…
            </button>
          </div>

          {sourcePath && (
            <div className="rounded-lg bg-black/20 px-3 py-2 text-xs text-white/60">
              <p className="truncate" title={sourcePath}>
                <span className="text-white/40">Source:</span> {basename(sourcePath)}
              </p>
              <p className="mt-1 truncate text-white/45" title={outputPath || 'Beside source as .m4a'}>
                <span className="text-white/40">Output:</span>{' '}
                {outputPath ? basename(outputPath) : 'Same folder (auto .m4a name)'}
              </p>
            </div>
          )}

          <button
            type="button"
            onClick={() => void convert()}
            disabled={disabled || converting || !sourcePath}
            className="w-full rounded-lg border border-accent/40 bg-accent/15 py-2.5 text-sm font-semibold text-accent hover:bg-accent/25 disabled:opacity-50"
          >
            {converting ? 'Converting to AAC 256 kbps…' : 'Convert to AAC / M4A'}
          </button>

          {error && (
            <p className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-200">{error}</p>
          )}
        </div>
      )}
    </section>
  )
}
