import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  GetSettings,
  CheckDependencies,
  CancelDownload,
  OpenFolder,
  PickFolder,
  GetWizardComplete,
  SetWizardComplete,
  GetSetupComplete,
  SetSetupComplete,
  GetPlatform,
  EventsOn,
  IsDownloading,
  SpliceIsExporting,
  TrimIsExporting,
} from './wailsjs/go/main/App'
import Wizard from './components/Wizard'
import SetupChecklist from './components/SetupChecklist'
import DownloadTab from './components/DownloadTab'
import QueueTab from './components/QueueTab'
import SettingsTab from './components/SettingsTab'
import RequirementsTab from './components/RequirementsTab'
import SpliceTab from './features/splice/SpliceTab'
import TrimTab from './features/trim/TrimTab'
import ConvertTab from './features/convert/ConvertTab'
import MetadataTab from './features/metadata/MetadataTab'
import ErrorBoundary from './components/ErrorBoundary'
import { parseYouTubeProgress } from './lib/downloadStatus'
import { featuresForPlatform, isTabEnabled } from './config/platform'
import { persistSettings } from './lib/settings'

function canSwitchTabWhileDownloading(targetTab, downloadJob, multitaskTabs) {
  if (!downloadJob?.source) return true
  if (targetTab === downloadJob.source) return true
  if (multitaskTabs.has(targetTab)) return true
  if (targetTab === 'apple' || targetTab === 'youtube') return false
  return true
}

function tabSwitchBlockedReason(targetTab, downloadJob, multitaskTabs) {
  if (!downloadJob?.source) return ''
  if (canSwitchTabWhileDownloading(targetTab, downloadJob, multitaskTabs)) return ''
  const label = downloadJob.source === 'youtube' ? 'YouTube' : 'Apple Music'
  return `A ${label} download is in progress — finish it or wait before switching download tabs.`
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
    <div
      className={`min-h-0 flex-1 flex-col overflow-hidden ${active ? 'flex' : 'hidden'}`}
      aria-hidden={!active}
    >
      {children}
    </div>
  )
}

