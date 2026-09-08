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

	example := "media-user-token: example-token\nalac-save-folder: example\nproxy: socks5://127.0.0.1:1080\nalac-max: 192000\natmos-max: 2768\naac-type: aac-lc\nmv-audio-type: atmos\ncover-format: jpg\ncover-size: 5000x5000\nsong-file-format: \"{SongName}\"\nalbum-folder-format: \"{AlbumName}\"\nplaylist-folder-format: \"{PlaylistName}\"\nartist-folder-format: \"{ArtistName}\"\nlanguage: en-US\nget-m3u8-mode: hires\nlrc-format: lrc\nlrc-type: word\nlrc-extra: false\nembed-lrc: false\nsave-lrc-file: false\nembed-cover: false\ndl-albumcover-for-playlist: false\nsave-animated-artwork: false\nemby-animated-artwork: false\nalacfix: false\ntag-sort-order: false\ntag-itunes-id: false\nuse-song-info-for-playlist: false\nconvert-after-download: false\nconvert-format: \"\"\nconvert-keep-original: false\napple-master-choice: \"\"\nexplicit-choice: \"\"\nclean-choice: \"\"\nauthorization-token: \"\"\nlite-server: \"\"\n"
	user := "media-user-token: user-token\nproxy: http://127.0.0.1:7890\nlite-server: http://localhost:10020\n"
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
	if r.Config.MediaUserToken != "user-token" {
		t.Fatalf("MediaUserToken = %q, want user override", r.Config.MediaUserToken)
	}
	if r.Config.Proxy != "http://127.0.0.1:7890" {
		t.Fatalf("Proxy = %q, want user override", r.Config.Proxy)
	}
	if r.Config.LiteServer != "http://localhost:10020" {
		t.Fatalf("LiteServer = %q, want user override", r.Config.LiteServer)
	}
	if r.Config.AlacMax != 192000 {
		t.Fatalf("AlacMax = %d, want example default", r.Config.AlacMax)
	}
	if r.Config.CoverFormat != "jpg" {
		t.Fatalf("CoverFormat = %q, want example default", r.Config.CoverFormat)
	}
}
