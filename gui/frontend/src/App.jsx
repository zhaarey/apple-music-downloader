import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  GetSettings,
  SaveSettings,
  CheckDependencies,
  CancelDownload,
  OpenFolder,
  PickFolder,
  GetWizardComplete,
  SetWizardComplete,
  EventsOn,
  IsDownloading,
  SpliceIsExporting,
} from './wailsjs/go/main/App'
import Wizard from './components/Wizard'
import DownloadTab from './components/DownloadTab'
import QueueTab from './components/QueueTab'
import SettingsTab from './components/SettingsTab'
import RequirementsTab from './components/RequirementsTab'
import SpliceTab from './features/splice/SpliceTab'
import MetadataTab from './features/metadata/MetadataTab'
import ErrorBoundary from './components/ErrorBoundary'
import { parseYouTubeProgress } from './lib/downloadStatus'

const WORKFLOW_TABS = [
  { id: 'apple', label: 'Apple Music' },
  { id: 'youtube', label: 'YouTube' },
  { id: 'splice', label: 'Split mix' },
  { id: 'metadata', label: 'Tag Editor' },
]

const UTILITY_TABS = [
  { id: 'activity', label: 'Activity' },
  { id: 'requirements', label: 'Requirements' },
  { id: 'settings', label: 'Settings' },
]

/** Tabs you can always open while a download runs (multitask-friendly). */
const DOWNLOAD_MULTITASK_TABS = new Set(['activity', 'splice', 'metadata', 'requirements', 'settings'])

function canSwitchTabWhileDownloading(targetTab, downloadJob) {
  if (!downloadJob?.source) return true
  if (targetTab === downloadJob.source) return true
  if (DOWNLOAD_MULTITASK_TABS.has(targetTab)) return true
  if (targetTab === 'apple' || targetTab === 'youtube') return false
  return true
}

function tabSwitchBlockedReason(targetTab, downloadJob) {
  if (!downloadJob?.source) return ''
  if (canSwitchTabWhileDownloading(targetTab, downloadJob)) return ''
  const label = downloadJob.source === 'youtube' ? 'YouTube' : 'Apple Music'
  return `A ${label} download is in progress — finish it or open Activity to monitor before switching download tabs.`
}

function TabButton({ tab, active, onClick, badge, disabled, title }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title || undefined}
      className={`rounded-lg px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-40 ${
        active ? 'bg-accent text-white shadow' : 'text-white/60 hover:bg-surface-hover hover:text-white'
      }`}
    >
      <span className="flex items-center gap-2">
        {tab.label}
        {badge && (
          <span className="relative flex h-2 w-2 shrink-0" aria-hidden>
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-accent" />
          </span>
        )}
      </span>
    </button>
  )
}

function TabPanel({ active, children }) {
  return (
    <div className={`h-full min-h-0 ${active ? 'block' : 'hidden'}`} aria-hidden={!active}>
      {children}
    </div>
  )
}

