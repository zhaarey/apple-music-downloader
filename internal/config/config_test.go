package config

import (
	"os"
	"path/filepath"
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
	if cfg.Paths.Alac != "AM-DL downloads" {
		t.Errorf("Paths.Alac = %q, want 'AM-DL downloads'", cfg.Paths.Alac)
	}
	if !cfg.Metadata.Artwork.Embed {
		t.Errorf("Metadata.Artwork.Embed = %v, want true", cfg.Metadata.Artwork.Embed)
	}
	if cfg.Metadata.Lyrics.Format != "lrc" {
		t.Errorf("Metadata.Lyrics.Format = %q, want 'lrc'", cfg.Metadata.Lyrics.Format)
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
	if cfg.Metadata.Format.LimitMax != 200 {
		t.Errorf("Metadata.Format.LimitMax = %d, want 200", cfg.Metadata.Format.LimitMax)
	}
}
