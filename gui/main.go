package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	appconfig "main/internal/config"
	"main/internal/engine"
	"main/internal/events"
	"main/internal/logging"
	"main/internal/osutil"
	"main/internal/platform"
	"main/internal/splice"
	"main/utils/structs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	_ = logging.Init()
	logging.InstallGlobalPanicHandler()

	app := NewApp()
	err := wails.Run(&options.App{
		Title:     "Aura Audio Downloader",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: localMediaMiddleware,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 20, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			Theme:              windows.Dark,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
	})
	if err != nil {
		logging.Error("Wails exited with error: %v", err)
		println("Error:", err.Error())
	}
}

func (a *App) emitEngineEvent(ev events.Event) {
	logging.Info("[%s] %s", ev.Type, ev.Message)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "engine:event", ev)
	}
}

func (a *App) emitSpliceEvent(ev events.Event) {
	logging.Info("[splice:%s] %s", ev.Type, ev.Message)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "splice:event", ev)
	}
}

type App struct {
	ctx     context.Context
	eng     *engine.Engine
	splice  *splice.Service
	mu      sync.Mutex
	running bool
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.eng = engine.New(events.FuncEmitter(a.emitEngineEvent))
	if _, _, err := appconfig.InitIfMissing(); err != nil {
		_ = a.eng.LoadConfig(appconfig.DefaultConfigPath())
		return
	}
	_ = a.eng.LoadConfig(appconfig.ConfigPath())
}

func (a *App) shutdown(ctx context.Context) {
	a.eng.Cancel()
}

func (a *App) GetPlatform() string {
	return platform.GOOS()
}

func (a *App) GetSettings() structs.ConfigSet {
	return a.eng.GetConfig()
}

func (a *App) SaveSettings(cfg structs.ConfigSet) error {
	return a.eng.SaveConfig(appconfig.ConfigPath(), cfg)
}

func (a *App) CheckDependencies() []engine.DependencyStatus {
	return a.eng.CheckDependencies()
}

func (a *App) Search(queryType, query string, offset int) map[string]interface{} {
	hits, hasNext, err := a.eng.Search(queryType, query, 15, offset)
	if err != nil {
		return map[string]interface{}{"error": err.Error(), "hits": []engine.SearchHit{}, "hasNext": false}
	}
	return map[string]interface{}{"hits": hits, "hasNext": hasNext}
}

func (a *App) DetectURLType(url string) string {
	return a.eng.DetectURLType(url)
}

func (a *App) PreviewURL(url string) engine.PreviewResult {
	return a.eng.PreviewURL(url)
}

func (a *App) StartDownloadJob(url string, quality string, selectedTrackNums []int, childURLs []string, youtubeDeliveryMode string, youtubeMeta []engine.YouTubeDownloadMeta) error {
	opts := engine.RunOptions{
		URLs:                []string{url},
		Quality:             quality,
		SelectedTrackNums:   selectedTrackNums,
		ChildURLs:           childURLs,
		YouTubeDeliveryMode: youtubeDeliveryMode,
		YouTubeMeta:         youtubeMeta,
	}
	if err := a.eng.ValidateDownloadRequest(opts); err != nil {
		return err
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("a download is already running")
	}
	a.running = true
	a.mu.Unlock()

	logging.Info("StartDownloadJob quality=%s url=%s tracks=%v childURLs=%d delivery=%s", quality, url, selectedTrackNums, len(childURLs), youtubeDeliveryMode)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				msg := logging.LogPanic("StartDownloadJob", r)
				a.emitEngineEvent(events.Event{
					Type:    events.EventError,
					Message: "The app hit an unexpected error during download. Details were saved to: " + logging.Path(),
				})
				a.emitEngineEvent(events.Event{Type: events.EventLog, Message: msg})
				a.emitEngineEvent(events.Event{
					Type:    events.EventJobComplete,
					Message: "Download stopped due to an unexpected error",
					Phase:   "failed",
				})
			}
			a.mu.Lock()
			a.running = false
			a.mu.Unlock()
		}()

		if err := a.eng.StartDownload(opts); err != nil {
			logging.Error("StartDownloadJob failed: %v", err)
			a.emitEngineEvent(events.Event{Type: events.EventError, Message: err.Error()})
			a.emitEngineEvent(events.Event{
				Type:    events.EventJobComplete,
				Message: err.Error(),
				Phase:   "failed",
			})
		}
	}()
	return nil
}

