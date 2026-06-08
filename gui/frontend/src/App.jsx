import { useEffect, useState } from 'react'
import {
  GetSettings,
  SaveSettings,
  CheckDependencies,
  Search,
  DetectURLType,
  StartDownload,
  CancelDownload,
  IsDownloading,
  PickFolder,
  OpenFolder,
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
  { id: 'queue', label: 'Queue' },
  { id: 'requirements', label: 'Requirements' },
  { id: 'settings', label: 'Settings' },
]

export default function App() {
  const [tab, setTab] = useState('download')
  const [showWizard, setShowWizard] = useState(true)
  const [settings, setSettings] = useState(null)
  const [deps, setDeps] = useState([])
  const [logs, setLogs] = useState([])
  const [downloading, setDownloading] = useState(false)
  const [prefillUrl, setPrefillUrl] = useState('')

  useEffect(() => {
    GetWizardComplete().then((done) => setShowWizard(!done))
    GetSettings().then(setSettings)
    CheckDependencies().then(setDeps)

    const off = EventsOn('engine:event', (ev) => {
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

  const handleDownload = async (opts) => {
    setDownloading(true)
    setTab('queue')
    await StartDownload(opts.urls, opts.quality, opts.singleSong, opts.selectTracks, opts.allArtistAlbums)
  }

  const handleSearchSelect = (url) => {
    setPrefillUrl(url)
    setTab('download')
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
            <p className="text-xs text-white/50">High-quality downloads for your library</p>
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
            onDownload={handleDownload}
            downloading={downloading}
          />
        )}
        {tab === 'search' && <SearchTab onSelect={handleSearchSelect} onDownload={handleDownload} />}
        {tab === 'queue' && (
          <QueueTab logs={logs} downloading={downloading} onCancel={CancelDownload} onOpenFolder={() => OpenFolder('')} />
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
