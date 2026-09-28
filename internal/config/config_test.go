package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.General.Storefront != "us" {
		t.Errorf("General.Storefront = %q, want 'us'", cfg.General.Storefront)
	}
	if cfg.Media.AlacMax != 192000 {
		t.Errorf("Media.AlacMax = %d, want 192000", cfg.Media.AlacMax)
	}
	if cfg.Media.AtmosMax != 2768 {
		t.Errorf("Media.AtmosMax = %d, want 2768", cfg.Media.AtmosMax)
	}
	if cfg.Media.AacType != "aac-lc" {
		t.Errorf("Media.AacType = %q, want 'aac-lc'", cfg.Media.AacType)
	}
	if cfg.Media.MV.AudioType != "atmos" {
		t.Errorf("Media.MV.AudioType = %q, want 'atmos'", cfg.Media.MV.AudioType)
	}
	if cfg.Paths.Alac != "AM-Lossless" {
		t.Errorf("Paths.Alac = %q, want 'AM-Lossless'", cfg.Paths.Alac)
	}
	if cfg.Paths.AlbumFolder != "{AlbumName}" {
		t.Errorf("Paths.AlbumFolder = %q, want '{AlbumName}'", cfg.Paths.AlbumFolder)
	}
	if cfg.Paths.PlaylistFolder != "{PlaylistName}" {
		t.Errorf("Paths.PlaylistFolder = %q, want '{PlaylistName}'", cfg.Paths.PlaylistFolder)
	}
	if cfg.Paths.ArtistFolder != "{UrlArtistName}" {
		t.Errorf("Paths.ArtistFolder = %q, want '{UrlArtistName}'", cfg.Paths.ArtistFolder)
	}
	if cfg.Paths.SongFile != "{SongNumer}. {SongName}" {
		t.Errorf("Paths.SongFile = %q, want '{SongNumer}. {SongName}'", cfg.Paths.SongFile)
	}
	if cfg.Paths.LimitMax != 200 {
		t.Errorf("Paths.LimitMax = %d, want 200", cfg.Paths.LimitMax)
	}
	if cfg.Paths.Explicit != "[E]" {
		t.Errorf("Paths.Explicit = %q, want '[E]'", cfg.Paths.Explicit)
	}
	if cfg.Paths.Clean != "[C]" {
		t.Errorf("Paths.Clean = %q, want '[C]'", cfg.Paths.Clean)
	}
	if cfg.Paths.AppleMaster != "[M]" {
		t.Errorf("Paths.AppleMaster = %q, want '[M]'", cfg.Paths.AppleMaster)
	}
	if !cfg.Metadata.Artwork.Embed {
		t.Errorf("Metadata.Artwork.Embed = %v, want true", cfg.Metadata.Artwork.Embed)
	}
	if cfg.Metadata.Lyrics.Format != "lrc" {
		t.Errorf("Metadata.Lyrics.Format = %q, want 'lrc'", cfg.Metadata.Lyrics.Format)
	}
	if cfg.Metadata.Tags.UseSongInfoForPlaylist {
		t.Errorf("Metadata.Tags.UseSongInfoForPlaylist = %v, want false", cfg.Metadata.Tags.UseSongInfoForPlaylist)
	}
	if cfg.Convert.Format != "flac" {
		t.Errorf("Convert.Format = %q, want 'flac'", cfg.Convert.Format)
	}
}

