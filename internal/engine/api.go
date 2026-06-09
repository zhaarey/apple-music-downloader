package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appconfig "main/internal/config"
	"main/internal/events"
	"main/utils/ampapi"
	"main/utils/structs"
)

type Engine struct {
	mu      sync.Mutex
	emitter events.Emitter
	ctx     context.Context
	cancel  context.CancelFunc
}

type RunOptions struct {
	URLs              []string
	Quality           string // alac, aac, atmos, youtube
	SingleSong        bool
	SelectTracks      bool
	SelectedTrackNums []int
	ChildURLs         []string
	AllArtistAlbums   bool
	YouTubeSaveVideo  bool
	YouTubeMeta       []YouTubeDownloadMeta
	Debug             bool
	PrintJSON         bool
}

type DependencyStatus struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail"`
	Required bool  `json:"required"`
}

type SearchHit struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
	URL    string `json:"url"`
	ID     string `json:"id"`
	ArtURL string `json:"art_url,omitempty"`
}

func New(emitter events.Emitter) *Engine {
	if emitter == nil {
		emitter = events.CLIEmitter{}
	}
	return &Engine{emitter: emitter}
}

func (e *Engine) SetEmitter(emitter events.Emitter) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.emitter = emitter
}

func (e *Engine) emit(ev events.Event) {
	e.mu.Lock()
	em := e.emitter
	e.mu.Unlock()
	if em != nil {
		em.Emit(ev)
	}
}

func (e *Engine) log(msg string) {
	e.emit(events.Event{Type: events.EventLog, Message: msg})
}

func (e *Engine) logError(msg string) {
	e.emit(events.Event{Type: events.EventError, Message: msg})
}

func (e *Engine) LoadConfig(path string) error {
	cfg, err := appconfig.Load(path)
	if err != nil {
		if _, _, initErr := appconfig.InitIfMissing(); initErr == nil {
			cfg, err = appconfig.Load(appconfig.ConfigPath())
		}
		if err != nil {
			return err
		}
	}
	Config = cfg
	Config.FFmpegPath = appconfig.FFmpegPath(Config.FFmpegPath)
	Config.YtDlpPath = appconfig.YtDlpPath(Config.YtDlpPath)
	appconfig.Normalize(&Config)
	return nil
}

func (e *Engine) GetConfig() structs.ConfigSet {
	cfg := Config
	appconfig.Normalize(&cfg)
	return cfg
}

func (e *Engine) SaveConfig(path string, cfg structs.ConfigSet) error {
	appconfig.Normalize(&cfg)
	if err := appconfig.Save(path, cfg); err != nil {
		return err
	}
	Config = cfg
	return nil
}

func (e *Engine) Cancel() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Engine) CheckDependencies() []DependencyStatus {
	mp4box := appconfig.MP4BoxPath()
	_, mp4boxErr := exec.LookPath(mp4box)
	if mp4boxErr != nil {
		if _, statErr := os.Stat(mp4box); statErr == nil {
			mp4boxErr = nil
		}
	}

	ffmpeg := appconfig.FFmpegPath(Config.FFmpegPath)
	_, ffmpegErr := exec.LookPath(ffmpeg)
	if ffmpegErr != nil {
		if _, statErr := os.Stat(ffmpeg); statErr == nil {
			ffmpegErr = nil
		}
	}
	ffmpegBundleErr := appconfig.ValidateFFmpegForYouTube(Config.FFmpegPath)

	ytdlp := appconfig.YtDlpPath(Config.YtDlpPath)
	_, ytdlpErr := exec.LookPath(ytdlp)
	if ytdlpErr != nil {
		if _, statErr := os.Stat(ytdlp); statErr == nil {
			ytdlpErr = nil
		}
	}

	mp4decrypt := appconfig.MP4DecryptPath()
	_, mp4decErr := exec.LookPath(mp4decrypt)
	if mp4decErr != nil {
		if _, statErr := os.Stat(mp4decrypt); statErr == nil {
			mp4decErr = nil
		}
	}

	wrapperDecrypt := probePort(Config.DecryptM3u8Port)
	wrapperM3u8 := probePort(Config.GetM3u8Port)

	return []DependencyStatus{
		{Name: "MP4Box", OK: mp4boxErr == nil, Detail: mp4box, Required: true},
		{Name: "ffmpeg", OK: ffmpegErr == nil, Detail: ffmpeg, Required: false},
		{Name: "ffprobe (YouTube)", OK: ffmpegBundleErr == nil, Detail: appconfig.FFmpegLocation(Config.FFmpegPath), Required: false},
		{Name: "yt-dlp", OK: ytdlpErr == nil, Detail: ytdlp, Required: false},
		{Name: "mp4decrypt", OK: mp4decErr == nil, Detail: mp4decrypt, Required: false},
		{Name: "wrapper (decrypt)", OK: wrapperDecrypt, Detail: Config.DecryptM3u8Port, Required: false},
		{Name: "wrapper (m3u8)", OK: wrapperM3u8, Detail: Config.GetM3u8Port, Required: false},
	}
}

