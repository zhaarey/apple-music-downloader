package main

import (
	"context"
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	appconfig "main/internal/config"
	"main/internal/engine"
	"main/internal/events"
	"main/utils/structs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:     "Apple Music Downloader",
		Width:     1100,
		Height:    720,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
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
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

type App struct {
	ctx     context.Context
	eng     *engine.Engine
	mu      sync.Mutex
	running bool
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.eng = engine.New(events.FuncEmitter(func(ev events.Event) {
		runtime.EventsEmit(a.ctx, "engine:event", ev)
	}))
	if _, _, err := appconfig.InitIfMissing(); err != nil {
		_ = a.eng.LoadConfig(appconfig.DefaultConfigPath())
		return
	}
	_ = a.eng.LoadConfig(appconfig.ConfigPath())
}

func (a *App) shutdown(ctx context.Context) {
	a.eng.Cancel()
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

func (a *App) StartDownload(urls []string, quality string, singleSong, selectTracks, allArtistAlbums bool) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = true
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			a.running = false
			a.mu.Unlock()
		}()
		opts := engine.RunOptions{
			URLs:            urls,
			Quality:         quality,
			SingleSong:      singleSong,
			SelectTracks:    selectTracks,
			AllArtistAlbums: allArtistAlbums,
		}
		err := a.eng.StartDownload(opts)
		if err != nil {
			runtime.EventsEmit(a.ctx, "engine:event", events.Event{Type: events.EventError, Message: err.Error()})
		}
	}()
	return nil
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
		path = cfg.AacSaveFolder
	}
	runtime.BrowserOpenURL(a.ctx, "file:///"+filepath.ToSlash(path))
	return nil
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