func TestLoadLayering(t *testing.T) {
	dir := t.TempDir()

	exampleContent := `
general:
  storefront: jp
  media-user-token: example-token
  lite-server: http://example.org:1234
media:
  alac-max: 192000
  atmos-max: 2768
  aac-type: aac-lc
`
	userContent := `
general:
  media-user-token: user-token
  lite-server: http://localhost:8080
media:
  alac-max: 96000
`
	exampleFile := filepath.Join(dir, "config.yaml.example")
	userFile := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(exampleFile, []byte(exampleContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte(userContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigFile:             userFile,
		ExampleFile:            exampleFile,
		DisableMissingWarnings: true,
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Overridden by user config
	if cfg.General.MediaUserToken != "user-token" {
		t.Errorf("General.MediaUserToken = %q, want 'user-token'", cfg.General.MediaUserToken)
	}
	if cfg.General.LiteServer != "http://localhost:8080" {
		t.Errorf("General.LiteServer = %q, want 'http://localhost:8080'", cfg.General.LiteServer)
	}
	if cfg.Media.AlacMax != 96000 {
		t.Errorf("Media.AlacMax = %d, want 96000", cfg.Media.AlacMax)
	}
	// Inherited from example config
	if cfg.General.Storefront != "jp" {
		t.Errorf("General.Storefront = %q, want 'jp'", cfg.General.Storefront)
	}
	if cfg.Media.AtmosMax != 2768 {
		t.Errorf("Media.AtmosMax = %d, want 2768", cfg.Media.AtmosMax)
	}
	// Inherited from in-code default
	if cfg.Metadata.Artwork.Format != "jpg" {
		t.Errorf("Metadata.Artwork.Format = %q, want 'jpg'", cfg.Metadata.Artwork.Format)
	}
}

func TestLoadWithFlagSetOverride(t *testing.T) {
	dir := t.TempDir()
	userContent := `
media:
  alac-max: 96000
general:
  lite-server: http://from-config:1234
`
	userFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(userFile, []byte(userContent), 0644); err != nil {
		t.Fatal(err)
	}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Int("alac-max", 192000, "alac max")
	fs.String("lite-server", "", "lite server")
	fs.String("unrelated-flag", "hello", "unrelated")

	// User specifies --lite-server on CLI, but does NOT specify --alac-max
	if err := fs.Parse([]string{"--lite-server=http://from-cli:9999", "--unrelated-flag=world"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigFile:             userFile,
		ExampleFile:            filepath.Join(dir, "nonexistent.example"),
		FlagSet:                fs,
		DisableMissingWarnings: true,
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Flag was changed on CLI, so it should override general.lite-server
	if cfg.General.LiteServer != "http://from-cli:9999" {
		t.Errorf("General.LiteServer = %q, want 'http://from-cli:9999'", cfg.General.LiteServer)
	}
	// Flag was NOT changed on CLI, so media.alac-max from config file (96000) must be preserved
	if cfg.Media.AlacMax != 96000 {
		t.Errorf("Media.AlacMax = %d, want 96000 (from config file, not flag default)", cfg.Media.AlacMax)
	}
}

func TestValidate(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			Storefront: "invalid-length",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}
	if cfg.General.Storefront != "us" {
		t.Errorf("General.Storefront = %q, want 'us'", cfg.General.Storefront)
	}
	if cfg.Media.AlacMax != 192000 {
		t.Errorf("Media.AlacMax = %d, want 192000", cfg.Media.AlacMax)
	}
	if cfg.Media.MV.Max != 2160 {
		t.Errorf("Media.MV.Max = %d, want 2160", cfg.Media.MV.Max)
	}
}

func TestLoadActualExampleFile(t *testing.T) {
	exampleFile := filepath.Join("..", "..", "config.yaml.example")
	cfg, err := Load(LoadOptions{
		ConfigFile:             exampleFile,
		DisableMissingWarnings: true,
	})
	if err != nil {
		t.Fatalf("Failed to load config.yaml.example: %v", err)
	}
	if cfg.General.Storefront != "us" {
		t.Errorf("General.Storefront = %q, want 'us'", cfg.General.Storefront)
	}
	if cfg.General.LiteServer != "http://127.0.0.1:12340" {
		t.Errorf("General.LiteServer = %q, want 'http://127.0.0.1:12340'", cfg.General.LiteServer)
	}
	if cfg.Media.AlacMax != 192000 {
		t.Errorf("Media.AlacMax = %d, want 192000", cfg.Media.AlacMax)
	}
	if cfg.Paths.LimitMax != 200 {
		t.Errorf("Paths.LimitMax = %d, want 200", cfg.Paths.LimitMax)
	}
	if cfg.Paths.AlbumFolder != "{AlbumName}" {
		t.Errorf("Paths.AlbumFolder = %q, want '{AlbumName}'", cfg.Paths.AlbumFolder)
	}
	if cfg.Paths.Explicit != "[E]" {
		t.Errorf("Paths.Explicit = %q, want '[E]'", cfg.Paths.Explicit)
	}
	if cfg.Metadata.Tags.UseSongInfoForPlaylist {
		t.Errorf("Metadata.Tags.UseSongInfoForPlaylist = %v, want false", cfg.Metadata.Tags.UseSongInfoForPlaylist)
	}
}

func TestLoadBackwardCompatibility(t *testing.T) {
	dir := t.TempDir()
	userContent := `
metadata:
  format:
    album-folder: "custom-album"
    playlist-folder: "custom-playlist"
    artist-folder: "custom-artist"
    song-file: "custom-song"
    limit-max: 150
    use-songinfo-for-playlist: true
  tags:
    explicit: "[EXP]"
    clean: "[CLN]"
    apple-master: "[ADM]"
`
	userFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(userFile, []byte(userContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigFile:             userFile,
		ExampleFile:            filepath.Join(dir, "nonexistent.example"),
		DisableMissingWarnings: true,
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Paths.AlbumFolder != "custom-album" {
		t.Errorf("Paths.AlbumFolder = %q, want 'custom-album'", cfg.Paths.AlbumFolder)
	}
	if cfg.Paths.PlaylistFolder != "custom-playlist" {
		t.Errorf("Paths.PlaylistFolder = %q, want 'custom-playlist'", cfg.Paths.PlaylistFolder)
	}
	if cfg.Paths.ArtistFolder != "custom-artist" {
		t.Errorf("Paths.ArtistFolder = %q, want 'custom-artist'", cfg.Paths.ArtistFolder)
	}
	if cfg.Paths.SongFile != "custom-song" {
		t.Errorf("Paths.SongFile = %q, want 'custom-song'", cfg.Paths.SongFile)
	}
	if cfg.Paths.LimitMax != 150 {
		t.Errorf("Paths.LimitMax = %d, want 150", cfg.Paths.LimitMax)
	}
	if cfg.Paths.Explicit != "[EXP]" {
		t.Errorf("Paths.Explicit = %q, want '[EXP]'", cfg.Paths.Explicit)
	}
	if cfg.Paths.Clean != "[CLN]" {
		t.Errorf("Paths.Clean = %q, want '[CLN]'", cfg.Paths.Clean)
	}
	if cfg.Paths.AppleMaster != "[ADM]" {
		t.Errorf("Paths.AppleMaster = %q, want '[ADM]'", cfg.Paths.AppleMaster)
	}
	if !cfg.Metadata.Tags.UseSongInfoForPlaylist {
		t.Errorf("Metadata.Tags.UseSongInfoForPlaylist = %v, want true", cfg.Metadata.Tags.UseSongInfoForPlaylist)
	}
}

func TestAutoCreateConfigFromDefaultTemplate(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")

	oldTemplate := DefaultConfigTemplate
	DefaultConfigTemplate = `
general:
  storefront: gb
  lite-server: http://embedded-test:9999
media:
  alac-max: 48000
`
	t.Cleanup(func() {
		DefaultConfigTemplate = oldTemplate
	})

	cfg, err := Load(LoadOptions{
		ConfigFile:             configFile,
		ExampleFile:            filepath.Join(dir, "nonexistent.example"),
		DisableMissingWarnings: true,
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify file was created on disk
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Expected config.yaml to be created, read failed: %v", err)
	}
	if !strings.Contains(string(data), "http://embedded-test:9999") {
		t.Fatalf("Created config.yaml does not match template content: %s", string(data))
	}

	if cfg.General.Storefront != "gb" {
		t.Errorf("General.Storefront = %q, want 'gb'", cfg.General.Storefront)
	}
	if cfg.General.LiteServer != "http://embedded-test:9999" {
		t.Errorf("General.LiteServer = %q, want 'http://embedded-test:9999'", cfg.General.LiteServer)
	}
	if cfg.Media.AlacMax != 48000 {
		t.Errorf("Media.AlacMax = %d, want 48000", cfg.Media.AlacMax)
	}
}

func TestAutoCreateConfigFromExampleFile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	exampleFile := filepath.Join(dir, "config.yaml.example")

	exampleContent := `
general:
  storefront: fr
  lite-server: http://disk-example:8888
media:
  alac-max: 96000
`
	if err := os.WriteFile(exampleFile, []byte(exampleContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldTemplate := DefaultConfigTemplate
	DefaultConfigTemplate = ""
	t.Cleanup(func() {
		DefaultConfigTemplate = oldTemplate
	})

	cfg, err := Load(LoadOptions{
		ConfigFile:             configFile,
		ExampleFile:            exampleFile,
		DisableMissingWarnings: true,
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Expected config.yaml to be created, read failed: %v", err)
	}
	if !strings.Contains(string(data), "http://disk-example:8888") {
		t.Fatalf("Created config.yaml does not match example file content: %s", string(data))
	}

	if cfg.General.Storefront != "fr" {
		t.Errorf("General.Storefront = %q, want 'fr'", cfg.General.Storefront)
	}
}

func TestEmbeddedTemplateWinsOverStaleExampleInWorkingDir(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	// A config.yaml.example left over from an older release.
	stale := "general:\n  storefront: fr\nmedia:\n  alac-max: 44100\n"
	if err := os.WriteFile("config.yaml.example", []byte(stale), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("config.yaml", []byte("general:\n  storefront: jp\n"), 0644); err != nil {
		t.Fatal(err)
	}

	oldTemplate := DefaultConfigTemplate
	DefaultConfigTemplate = "general:\n  storefront: gb\nmedia:\n  alac-max: 48000\n"
	t.Cleanup(func() {
		DefaultConfigTemplate = oldTemplate
	})

	cfg, err := Load(LoadOptions{DisableMissingWarnings: true})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Media.AlacMax != 48000 {
		t.Errorf("Media.AlacMax = %d, want 48000 from the embedded template", cfg.Media.AlacMax)
	}
	if cfg.General.Storefront != "jp" {
		t.Errorf("General.Storefront = %q, want 'jp' from the user config", cfg.General.Storefront)
	}
}

func TestCustomConfigMissingDoesNotAutoCreate(t *testing.T) {
	dir := t.TempDir()
	customFile := filepath.Join(dir, "custom.yaml")

	_, err := Load(LoadOptions{
		ConfigFile:             customFile,
		DisableMissingWarnings: true,
	})
	if err == nil {
		t.Fatal("Expected error for missing custom config file, got nil")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("Expected 'config file not found' error, got %v", err)
	}

	if _, statErr := os.Stat(customFile); !os.IsNotExist(statErr) {
		t.Errorf("Custom config file %s should not have been created", customFile)
	}
}

