package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"main/internal/platform"
	"main/utils/structs"

	"gopkg.in/yaml.v2"
)

const AppName = "AuraAudioDownloader"

// LegacyAppName is the previous product folder under %APPDATA%.
const LegacyAppName = "AppleMusicDownloader"

func AppDataDir() string {
	return platform.AppDataDir()
}

func LegacyAppDataDir() string {
	return platform.LegacyAppDataDir()
}

func SpliceProjectsDir() string {
	return filepath.Join(AppDataDir(), "splice-projects")
}

// MigrateLegacyAppData copies config/logs from older product folders once.
func MigrateLegacyAppData() error {
	return platform.MigrateAppDataDir()
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
		return fmt.Errorf("ffmpeg not found — install ffmpeg and add to PATH, or place ffmpeg and ffprobe in the app tools folder (dist/tools/)")
	}
	ffmpegDir := filepath.Dir(ffmpeg)
	if _, ok := ffprobeInDir(ffmpegDir); ok {
		return nil
	}
	if probe, ok := resolveBinaryPath(FFprobePath(configured)); ok {
		if filepath.Dir(probe) != ffmpegDir {
			return fmt.Errorf("ffmpeg and ffprobe must be in the same folder for YouTube downloads (install via Homebrew or copy both into tools/)")
		}
		return nil
	}
	return fmt.Errorf("ffprobe not found — YouTube downloads require ffprobe alongside ffmpeg. Install ffmpeg or add both to tools/")
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
		CoverSize:            "1200x1200",
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
	cfg.AlacSaveFolder = strings.TrimSpace(cfg.AlacSaveFolder)
	cfg.AtmosSaveFolder = strings.TrimSpace(cfg.AtmosSaveFolder)
	cfg.AacSaveFolder = strings.TrimSpace(cfg.AacSaveFolder)
	cfg.MVSaveFolder = strings.TrimSpace(cfg.MVSaveFolder)
	cfg.YouTubeSaveFolder = strings.TrimSpace(cfg.YouTubeSaveFolder)
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
		cfg.CoverSize = "1200x1200"
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
	cfg.DuplicateCheckFolders = normalizeFolderList(cfg.DuplicateCheckFolders)
}

func normalizeFolderList(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(filepath.Clean(p))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, filepath.Clean(p))
	}
	return out
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
