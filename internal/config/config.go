package config

import (
	"os"
	"path/filepath"
	"runtime"

	"main/utils/structs"

	"gopkg.in/yaml.v2"
)

const AppName = "AppleMusicDownloader"

func AppDataDir() string {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, AppName)
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "."+AppName)
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
	}
	normalize(&cfg)
	return cfg
}

func InitIfMissing() (structs.ConfigSet, string, error) {
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