func probePort(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (e *Engine) Search(queryType, query string, limit, offset int) ([]SearchHit, bool, error) {
	token, err := e.getToken()
	if err != nil {
		return nil, false, err
	}
	apiType := queryType + "s"
	resp, err := ampapi.Search(Config.Storefront, query, apiType, Config.Language, token, limit, offset)
	if err != nil {
		return nil, false, err
	}
	var hits []SearchHit
	hasNext := false
	switch queryType {
	case "album":
		if resp.Results.Albums != nil {
			for _, item := range resp.Results.Albums.Data {
				year := ""
				if len(item.Attributes.ReleaseDate) >= 4 {
					year = item.Attributes.ReleaseDate[:4]
				}
				hits = append(hits, SearchHit{
					Type: "Album", Name: item.Attributes.Name,
					Detail: fmt.Sprintf("%s (%s, %d tracks)", item.Attributes.ArtistName, year, item.Attributes.TrackCount),
					URL: item.Attributes.URL, ID: item.ID,
					ArtURL: strings.Replace(item.Attributes.Artwork.URL, "{w}x{h}", "300x300", 1),
				})
			}
			hasNext = resp.Results.Albums.Next != ""
		}
	case "song":
		if resp.Results.Songs != nil {
			for _, item := range resp.Results.Songs.Data {
				hits = append(hits, SearchHit{
					Type: "Song", Name: item.Attributes.Name,
					Detail: fmt.Sprintf("%s — %s", item.Attributes.ArtistName, item.Attributes.AlbumName),
					URL: item.Attributes.URL, ID: item.ID,
					ArtURL: strings.Replace(item.Attributes.Artwork.URL, "{w}x{h}", "300x300", 1),
				})
			}
			hasNext = resp.Results.Songs.Next != ""
		}
	case "artist":
		if resp.Results.Artists != nil {
			for _, item := range resp.Results.Artists.Data {
				detail := strings.Join(item.Attributes.GenreNames, ", ")
				hits = append(hits, SearchHit{
					Type: "Artist", Name: item.Attributes.Name, Detail: detail,
					URL: item.Attributes.URL, ID: item.ID,
				})
			}
			hasNext = resp.Results.Artists.Next != ""
		}
	}
	return hits, hasNext, nil
}

// IsAppleMusicURL reports whether raw is an Apple Music catalog link.
func IsAppleMusicURL(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	return strings.Contains(raw, "music.apple.com") || strings.Contains(raw, "classical.music.apple.com")
}

func (e *Engine) DetectURLType(raw string) string {
	if IsYouTubeURL(raw) {
		if strings.Contains(raw, "list=") {
			return "YouTube Playlist"
		}
		return "YouTube Video"
	}
	if strings.Contains(raw, "/music-video/") {
		return "Music Video"
	}
	if strings.Contains(raw, "/song/") {
		return "Song"
	}
	if strings.Contains(raw, "/album/") {
		return "Album"
	}
	if strings.Contains(raw, "/playlist/") {
		return "Playlist"
	}
	if strings.Contains(raw, "/station/") {
		return "Station"
	}
	if strings.Contains(raw, "/artist/") {
		return "Artist"
	}
	return "Unknown"
}

func (e *Engine) StartDownload(opts RunOptions) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("download crashed unexpectedly: %v (details written to log file)", r)
			e.logError(err.Error())
		}
	}()

	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
	}
	e.ctx, e.cancel = context.WithCancel(context.Background())
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.cancel = nil
		e.mu.Unlock()
	}()

	if err := e.applyOptions(opts); err != nil {
		return err
	}
	if err := e.validateDownload(opts); err != nil {
		return err
	}
	if useYouTubePipeline(opts) {
		e.runYouTubeDownload(opts)
		return nil
	}
	token, err := e.getToken()
	if err != nil {
		return formatTokenError(err)
	}
	urls := opts.URLs
	if len(opts.ChildURLs) > 0 {
		urls = opts.ChildURLs
	} else if len(urls) == 1 && strings.Contains(urls[0], "/artist/") && opts.AllArtistAlbums {
		artist_select = true
		urlArtistName, urlArtistID, err := getUrlArtistName(urls[0], token)
		if err != nil {
			return err
		}
		Config.ArtistFolderFormat = strings.NewReplacer(
			"{UrlArtistName}", LimitString(urlArtistName),
			"{ArtistId}", urlArtistID,
		).Replace(Config.ArtistFolderFormat)
		albumArgs, err := checkArtist(urls[0], token, "albums")
		if err != nil {
			return err
		}
		mvArgs, _ := checkArtist(urls[0], token, "music-videos")
		urls = append(albumArgs, mvArgs...)
		artist_select = false
	}

	e.runDownloadLoop(urls, token, opts)
	return nil
}