function GlobalJobBar({ downloading, downloadJob, spliceExporting, progress, onOpenActivity }) {
  if (!downloading && !spliceExporting) return null

  const downloadLabel =
    downloadJob?.source === 'youtube' ? 'YouTube download' : downloadJob?.source === 'apple' ? 'Apple Music download' : 'Download'

  return (
    <div className="border-b border-white/10 bg-surface-raised px-6 py-2">
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-white/70">
          {downloading && (
            <span>
              {downloadLabel} in progress
              {progress?.label ? ` · ${progress.label}` : ''}
            </span>
          )}
          {spliceExporting && (
            <span className={downloading ? 'text-white/50' : ''}>
              {downloading ? '·' : ''} Split mix export in progress
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {downloading && progress?.percent > 0 && (
            <span className="tabular-nums text-accent">{progress.percent}%</span>
          )}
          {downloading && (
            <button type="button" onClick={onOpenActivity} className="text-accent hover:underline">
              Open Activity
            </button>
          )}
        </div>
      </div>
      {downloading && (
        <div className="mt-1.5 h-1 overflow-hidden rounded-full bg-black/30">
          <div
            className="h-full rounded-full bg-accent transition-all duration-300"
            style={{ width: `${Math.max(progress?.percent ?? 0, 6)}%` }}
          />
        </div>
      )}
    </div>
  )
}

export default function App() {
  const [tab, setTab] = useState('apple')
  const [showWizard, setShowWizard] = useState(true)
  const [settings, setSettings] = useState(null)
  const [deps, setDeps] = useState([])
  const [logs, setLogs] = useState([])
  const [engineEvents, setEngineEvents] = useState([])
  const [downloading, setDownloading] = useState(false)
  const [downloadJob, setDownloadJob] = useState(null)
  const [spliceExporting, setSpliceExporting] = useState(false)
  const [prefillUrl, setPrefillUrl] = useState('')
  const [jobSessions, setJobSessions] = useState({ apple: null, youtube: null })
  const [spliceHandoff, setSpliceHandoff] = useState(null)
  const [navBlockHint, setNavBlockHint] = useState('')

  const syncBackendJobs = useCallback(async () => {
    const [dl, splice] = await Promise.all([IsDownloading(), SpliceIsExporting()])
    setDownloading(dl)
    setSpliceExporting(splice)
    if (!dl) setDownloadJob(null)
  }, [])

  useEffect(() => {
    GetWizardComplete().then((done) => setShowWizard(!done))
    GetSettings().then(setSettings)
    CheckDependencies().then(setDeps)
    syncBackendJobs()

    const offEngine = EventsOn('engine:event', (ev) => {
      setEngineEvents((prev) => [...prev.slice(-150), ev])
      if (ev?.message) {
        setLogs((prev) => [...prev.slice(-200), { time: new Date().toLocaleTimeString(), msg: ev.message, type: ev.type }])
      }
      if (ev?.type === 'job_complete') {
        setDownloading(false)
        setDownloadJob(null)
      }
    })

    const offSplice = EventsOn('splice:event', (ev) => {
      if (ev?.type === 'splice_progress') setSpliceExporting(true)
      if (ev?.type === 'splice_complete' || ev?.type === 'splice_error') {
        setSpliceExporting(false)
      }
    })

    const poll = setInterval(syncBackendJobs, 2500)

    return () => {
      offEngine?.()
      offSplice?.()
      clearInterval(poll)
    }
  }, [syncBackendJobs])

  const globalProgress = useMemo(
    () => (downloading ? parseYouTubeProgress(engineEvents) : null),
    [downloading, engineEvents],
  )

  const refreshDeps = () => CheckDependencies().then(setDeps)

  const handleWizardDone = async (cfg) => {
    await SaveSettings(cfg)
    setSettings(cfg)
    await SetWizardComplete(true)
    setShowWizard(false)
    refreshDeps()
  }

  const resetDownloadPipeline = () => {
    setEngineEvents([])
    setLogs([])
    setJobSessions({ apple: null, youtube: null })
  }

  const handleDownloadStart = (source) => {
    resetDownloadPipeline()
    setDownloading(true)
    setDownloadJob({ source: source || 'apple' })
    setNavBlockHint('')
  }

  const handleDownloadEnd = (result, source) => {
    setDownloading(false)
    setDownloadJob(null)
    if (result && source) {
      setJobSessions((prev) => ({ ...prev, [source]: result }))
    }
    syncBackendJobs()
  }

  const navigateTab = (targetTab) => {
    const reason = tabSwitchBlockedReason(targetTab, downloading ? downloadJob : null)
    if (reason) {
      setNavBlockHint(reason)
      return
    }
    setNavBlockHint('')
    setTab(targetTab)
  }

  const openSpliceWithHandoff = (handoff) => {
    setSpliceHandoff(handoff)
    navigateTab('splice')
  }

  const makeDownloadTabProps = (sourceMode) => ({
    settings,
    deps,
    prefillUrl: sourceMode === 'apple' ? prefillUrl : '',
    onPrefillConsumed: () => setPrefillUrl(''),
    downloading,
    onDownloadStart: () => handleDownloadStart(sourceMode),
    onDownloadEnd: (result) => handleDownloadEnd(result, sourceMode),
    engineEvents,
    jobSession: jobSessions[sourceMode],
    onClearJobSession: () => setJobSessions((prev) => ({ ...prev, [sourceMode]: null })),
    onResetPipeline: resetDownloadPipeline,
    onSplitIntoTracks: openSpliceWithHandoff,
    onSettingsChange: async (patch) => {
      const cfg = { ...settings, ...patch }
      await SaveSettings(cfg)
      setSettings(cfg)
      refreshDeps()
    },
    sourceMode,
  })

  const tabBadge = (tabId) => {
    if (downloading && downloadJob?.source === tabId) return true
    if (tabId === 'splice' && spliceExporting) return true
    if (tabId === 'activity' && downloading) return true
    return false
  }

  const tabDisabled = (tabId) => downloading && !canSwitchTabWhileDownloading(tabId, downloadJob)

  if (showWizard) {
    return <Wizard settings={settings} deps={deps} onComplete={handleWizardDone} onRefreshDeps={refreshDeps} />
  }

  return (
    <div className="flex h-screen flex-col bg-surface">
      <header className="flex items-center justify-between gap-4 border-b border-white/10 px-6 py-4">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-accent text-lg font-bold">♫</div>
          <div className="min-w-0">
            <h1 className="text-lg font-semibold tracking-tight">Aura Audio Downloader</h1>
            <p className="truncate text-xs text-white/50">Download · split · tag for Apple Music</p>
          </div>
        </div>
        <nav className="flex shrink-0 items-center gap-3">
          <div className="flex gap-1 rounded-xl bg-surface-raised p-1">
            {WORKFLOW_TABS.map((t) => (
              <TabButton
                key={t.id}
                tab={t}
                active={tab === t.id}
                badge={tabBadge(t.id)}
                disabled={tabDisabled(t.id)}
                title={tabDisabled(t.id) ? tabSwitchBlockedReason(t.id, downloadJob) : undefined}
                onClick={() => navigateTab(t.id)}
              />
            ))}
          </div>
          <div className="hidden h-6 w-px bg-white/10 sm:block" aria-hidden />
          <div className="flex gap-1 rounded-xl bg-surface-raised/70 p-1">
            {UTILITY_TABS.map((t) => (
              <TabButton
                key={t.id}
                tab={t}
                active={tab === t.id}
                badge={tabBadge(t.id)}
                onClick={() => navigateTab(t.id)}
              />
            ))}
          </div>
        </nav>
      </header>

      <GlobalJobBar
        downloading={downloading}
        downloadJob={downloadJob}
        spliceExporting={spliceExporting}
        progress={globalProgress}
        onOpenActivity={() => navigateTab('activity')}
      />

      {navBlockHint && (
        <p className="border-b border-yellow-500/20 bg-yellow-500/10 px-6 py-2 text-sm text-yellow-200">{navBlockHint}</p>
      )}

      <main className="flex min-h-0 flex-1 flex-col overflow-hidden p-6">
        <TabPanel active={tab === 'apple'}>
          <DownloadTab {...makeDownloadTabProps('apple')} />
        </TabPanel>
        <TabPanel active={tab === 'youtube'}>
          <DownloadTab {...makeDownloadTabProps('youtube')} />
        </TabPanel>
        <TabPanel active={tab === 'splice'}>
          <ErrorBoundary
            name="SpliceTab"
            title="Split mix tab crashed"
            hint="Try building tracks again. If it keeps failing, check the log file for the exact error."
            onRetry={() => setTab('splice')}
          >
            <SpliceTab handoff={spliceHandoff} onHandoffConsumed={() => setSpliceHandoff(null)} />
          </ErrorBoundary>
        </TabPanel>
        <TabPanel active={tab === 'metadata'}>
          <ErrorBoundary name="MetadataTab" title="Tag Editor crashed" onRetry={() => setTab('metadata')}>
            <MetadataTab />
          </ErrorBoundary>
        </TabPanel>

        {tab === 'activity' && (
          <QueueTab
            logs={logs}
            engineEvents={engineEvents}
            downloading={downloading}
            onCancel={CancelDownload}
            onOpenFolder={() => OpenFolder('')}
            jobSession={jobSessions[downloadJob?.source] || jobSessions.apple || jobSessions.youtube}
          />
        )}
        {tab === 'requirements' && <RequirementsTab deps={deps} onRefreshDeps={refreshDeps} />}
        {tab === 'settings' && (
          <SettingsTab
            settings={settings}
            deps={deps}
            onSave={async (cfg) => {
              await SaveSettings(cfg)
              setSettings(cfg)
              refreshDeps()
            }}
            onPickFolder={PickFolder}
            onRefreshDeps={refreshDeps}
            onShowWizard={() => setShowWizard(true)}
          />
        )}
      </main>
    </div>
  )
}
