import { useEffect, useState } from 'react'
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
} from './wailsjs/go/main/App'
import Wizard from './components/Wizard'
import DownloadTab from './components/DownloadTab'
import QueueTab from './components/QueueTab'
import SettingsTab from './components/SettingsTab'
import RequirementsTab from './components/RequirementsTab'
import SpliceTab from './features/splice/SpliceTab'
import MetadataTab from './features/metadata/MetadataTab'
import ErrorBoundary from './components/ErrorBoundary'

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

function TabButton({ tab, active, onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-lg px-4 py-2 text-sm font-medium transition ${
        active ? 'bg-accent text-white shadow' : 'text-white/60 hover:bg-surface-hover hover:text-white'
      }`}
    >
      {tab.label}
    </button>
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
  const [prefillUrl, setPrefillUrl] = useState('')
  const [jobSession, setJobSession] = useState(null)
  const [spliceHandoff, setSpliceHandoff] = useState(null)

  useEffect(() => {
    GetWizardComplete().then((done) => setShowWizard(!done))
    GetSettings().then(setSettings)
    CheckDependencies().then(setDeps)

    const off = EventsOn('engine:event', (ev) => {
      setEngineEvents((prev) => [...prev.slice(-150), ev])
      if (ev?.message) {
        setLogs((prev) => [...prev.slice(-200), { time: new Date().toLocaleTimeString(), msg: ev.message, type: ev.type }])
      }
      if (ev?.type === 'job_complete') {
        setDownloading(false)
      }
    })
    return () => off?.()
  }, [])

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
    setJobSession(null)
  }

  const handleDownloadStart = () => {
    resetDownloadPipeline()
    setDownloading(true)
  }

  const handleDownloadEnd = (result) => {
    setDownloading(false)
    if (result) setJobSession(result)
  }

  const openSpliceWithHandoff = (handoff) => {
    setSpliceHandoff(handoff)
    setTab('splice')
  }

  const downloadTabProps = {
    settings,
    deps,
    prefillUrl,
    onPrefillConsumed: () => setPrefillUrl(''),
    downloading,
    onDownloadStart: handleDownloadStart,
    onDownloadEnd: handleDownloadEnd,
    engineEvents,
    jobSession,
    onClearJobSession: () => setJobSession(null),
    onResetPipeline: resetDownloadPipeline,
    onSplitIntoTracks: openSpliceWithHandoff,
    onSettingsChange: async (patch) => {
      const cfg = { ...settings, ...patch }
      await SaveSettings(cfg)
      setSettings(cfg)
      refreshDeps()
    },
  }

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
              <TabButton key={t.id} tab={t} active={tab === t.id} onClick={() => setTab(t.id)} />
            ))}
          </div>
          <div className="hidden h-6 w-px bg-white/10 sm:block" aria-hidden />
          <div className="flex gap-1 rounded-xl bg-surface-raised/70 p-1">
            {UTILITY_TABS.map((t) => (
              <TabButton key={t.id} tab={t} active={tab === t.id} onClick={() => setTab(t.id)} />
            ))}
          </div>
        </nav>
      </header>

      <main className="flex-1 overflow-hidden p-6">
        {(tab === 'apple' || tab === 'youtube') && (
          <DownloadTab key={tab} {...downloadTabProps} sourceMode={tab} />
        )}
        {tab === 'splice' && (
          <ErrorBoundary
            name="SpliceTab"
            title="Split mix tab crashed"
            hint="Try building tracks again. If it keeps failing, check the log file for the exact error."
            onRetry={() => setTab('splice')}
          >
            <SpliceTab handoff={spliceHandoff} onHandoffConsumed={() => setSpliceHandoff(null)} />
          </ErrorBoundary>
        )}
        {tab === 'metadata' && (
          <ErrorBoundary name="MetadataTab" title="Tag Editor crashed" onRetry={() => setTab('metadata')}>
            <MetadataTab />
          </ErrorBoundary>
        )}
        {tab === 'activity' && (
          <QueueTab
            logs={logs}
            engineEvents={engineEvents}
            downloading={downloading}
            onCancel={CancelDownload}
            onOpenFolder={() => OpenFolder('')}
            jobSession={jobSession}
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