function GlobalJobBar({ downloading, downloadJob, spliceExporting, trimExporting, progress, onOpenActivity, showActivityLink }) {
  if (!downloading && !spliceExporting && !trimExporting) return null

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
          {trimExporting && (
            <span className={downloading || spliceExporting ? 'text-white/50' : ''}>
              {downloading || spliceExporting ? '·' : ''} Trim export in progress
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {downloading && progress?.percent > 0 && (
            <span className="tabular-nums text-accent">{progress.percent}%</span>
          )}
          {downloading && showActivityLink && (
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
  const [platform, setPlatform] = useState('windows')
  const features = useMemo(() => featuresForPlatform(platform), [platform])

  const [tab, setTab] = useState('apple')
  const [showWizard, setShowWizard] = useState(false)
  const [showSetupChecklist, setShowSetupChecklist] = useState(false)
  const [settings, setSettings] = useState(null)
  const [deps, setDeps] = useState([])
  const [logs, setLogs] = useState([])
  const [jobEvents, setJobEvents] = useState({ apple: [], youtube: [] })
  const [downloading, setDownloading] = useState(false)
  const [downloadJob, setDownloadJob] = useState(null)
  const activeDownloadSourceRef = useRef(null)
  const [spliceExporting, setSpliceExporting] = useState(false)
  const [prefillUrl, setPrefillUrl] = useState('')
  const [jobSessions, setJobSessions] = useState({ apple: null, youtube: null })
  const [spliceHandoff, setSpliceHandoff] = useState(null)
  const [trimHandoff, setTrimHandoff] = useState(null)
  const [trimExporting, setTrimExporting] = useState(false)
  const [tagEditorHandoff, setTagEditorHandoff] = useState(null)
  const [navBlockHint, setNavBlockHint] = useState('')

  const spliceEnabled = isTabEnabled(features, 'splice')
  const trimEnabled = isTabEnabled(features, 'trim')

  const syncBackendJobs = useCallback(async () => {
    const dl = await IsDownloading()
    setDownloading(dl)
    if (!dl) setDownloadJob(null)
    if (spliceEnabled) {
      setSpliceExporting(await SpliceIsExporting())
    }
    if (trimEnabled) {
      setTrimExporting(await TrimIsExporting())
    }
  }, [spliceEnabled, trimEnabled])

  useEffect(() => {
    GetPlatform().then((goos) => {
      const p = goos || 'windows'
      setPlatform(p)
      const f = featuresForPlatform(p)
      if (f.showSetupChecklist) {
        GetSetupComplete().then((done) => setShowSetupChecklist(!done))
      } else {
        setShowSetupChecklist(false)
      }
      if (f.showWizard) {
        GetWizardComplete().then((done) => setShowWizard(!done))
      } else {
        setShowWizard(false)
      }
    })
    GetSettings().then(setSettings)
    CheckDependencies().then(setDeps)
    syncBackendJobs()

    const offEngine = EventsOn('engine:event', (ev) => {
      const source = activeDownloadSourceRef.current
      if (source) {
        setJobEvents((prev) => ({
          ...prev,
          [source]: [...(prev[source] || []).slice(-150), ev],
        }))
      }
      if (ev?.message) {
        setLogs((prev) => [...prev.slice(-200), { time: new Date().toLocaleTimeString(), msg: ev.message, type: ev.type }])
      }
      if (ev?.type === 'job_complete') {
        activeDownloadSourceRef.current = null
        setDownloading(false)
        setDownloadJob(null)
      }
    })

    let offSplice
    if (spliceEnabled) {
      offSplice = EventsOn('splice:event', (ev) => {
        if (ev?.type === 'splice_progress') setSpliceExporting(true)
        if (ev?.type === 'splice_complete' || ev?.type === 'splice_error') {
          setSpliceExporting(false)
        }
      })
    }

    const offTrim = trimEnabled
      ? EventsOn('trim:event', (ev) => {
          if (ev?.type === 'trim_progress') setTrimExporting(true)
          if (ev?.type === 'trim_complete' || ev?.type === 'trim_error') {
            setTrimExporting(false)
          }
        })
      : undefined

    const poll = setInterval(syncBackendJobs, 2500)

    return () => {
      offEngine?.()
      offSplice?.()
      offTrim?.()
      clearInterval(poll)
    }
  }, [syncBackendJobs, spliceEnabled, trimEnabled])

  const activeJobEvents = useMemo(() => {
    if (downloadJob?.source) return jobEvents[downloadJob.source] || []
    return jobEvents.apple?.length ? jobEvents.apple : jobEvents.youtube || []
  }, [downloadJob, jobEvents])

  const globalProgress = useMemo(
    () => (downloading ? parseYouTubeProgress(activeJobEvents) : null),
    [downloading, activeJobEvents],
  )

  const refreshDeps = () => CheckDependencies().then(setDeps)

  const handleSaveSettings = async (updates) => {
    const merged = await persistSettings(settings, updates)
    setSettings(merged)
    refreshDeps()
    return merged
  }

  const handleWizardDone = async (cfg) => {
    await handleSaveSettings(cfg)
    await SetWizardComplete(true)
    setShowWizard(false)
    refreshDeps()
  }

  const handleSetupChecklistDone = async (cfg) => {
    await handleSaveSettings(cfg)
    await SetSetupComplete(true)
    setShowSetupChecklist(false)
    refreshDeps()
  }

  const resetDownloadPipeline = () => {
    setJobEvents({ apple: [], youtube: [] })
    setLogs([])
    setJobSessions({ apple: null, youtube: null })
  }

  const handleDownloadStart = (source) => {
    const src = source || 'apple'
    activeDownloadSourceRef.current = src
    resetDownloadPipeline()
    activeDownloadSourceRef.current = src
    setDownloading(true)
    setDownloadJob({ source: src })
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
    const reason = tabSwitchBlockedReason(targetTab, downloading ? downloadJob : null, features.multitaskTabs)
    if (reason) {
      setNavBlockHint(reason)
      return
    }
    setNavBlockHint('')
    setTab(targetTab)
  }

  const openSpliceWithHandoff = (handoff) => {
    if (!features.showSplitHandoff) return
    setSpliceHandoff(handoff)
    navigateTab('splice')
  }

  const openTrimWithHandoff = (path) => {
    if (!trimEnabled || !path) return
    setTrimHandoff(path)
    navigateTab('trim')
  }

  const openTagEditorWithHandoff = (path) => {
    if (!path) return
    setTagEditorHandoff(path)
    navigateTab('metadata')
  }

  const makeDownloadTabProps = (sourceMode) => ({
    settings,
    deps,
    platform,
    onOpenSettings: () => navigateTab('settings'),
    prefillUrl: sourceMode === 'apple' ? prefillUrl : '',
    onPrefillConsumed: () => setPrefillUrl(''),
    downloading,
    downloadActiveForThisTab: downloading && downloadJob?.source === sourceMode,
    onDownloadStart: () => handleDownloadStart(sourceMode),
    onDownloadEnd: (result) => handleDownloadEnd(result, sourceMode),
    engineEvents: jobEvents[sourceMode] || [],
    jobSession: jobSessions[sourceMode],
    onClearJobSession: () => setJobSessions((prev) => ({ ...prev, [sourceMode]: null })),
    onResetPipeline: resetDownloadPipeline,
    onSplitIntoTracks: features.showSplitHandoff ? openSpliceWithHandoff : undefined,
    onTrimFile: trimEnabled ? openTrimWithHandoff : undefined,
    onSettingsChange: handleSaveSettings,
    sourceMode,
    showAppleSearch: features.showAppleSearch,
    showLosslessQualities: features.showLosslessQualities,
  })

  const tabBadge = (tabId) => {
    if (downloading && downloadJob?.source === tabId) return true
    if (tabId === 'splice' && spliceExporting) return true
    if (tabId === 'trim' && trimExporting) return true
    if (tabId === 'activity' && downloading) return true
    return false
  }

  const tabDisabled = (tabId) =>
    downloading && !canSwitchTabWhileDownloading(tabId, downloadJob, features.multitaskTabs)

  if (showSetupChecklist && features.showSetupChecklist) {
    return (
      <SetupChecklist
        settings={settings}
        deps={deps}
        onComplete={handleSetupChecklistDone}
        onRefreshDeps={refreshDeps}
        onPickFolder={PickFolder}
      />
    )
  }

  if (showWizard && features.showWizard) {
    return <Wizard settings={settings} deps={deps} onComplete={handleWizardDone} onRefreshDeps={refreshDeps} />
  }

  return (
    <div className="flex h-screen min-h-0 flex-col overflow-hidden bg-surface">
      <header className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-white/10 px-4 py-3 sm:gap-4 sm:px-6 sm:py-4">
        <div className="flex min-w-0 flex-1 items-center gap-3 sm:flex-none">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-accent text-lg font-bold">♫</div>
          <div className="min-w-0">
            <h1 className="text-base font-semibold tracking-tight sm:text-lg">Aura Audio Downloader</h1>
            <p className="truncate text-xs text-white/50">{features.tagline}</p>
          </div>
        </div>
        <nav className="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto sm:justify-start sm:gap-3">
          <div className="flex gap-1 rounded-xl bg-surface-raised p-1">
            {features.workflowTabs.map((t) => (
              <TabButton
                key={t.id}
                tab={t}
                active={tab === t.id}
                badge={tabBadge(t.id)}
                disabled={tabDisabled(t.id)}
                title={tabDisabled(t.id) ? tabSwitchBlockedReason(t.id, downloadJob, features.multitaskTabs) : undefined}
                onClick={() => navigateTab(t.id)}
              />
            ))}
          </div>
          {features.utilityTabs.length > 0 && (
            <>
              <div className="hidden h-6 w-px bg-white/10 sm:block" aria-hidden />
              <div className="flex gap-1 rounded-xl bg-surface-raised/70 p-1">
                {features.utilityTabs.map((t) => (
                  <TabButton
                    key={t.id}
                    tab={t}
                    active={tab === t.id}
                    badge={tabBadge(t.id)}
                    onClick={() => navigateTab(t.id)}
                  />
                ))}
              </div>
            </>
          )}
        </nav>
      </header>

      <GlobalJobBar
        downloading={downloading}
        downloadJob={downloadJob}
        spliceExporting={spliceExporting}
        trimExporting={trimExporting}
        progress={globalProgress}
        onOpenActivity={() => navigateTab('activity')}
        showActivityLink={isTabEnabled(features, 'activity')}
      />

      {navBlockHint && (
        <p className="border-b border-yellow-500/20 bg-yellow-500/10 px-6 py-2 text-sm text-yellow-200">{navBlockHint}</p>
      )}

      <main className="flex min-h-0 flex-1 flex-col overflow-hidden p-4 sm:p-6">
        <TabPanel active={tab === 'apple'}>
          <DownloadTab {...makeDownloadTabProps('apple')} />
        </TabPanel>
        <TabPanel active={tab === 'youtube'}>
          <DownloadTab {...makeDownloadTabProps('youtube')} />
        </TabPanel>
        {spliceEnabled && (
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
        )}
        {trimEnabled && (
          <TabPanel active={tab === 'trim'}>
            <ErrorBoundary name="TrimTab" title="Trim tab crashed" onRetry={() => setTab('trim')}>
              <TrimTab handoff={trimHandoff} onHandoffConsumed={() => setTrimHandoff(null)} />
            </ErrorBoundary>
          </TabPanel>
        )}
        {isTabEnabled(features, 'convert') && (
          <TabPanel active={tab === 'convert'}>
            <ErrorBoundary name="ConvertTab" title="Convert tab crashed" onRetry={() => setTab('convert')}>
              <ConvertTab onOpenInTagEditor={openTagEditorWithHandoff} />
            </ErrorBoundary>
          </TabPanel>
        )}
        <TabPanel active={tab === 'metadata'}>
          <ErrorBoundary name="MetadataTab" title="Tag Editor crashed" onRetry={() => setTab('metadata')}>
            <MetadataTab
              platform={platform}
              active={tab === 'metadata'}
              handoff={tagEditorHandoff}
              onHandoffConsumed={() => setTagEditorHandoff(null)}
              onOpenInTrim={trimEnabled ? openTrimWithHandoff : undefined}
            />
          </ErrorBoundary>
        </TabPanel>

        <TabPanel active={tab === 'activity' && isTabEnabled(features, 'activity')}>
          <QueueTab
            logs={logs}
            engineEvents={activeJobEvents}
            downloading={downloading}
            onCancel={CancelDownload}
            onOpenFolder={() => {
              const session = jobSessions[downloadJob?.source] || jobSessions.apple || jobSessions.youtube
              const p = session?.outputPath || ''
              OpenFolder(p || '')
            }}
            jobSession={jobSessions[downloadJob?.source] || jobSessions.apple || jobSessions.youtube}
          />
        </TabPanel>
        <TabPanel active={tab === 'requirements' && isTabEnabled(features, 'requirements')}>
          <RequirementsTab deps={deps} onRefreshDeps={refreshDeps} />
        </TabPanel>
        <TabPanel active={tab === 'settings'}>
          <SettingsTab
            settings={settings}
            deps={deps}
            platform={platform}
            activityLogs={features.showActivityLogInSettings ? logs : []}
            onSave={handleSaveSettings}
            onPickFolder={PickFolder}
            onRefreshDeps={refreshDeps}
            onShowWizard={features.showWizard ? () => setShowWizard(true) : undefined}
            onShowSetupChecklist={features.showSetupChecklist ? () => setShowSetupChecklist(true) : undefined}
          />
        </TabPanel>
      </main>
    </div>
  )
}
