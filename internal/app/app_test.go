package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"amdl/internal/config"
	"amdl/internal/download"
)

func TestCheckURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		kind       string
		storefront string
		id         string
	}{
		{name: "album", url: "https://music.apple.com/us/album/example/1234567890", kind: "album", storefront: "us", id: "1234567890"},
		{name: "album query", url: "https://music.apple.com/jp/album/example/1234567890?i=9876543210", kind: "album", storefront: "jp", id: "1234567890"},
		{name: "song id", url: "https://music.apple.com/gb/song/id9876543210", kind: "song", storefront: "gb", id: "9876543210"},
		{name: "playlist", url: "https://music.apple.com/de/playlist/example/pl.abcdef123456", kind: "playlist", storefront: "de", id: "pl.abcdef123456"},
		{name: "station", url: "https://music.apple.com/us/station/example/ra.abcdef123456", kind: "station", storefront: "us", id: "ra.abcdef123456"},
		{name: "music video", url: "https://music.apple.com/us/music-video/example/1234567890", kind: "mv", storefront: "us", id: "1234567890"},
		{name: "artist", url: "https://music.apple.com/us/artist/example/1234567890", kind: "artist", storefront: "us", id: "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storefront, id := checkUrl(tt.url, tt.kind)
			if storefront != tt.storefront {
				t.Fatalf("storefront = %q, want %q", storefront, tt.storefront)
			}
			if id != tt.id {
				t.Fatalf("id = %q, want %q", id, tt.id)
			}
		})
	}
}

func TestAudioScore(t *testing.T) {
	tests := []struct {
		name      string
		groupID   string
		preferred string
		want      int
	}{
		{name: "atmos default", groupID: "audio-atmos", preferred: "atmos", want: 10000},
		{name: "atmos rejected for aac", groupID: "audio-atmos", preferred: "aac", want: -1},
		{name: "ac3 preferred", groupID: "audio-ac3", preferred: "ac3", want: 10000},
		{name: "stereo bitrate", groupID: "audio-stereo-256", preferred: "aac", want: 8256},
		{name: "he stereo bitrate", groupID: "audio-HE-stereo-64", preferred: "aac", want: 7064},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := audioScore(tt.groupID, tt.preferred); got != tt.want {
				t.Fatalf("r.audioScore(%q, %q) = %d, want %d", tt.groupID, tt.preferred, got, tt.want)
			}
		})
	}
}

func TestReleaseYear(t *testing.T) {
	if got := releaseYear("2024-01-02"); got != "2024" {
		t.Fatalf("releaseYear = %q, want %q", got, "2024")
	}
	if got := releaseYear(""); got != "" {
		t.Fatalf("releaseYear = %q, want empty", got)
	}
}

func TestFirstGenre(t *testing.T) {
	if got := firstGenre([]string{"Pop", "Rock"}); got != "Pop" {
		t.Fatalf("firstGenre = %q, want %q", got, "Pop")
	}
	if got := firstGenre(nil); got != "" {
		t.Fatalf("firstGenre = %q, want empty", got)
	}
}

func TestFlagValueFromArgs(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--lite-server=http://localhost:10020"}, want: "http://localhost:10020"},
		{args: []string{"--lite-server", "http://localhost:10020"}, want: "http://localhost:10020"},
		{args: []string{"--lite-server"}, want: ""},
		{args: []string{"url"}, want: ""},
	}
	for _, tt := range tests {
		if got := flagValueFromArgs(tt.args, "lite-server"); got != tt.want {
			t.Fatalf("r.flagValueFromArgs(%v) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestWriteCoverFailurePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(existing, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	originalClient := download.Client
	t.Cleanup(func() {
		download.Client = originalClient
	})
	r := NewRunner(config.ConfigSet{})
	r.Config.CoverFormat = "jpg"
	r.Config.CoverSize = "600x600"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	download.Client = server.Client()

	_, err := r.writeCover(dir, "cover", server.URL+"/image/{w}x{h}.jpg")
	if err == nil {
		t.Fatal("writeCover() error = nil, want failure")
	}
	data, readErr := os.ReadFile(existing)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "old" {
		t.Fatalf("existing cover changed to %q", string(data))
	}
}
