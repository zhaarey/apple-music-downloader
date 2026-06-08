package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"main/utils/structs"

	"gopkg.in/yaml.v2"
)

const AppName = "AuraAudioDownloader"

// LegacyAppName is the previous product folder under %APPDATA%.
const LegacyAppName = "AppleMusicDownloader"

func appDataRoot() string {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

func AppDataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(appDataRoot(), AppName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "."+AppName)
}

func LegacyAppDataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(appDataRoot(), LegacyAppName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "."+LegacyAppName)
}

func SpliceProjectsDir() string {
	return filepath.Join(AppDataDir(), "splice-projects")
}

// MigrateLegacyAppData copies config/logs from the old Apple Music Downloader folder once.
func MigrateLegacyAppData() error {
	newDir := AppDataDir()
	oldDir := LegacyAppDataDir()
	if oldDir == newDir {
		return nil
	}
	if _, err := os.Stat(oldDir); err != nil {
		return nil
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return err
	}
	copyIfMissing := func(name string) {
		src := filepath.Join(oldDir, name)
		dst := filepath.Join(newDir, name)
		if _, err := os.Stat(dst); err == nil {
			return
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return
		}
		_ = os.WriteFile(dst, data, 0644)
	}
	copyIfMissing("config.yaml")
	copyIfMissing("wizard.json")
	oldLogs := filepath.Join(oldDir, "logs")
	newLogs := filepath.Join(newDir, "logs")
	if _, err := os.Stat(newLogs); err != nil {
		if entries, err := os.ReadDir(oldLogs); err == nil && len(entries) > 0 {
			_ = os.MkdirAll(newLogs, 0755)
			for _, ent := range entries {
				if ent.IsDir() {
					continue
				}
				data, err := os.ReadFile(filepath.Join(oldLogs, ent.Name()))
				if err != nil {
					continue
				}
				_ = os.WriteFile(filepath.Join(newLogs, ent.Name()), data, 0644)
			}
		}
	}
	return nil
}

func ConfigPath() string {
	return filepath.Join(AppDataDir(), "config.yaml")
}

func DefaultConfigPath() string {
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}
	return ConfigPath()
}

func InstallDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

func ToolsDir() string {
	return filepath.Join(InstallDir(), "tools")
}

func MP4BoxPath() string {
	if p := filepath.Join(ToolsDir(), "MP4Box.exe"); fileExists(p) {
		return p
	}
	if p := filepath.Join(ToolsDir(), "MP4Box"); fileExists(p) {
		return p
	}
	return "MP4Box"
}

func MP4DecryptPath() string {
	if p := filepath.Join(ToolsDir(), "mp4decrypt.exe"); fileExists(p) {
		return p
	}
	if p := filepath.Join(ToolsDir(), "mp4decrypt"); fileExists(p) {
		return p
	}
	return "mp4decrypt"
}

func YtDlpPath(configured string) string {
	if configured != "" && configured != "yt-dlp" {
		if fileExists(configured) {
			return configured
		}
	}
	for _, name := range []string{"yt-dlp.exe", "yt-dlp", "youtube-dl.exe", "youtube-dl"} {
		if p := filepath.Join(ToolsDir(), name); fileExists(p) {
			return p
		}
	}
	if configured != "" {
		return configured
	}
	return "yt-dlp"
}

func FFmpegPath(configured string) string {
	if configured != "" && configured != "ffmpeg" {
		if fileExists(configured) {
			return configured
		}
	}
	if p := filepath.Join(ToolsDir(), "ffmpeg.exe"); fileExists(p) {
		return p
	}
	if p := filepath.Join(ToolsDir(), "ffmpeg"); fileExists(p) {
		return p
	}
	if configured != "" {
		return configured
	}
	return "ffmpeg"
}

func resolveBinaryPath(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	if fileExists(name) {
		abs, err := filepath.Abs(name)
		if err != nil {
			return name, true
		}
		return abs, true
	}
	if resolved, err := exec.LookPath(name); err == nil {
		return resolved, true
	}
	return "", false
}

func ffprobeInDir(dir string) (string, bool) {
	for _, name := range []string{"ffprobe.exe", "ffprobe"} {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p, true
		}
	}
	return "", false
}