func (a *App) StartBulkDownloadJob(entries []engine.BulkQueueEntry, quality string) error {
	urls := make([]string, len(entries))
	for i, ent := range entries {
		urls[i] = ent.URL
	}
	opts := engine.RunOptions{
		URLs:        urls,
		Quality:     quality,
		BulkEntries: entries,
	}
	if err := a.eng.ValidateDownloadRequest(opts); err != nil {
		return err
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("a download is already running")
	}
	a.running = true
	a.mu.Unlock()

	logging.Info("StartBulkDownloadJob quality=%s entries=%d", quality, len(entries))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				msg := logging.LogPanic("StartBulkDownloadJob", r)
				a.emitEngineEvent(events.Event{
					Type:    events.EventError,
					Message: "The app hit an unexpected error during download. Details were saved to: " + logging.Path(),
				})
				a.emitEngineEvent(events.Event{Type: events.EventLog, Message: msg})
				a.emitEngineEvent(events.Event{
					Type:    events.EventJobComplete,
					Message: "Download stopped due to an unexpected error",
					Phase:   "failed",
				})
			}
			a.mu.Lock()
			a.running = false
			a.mu.Unlock()
		}()

		if err := a.eng.StartDownload(opts); err != nil {
			logging.Error("StartBulkDownloadJob failed: %v", err)
			a.emitEngineEvent(events.Event{Type: events.EventError, Message: err.Error()})
			a.emitEngineEvent(events.Event{
				Type:    events.EventJobComplete,
				Message: err.Error(),
				Phase:   "failed",
			})
		}
	}()
	return nil
}

// StartDownload is kept for backwards compatibility with older frontend stubs.
func (a *App) StartDownload(urls []string, quality string, singleSong, selectTracks, allArtistAlbums bool) error {
	url := ""
	if len(urls) > 0 {
		url = urls[0]
	}
	return a.StartDownloadJob(url, quality, nil, nil, "audio", nil)
}

func (a *App) CancelDownload() {
	a.eng.Cancel()
}

func (a *App) IsDownloading() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func (a *App) PickFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select download folder",
	})
}

func (a *App) OpenFolder(path string) error {
	if path == "" {
		cfg := a.eng.GetConfig()
		if cfg.YouTubeMode && cfg.YouTubeSaveFolder != "" {
			path = cfg.YouTubeSaveFolder
		} else {
			path = cfg.AacSaveFolder
		}
	}
	runtime.BrowserOpenURL(a.ctx, "file:///"+filepath.ToSlash(path))
	return nil
}

func (a *App) RevealInFolder(filePath string) error {
	return osutil.RevealInFileManager(filePath)
}

func (a *App) GetWizardComplete() bool {
	data, err := os.ReadFile(filepath.Join(appconfig.AppDataDir(), "wizard.json"))
	if err != nil {
		return false
	}
	var state struct {
		Complete bool `json:"complete"`
	}
	_ = json.Unmarshal(data, &state)
	return state.Complete
}

func (a *App) SetWizardComplete(complete bool) error {
	_ = appconfig.EnsureAppDataDir()
	data, _ := json.Marshal(map[string]bool{"complete": complete})
	return os.WriteFile(filepath.Join(appconfig.AppDataDir(), "wizard.json"), data, 0644)
}

func (a *App) GetConfigPath() string {
	return appconfig.ConfigPath()
}

func (a *App) GetAppDataDir() string {
	return appconfig.AppDataDir()
}

func (a *App) GetLogPath() string {
	return logging.Path()
}

func (a *App) OpenLogFile() error {
	runtime.BrowserOpenURL(a.ctx, "file:///"+filepath.ToSlash(logging.Path()))
	return nil
}

// LogFrontendError records a UI error to the app log file.
func (a *App) LogFrontendError(source, message, detail string) {
	if source == "" {
		source = "frontend"
	}
	logging.Error("[frontend:%s] %s", source, message)
	if detail != "" {
		logging.Error("[frontend:%s detail] %s", source, detail)
	}
}