func (e *Engine) ValidateDownloadRequest(opts RunOptions) error {
	return e.validateDownload(opts)
}

func (e *Engine) applyOptions(opts RunOptions) error {
	appconfig.Normalize(&Config)
	dl_atmos = false
	dl_aac = false
	dl_song = opts.SingleSong
	guiSelectedTracks = nil
	if len(opts.SelectedTrackNums) > 0 {
		dl_select = true
		guiSelectedTracks = append([]int(nil), opts.SelectedTrackNums...)
	} else {
		dl_select = opts.SelectTracks
	}
	debug_mode = opts.Debug
	print_json = opts.PrintJSON
	artist_select = opts.AllArtistAlbums

	aacVal := effectiveAacType()
	aac_type = &aacVal
	Config.AacType = aacVal

	switch opts.Quality {
	case "youtube":
		// YouTube tab / quality — pipeline selected via useYouTubePipeline.
	case "atmos":
		dl_atmos = true
	case "aac":
		dl_aac = true
		// GUI "AAC" always uses the in-process AAC-LC (Widevine) path — no wrapper needed.
		Config.AacType = "aac-lc"
		aacVal = "aac-lc"
		aac_type = &aacVal
	default:
		// ALAC default
	}
	return nil
}

func (e *Engine) getToken() (string, error) {
	token, err := ampapi.GetToken()
	if err != nil {
		if Config.AuthorizationToken != "" && Config.AuthorizationToken != "your-authorization-token" {
			token = strings.Replace(Config.AuthorizationToken, "Bearer ", "", -1)
			return token, nil
		}
		return "", formatTokenError(err)
	}
	return token, nil
}

func jobCompletePhase() string {
	if counter.Success == 0 && (counter.Error > 0 || counter.Unavailable > 0 || counter.Total > 0) {
		return "failed"
	}
	if counter.Error > 0 || counter.Unavailable > 0 {
		return "partial"
	}
	return "success"
}

