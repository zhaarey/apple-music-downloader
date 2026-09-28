package app

import (
	"os"
	"path/filepath"
	"testing"

	"amdl/internal/config"
)

func TestLoadConfigOverridesExampleDefaults(t *testing.T) {
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	example := `
general:
  media-user-token: example-token
  proxy: socks5://127.0.0.1:1080
media:
  alac-max: 192000
  atmos-max: 2768
  aac-type: aac-lc
paths:
  alac: example
metadata:
  artwork:
    format: jpg
    size: 5000x5000
`
	user := `
general:
  media-user-token: user-token
  proxy: http://127.0.0.1:7890
  lite-server: http://localhost:10020
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml.example"), []byte(example), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(user), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewRunner(config.ConfigSet{})

	if err := r.loadConfig(); err != nil {
		t.Fatal(err)
	}
	if r.Config.General.MediaUserToken != "user-token" {
		t.Fatalf("MediaUserToken = %q, want user override", r.Config.General.MediaUserToken)
	}
	if r.Config.General.Proxy != "http://127.0.0.1:7890" {
		t.Fatalf("Proxy = %q, want user override", r.Config.General.Proxy)
	}
	if r.Config.General.LiteServer != "http://localhost:10020" {
		t.Fatalf("LiteServer = %q, want user override", r.Config.General.LiteServer)
	}
	if r.Config.Media.AlacMax != 192000 {
		t.Fatalf("AlacMax = %d, want example default", r.Config.Media.AlacMax)
	}
	if r.Config.Metadata.Artwork.Format != "jpg" {
		t.Fatalf("CoverFormat = %q, want example default", r.Config.Metadata.Artwork.Format)
	}
}
