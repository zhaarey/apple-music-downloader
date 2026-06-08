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
import SearchTab from './components/SearchTab'
import QueueTab from './components/QueueTab'
import SettingsTab from './components/SettingsTab'
import RequirementsTab from './components/RequirementsTab'

const TABS = [
  { id: 'download', label: 'Download' },
  { id: 'search', label: 'Search' },
  { id: 'activity', label: 'Activity' },
  { id: 'requirements', label: 'Requirements' },
  { id: 'settings', label: 'Settings' },
]

export default function App() {
  const [tab, setTab] = useState('download')
  const [showWizard, setShowWizard] = useState(true)
  const [settings, setSettings] = useState(null)
  const [deps, setDeps] = useState([])
  const [logs, setLogs] = useState([])
  const [engineEvents, setEngineEvents] = useState([])
  const [downloading, setDownloading] = useState(false)
  const [prefillUrl, setPrefillUrl] = useState('')
  const [jobSession, setJobSession] = useState(null)

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

  const handleSearchPreview = (url) => {
    setPrefillUrl(url)
    setTab('download')
  }

  const handleDownloadStart = () => {
    setDownloading(true)
    setJobSession(null)
  }

  const handleDownloadEnd = (result) => {
    setDownloading(false)
    if (result) setJobSession(result)
  }

  if (showWizard) {
    return <Wizard settings={settings} deps={deps} onComplete={handleWizardDone} onRefreshDeps={refreshDeps} />
  }

  return (
    <div className="flex h-screen flex-col bg-surface">
      <header className="flex items-center justify-between border-b border-white/10 px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent text-lg font-bold">♫</div>
          <div>
            <h1 className="text-lg font-semibold tracking-tight">Apple Music Downloader</h1>
            <p className="text-xs text-white/50">Fetch, preview, and download your music</p>
          </div>
        </div>
        <nav className="flex gap-1 rounded-xl bg-surface-raised p-1">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`rounded-lg px-4 py-2 text-sm font-medium transition ${
                tab === t.id ? 'bg-accent text-white shadow' : 'text-white/60 hover:text-white hover:bg-surface-hover'
              }`}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </header>

      <main className="flex-1 overflow-hidden p-6">
        {tab === 'download' && (
          <DownloadTab
            settings={settings}
            deps={deps}
            prefillUrl={prefillUrl}
            onPrefillConsumed={() => setPrefillUrl('')}
            downloading={downloading}
            onDownloadStart={handleDownloadStart}
            onDownloadEnd={handleDownloadEnd}
            engineEvents={engineEvents}
            jobSession={jobSession}
            onClearJobSession={() => setJobSession(null)}
          />
        )}
        {tab === 'search' && <SearchTab onPreview={handleSearchPreview} />}
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