func (e *Engine) runDownloadLoop(urls []string, token string, opts RunOptions) {
	currentEmitter = e.emitter
	defer func() { currentEmitter = nil }()

	counter = structs.Counter{}
	AddedTracks = nil
	okDict = make(map[string][]int)

	e.emit(events.Event{
		Type:    events.EventJobStart,
		Message: fmt.Sprintf("Starting download (%s quality)", opts.Quality),
		Phase:   opts.Quality,
	})

	albumTotal := len(urls)
	for albumNum, urlRaw := range urls {
		select {
		case <-e.ctx.Done():
			e.log("Download cancelled.")
			e.emit(events.Event{
				Type:    events.EventJobComplete,
				Message: fmt.Sprintf("Cancelled — %d completed before stop", counter.Success),
				Phase:   "cancelled",
				Success: counter.Success,
				Error:   counter.Error + counter.Unavailable,
				Total_:  counter.Total,
			})
			return
		default:
		}
		e.log(fmt.Sprintf("Queue %d of %d: %s", albumNum+1, albumTotal, e.DetectURLType(urlRaw)))

		if strings.Contains(urlRaw, "/music-video/") {
			e.processMV(urlRaw, token, opts)
			continue
		}
		if strings.Contains(urlRaw, "/song/") {
			storefront, songId := checkUrlSong(urlRaw)
			if storefront == "" || songId == "" {
				e.log("Invalid song URL format.")
				continue
			}
			_ = ripSong(songId, token, storefront, Config.MediaUserToken)
			continue
		}

		parse, err := url.Parse(urlRaw)
		if err != nil {
			e.emit(events.Event{Type: events.EventError, Message: err.Error()})
			continue
		}
		urlArgI := parse.Query().Get("i")

		if strings.Contains(urlRaw, "/album/") {
			storefront, albumId := checkUrl(urlRaw)
			_ = ripAlbum(albumId, token, storefront, Config.MediaUserToken, urlArgI)
		} else if strings.Contains(urlRaw, "/playlist/") {
			storefront, playlistId := checkUrlPlaylist(urlRaw)
			_ = ripPlaylist(playlistId, token, storefront, Config.MediaUserToken)
		} else if strings.Contains(urlRaw, "/station/") {
			storefront, stationId := checkUrlStation(urlRaw)
			if len(Config.MediaUserToken) <= 50 {
				e.log("media-user-token is not set, skip station dl")
				continue
			}
			_ = ripStation(stationId, token, storefront, Config.MediaUserToken)
		} else {
			e.log("Invalid URL type")
		}
	}

	msg := fmt.Sprintf("Finished: %d succeeded, %d failed, %d unavailable (of %d attempted)",
		counter.Success, counter.Error, counter.Unavailable, counter.Total)
	e.emit(events.Event{
		Type:    events.EventJobComplete,
		Message: msg,
		Phase:   jobCompletePhase(),
		Success: counter.Success,
		Error:   counter.Error + counter.Unavailable,
		Total_:  counter.Total,
	})

	if opts.PrintJSON {
		jsonOutput, err := json.Marshal(AddedTracks)
		if err == nil {
			e.log(string(jsonOutput))
		}
	}
}

func (e *Engine) processMV(urlRaw, token string, opts RunOptions) {
	if opts.Debug {
		return
	}
	counter.Total++
	if len(Config.MediaUserToken) <= 50 {
		e.log("media-user-token is not set, skip MV dl")
		counter.Success++
		return
	}
	mp4decrypt := appconfig.MP4DecryptPath()
	if _, err := exec.LookPath(mp4decrypt); err != nil {
		if _, statErr := os.Stat(mp4decrypt); statErr != nil {
			e.log("mp4decrypt is not found, skip MV dl")
			counter.Success++
			return
		}
	}
	mvSaveDir := strings.NewReplacer(
		"{ArtistName}", "", "{UrlArtistName}", "", "{ArtistId}", "",
	).Replace(Config.ArtistFolderFormat)
	if mvSaveDir != "" {
		mvSaveDir = filepath.Join(Config.MVSaveFolder, forbiddenNames.ReplaceAllString(mvSaveDir, "_"))
	} else {
		mvSaveDir = Config.MVSaveFolder
	}
	storefront, albumId := checkUrlMv(urlRaw)
	if err := mvDownloader(albumId, mvSaveDir, token, storefront, Config.MediaUserToken, nil); err != nil {
		e.emit(events.Event{Type: events.EventError, Message: err.Error()})
		counter.Error++
		return
	}
	counter.Success++
}

func RunCLI() {
	eng := New(events.CLIEmitter{})
	if err := eng.LoadConfig(appconfig.DefaultConfigPath()); err != nil {
		fmt.Printf("load Config failed: %v", err)
		return
	}
	runCLIWithEngine(eng)
}

func runCLIWithEngine(eng *Engine) {
	// Delegates to existing flag parsing in engine.go
	runCLIParsed(eng)
}
