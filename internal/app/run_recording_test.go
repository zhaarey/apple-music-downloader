package app

import (
	"errors"
	"os"
	"strings"
	"testing"

	"amdl/internal/classical"
	"amdl/internal/config"
	"amdl/internal/model"
)

func stubRecording(t *testing.T) (fetched *classical.Request, loadedLanguage *string) {
	t.Helper()
	fetched, loadedLanguage = new(classical.Request), new(string)
	origFetch, origLoad := fetchRecording, loadRecordingAlbum
	t.Cleanup(func() { fetchRecording, loadRecordingAlbum = origFetch, origLoad })
	fetchRecording = func(req classical.Request) (*classical.Recording, error) {
		*fetched = req
		return &classical.Recording{Request: req, AlbumID: "1873004116"}, nil
	}
	loadRecordingAlbum = func(_, _, _, language string) (*model.Album, error) {
		*loadedLanguage = language
		return nil, errors.New("stop before download")
	}
	return fetched, loadedLanguage
}

func TestRecordingLanguageIsIsolated(t *testing.T) {
	fetched, loaded := stubRecording(t)
	r := NewRunner(config.ConfigSet{Language: "zh-CN"})
	raw := "https://classical.music.apple.com/us/recording/abc-1873004116?l=en-US"
	if !r.handleClassicalURL(raw, "token") {
		t.Fatal("recording URL not handled")
	}
	if fetched.Language != "en-US" || *loaded != "en-US" {
		t.Fatalf("recording language = %q / %q, want en-US", fetched.Language, *loaded)
	}
	if r.Config.Language != "zh-CN" {
		t.Fatalf("config language after recording = %q, want zh-CN", r.Config.Language)
	}
	if r.handleClassicalURL("https://music.apple.com/us/album/x/1873004116", "token") {
		t.Fatal("a normal album URL must not be handled as classical")
	}
}

func TestClassicalWorkURLIsRejected(t *testing.T) {
	fetched, _ := stubRecording(t)
	r := NewRunner(config.ConfigSet{})
	if !r.handleClassicalURL("https://classical.music.apple.com/us/work/some-work", "token") {
		t.Fatal("work URL not handled")
	}
	// Like an unknown link type it is not an error: a retry cannot fix it.
	if r.State.Counter != (config.Counter{}) || fetched.RecordingID != "" {
		t.Fatalf("counter = %+v, fetched = %+v", r.State.Counter, *fetched)
	}
}

// stubAlbum makes loadRecordingAlbum return album without any request.
func stubAlbum(t *testing.T, album *model.Album) {
	t.Helper()
	orig := loadRecordingAlbum
	t.Cleanup(func() { loadRecordingAlbum = orig })
	loadRecordingAlbum = func(_, _, _, _ string) (*model.Album, error) { return album, nil }
}

func TestRipRecordingChecksFileTemplateBeforeWriting(t *testing.T) {
	album := beethovenAlbum(t)
	album.Resp.Data[0].Attributes.Artwork.URL = "" // no cover request
	stubAlbum(t, album)
	root := t.TempDir()
	r := NewRunner(config.ConfigSet{LimitMax: 200, AlacSaveFolder: root, ClassicalFileFormat: "{Nope}"})

	if err := r.ripRecording(beethovenRecording("1873004347"), "", ""); err == nil {
		t.Fatal("expected an error for the unknown placeholder")
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("save folder has %d entries, want none", len(entries))
	}
}

func TestRipRecordingDebugOnlyInspects(t *testing.T) {
	album := beethovenAlbum(t)
	album.Resp.Data[0].Attributes.Artwork.URL = "" // no cover request
	stubAlbum(t, album)
	var inspected []string
	origShow := showTrackQuality
	t.Cleanup(func() { showTrackQuality = origShow })
	showTrackQuality = func(_ *Runner, _ int, storefront, songID string, _ []string, language, _ string) {
		inspected = append(inspected, storefront+"/"+songID+"/"+language)
	}
	root := t.TempDir()
	r := NewRunner(config.ConfigSet{LimitMax: 200, AlacSaveFolder: root})
	r.Flags.Debug = true

	if err := r.ripRecording(beethovenRecording("1873004590", "1873004347"), "", ""); err != nil {
		t.Fatal(err)
	}
	// Only the Recording's songs, in Recording order, with its language.
	if got := strings.Join(inspected, ","); got != "us/1873004590/en-US,us/1873004347/en-US" {
		t.Fatalf("inspected = %s", got)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("save folder has %d entries, want none", len(entries))
	}
	if r.State.Counter != (config.Counter{}) {
		t.Fatalf("counter = %+v, want nothing counted", r.State.Counter)
	}
}