func FFprobePath(configured string) string {
	ffmpeg := FFmpegPath(configured)
	if resolved, ok := resolveBinaryPath(ffmpeg); ok {
		if probe, ok := ffprobeInDir(filepath.Dir(resolved)); ok {
			return probe
		}
	}
	for _, name := range []string{"ffprobe.exe", "ffprobe"} {
		if p := filepath.Join(ToolsDir(), name); fileExists(p) {
			return p
		}
	}
	if resolved, err := exec.LookPath("ffprobe"); err == nil {
		return resolved
	}
	return "ffprobe"
}

// FFmpegLocation returns the directory for yt-dlp --ffmpeg-location (must contain ffmpeg + ffprobe).
func FFmpegLocation(configured string) string {
	tools := ToolsDir()
	if _, ok := ffprobeInDir(tools); ok {
		if fileExists(filepath.Join(tools, "ffmpeg.exe")) || fileExists(filepath.Join(tools, "ffmpeg")) {
			abs, err := filepath.Abs(tools)
			if err == nil {
				return abs
			}
			return tools
		}
	}
	ffmpeg, ok := resolveBinaryPath(FFmpegPath(configured))
	if !ok {
		return FFmpegPath(configured)
	}
	dir := filepath.Dir(ffmpeg)
	if _, ok := ffprobeInDir(dir); ok {
		return dir
	}
	return dir
}

// ValidateFFmpegForYouTube ensures ffmpeg and ffprobe are available in the same folder.
func ValidateFFmpegForYouTube(configured string) error {
	ffmpeg, ffmpegOK := resolveBinaryPath(FFmpegPath(configured))
	if !ffmpegOK {
		return fmt.Errorf("ffmpeg not found — install the ffmpeg essentials build and add to PATH, or place ffmpeg.exe and ffprobe.exe in dist/tools/")
	}
	ffmpegDir := filepath.Dir(ffmpeg)
	if _, ok := ffprobeInDir(ffmpegDir); ok {
		return nil
	}
	if probe, ok := resolveBinaryPath(FFprobePath(configured)); ok {
		if filepath.Dir(probe) != ffmpegDir {
			return fmt.Errorf("ffmpeg and ffprobe must be in the same folder for YouTube downloads (copy ffprobe.exe next to ffmpeg.exe, or use dist/tools/)")
		}
		return nil
	}
	return fmt.Errorf("ffprobe not found — YouTube audio extraction requires ffprobe (included in the ffmpeg essentials build). Install ffmpeg or add ffprobe.exe to dist/tools/ next to ffmpeg.exe")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func EnsureAppDataDir() error {
	return os.MkdirAll(AppDataDir(), 0755)
}

func Load(path string) (structs.ConfigSet, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return structs.ConfigSet{}, err
	}
	var cfg structs.ConfigSet
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return structs.ConfigSet{}, err
	}
	normalize(&cfg)
	return cfg, nil
}

func Save(path string, cfg structs.ConfigSet) error {
	if path == "" {
		if err := EnsureAppDataDir(); err != nil {
			return err
		}
		path = ConfigPath()
	}
	normalize(&cfg)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func DefaultConfig() structs.ConfigSet {
	home, _ := os.UserHomeDir()
	downloads := filepath.Join(home, "Music", "Apple Music Downloads")
	cfg := structs.ConfigSet{
		Storefront:           "us",
		MediaUserToken:       "",
		AuthorizationToken:   "",
		EmbedLrc:             true,
		EmbedCover:           true,
		CoverSize:            "5000x5000",
		CoverFormat:          "jpg",
		TagSortOrder:         true,
		TagItunesID:          true,
		AlacSaveFolder:       downloads,
		AtmosSaveFolder:      filepath.Join(home, "Music", "Apple Music Atmos"),
		AacSaveFolder:        downloads,
		MVSaveFolder:         filepath.Join(home, "Music", "Apple Music Videos"),
		MaxMemoryLimit:       256,
		DecryptM3u8Port:      "127.0.0.1:10020",
		GetM3u8Port:          "127.0.0.1:20020",
		GetM3u8FromDevice:    true,
		GetM3u8Mode:          "hires",
		AacType:              "aac-lc",
		AlacMax:              192000,
		AtmosMax:             2768,
		LimitMax:             200,
		AlbumFolderFormat:    "{AlbumName}",
		PlaylistFolderFormat: "{PlaylistName}",
		SongFileFormat:       "{TrackNumber}. {SongName}",
		ArtistFolderFormat:   "{ArtistName}",
		ExplicitChoice:       "[E]",
		CleanChoice:          "[C]",
		AppleMasterChoice:    "[M]",
		MVAudioType:          "atmos",
		MVMax:                2160,
		FFmpegPath:           FFmpegPath(""),
		YtDlpPath:            YtDlpPath(""),
		YouTubeSaveFolder:    filepath.Join(home, "Music", "YouTube Downloads"),
	}
	normalize(&cfg)
	return cfg
}

func InitIfMissing() (structs.ConfigSet, string, error) {
	_ = MigrateLegacyAppData()
	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := EnsureAppDataDir(); err != nil {
			return structs.ConfigSet{}, "", err
		}
		cfg := DefaultConfig()
		if err := Save(path, cfg); err != nil {
			return structs.ConfigSet{}, "", err
		}
		return cfg, path, nil
	}
	cfg, err := Load(path)
	return cfg, path, err
}

// Normalize applies default values for empty or invalid config fields.
func Normalize(cfg *structs.ConfigSet) {
	normalize(cfg)
}

func normalize(cfg *structs.ConfigSet) {
	if len(cfg.Storefront) != 2 {
		cfg.Storefront = "us"
	}
	if cfg.AlacSaveFolder == "" {
		cfg.AlacSaveFolder = "AM-DL downloads"
	}
	if cfg.AacSaveFolder == "" {
		cfg.AacSaveFolder = "AM-DL-AAC downloads"
	}
	if cfg.AacType == "" {
		cfg.AacType = "aac-lc"
	}
	if cfg.AlacMax == 0 {
		cfg.AlacMax = 192000
	}
	if cfg.AtmosMax == 0 {
		cfg.AtmosMax = 2768
	}
	if cfg.LrcFormat == "" {
		cfg.LrcFormat = "lrc"
	}
	// Older configs often had limit-max: 0 which stripped all song names from filenames.
	if cfg.LimitMax <= 0 {
		cfg.LimitMax = 200
	}
	if cfg.MaxMemoryLimit <= 0 {
		cfg.MaxMemoryLimit = 256
	}
	if cfg.DecryptM3u8Port == "" {
		cfg.DecryptM3u8Port = "127.0.0.1:10020"
	}
	if cfg.GetM3u8Port == "" {
		cfg.GetM3u8Port = "127.0.0.1:20020"
	}
	if cfg.GetM3u8Mode == "" {
		cfg.GetM3u8Mode = "hires"
	}
	if cfg.CoverSize == "" {
		cfg.CoverSize = "5000x5000"
	}
	if cfg.CoverFormat == "" {
		cfg.CoverFormat = "jpg"
	}
	if cfg.SongFileFormat == "" {
		cfg.SongFileFormat = "{TrackNumber}. {SongName}"
	}
	if cfg.AlbumFolderFormat == "" {
		cfg.AlbumFolderFormat = "{AlbumName}"
	}
	if cfg.ArtistFolderFormat == "" {
		cfg.ArtistFolderFormat = "{ArtistName}"
	}
	if cfg.PlaylistFolderFormat == "" {
		cfg.PlaylistFolderFormat = "{PlaylistName}"
	}
	cfg.FFmpegPath = FFmpegPath(cfg.FFmpegPath)
	cfg.YtDlpPath = YtDlpPath(cfg.YtDlpPath)
	if cfg.YouTubeSaveFolder == "" {
		cfg.YouTubeSaveFolder = cfg.AacSaveFolder
	}
}

func FromExample(examplePath string) (structs.ConfigSet, error) {
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return DefaultConfig(), nil
	}
	var cfg structs.ConfigSet
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	normalize(&cfg)
	return cfg, nil
}
