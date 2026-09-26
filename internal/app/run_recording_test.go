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
	r := NewRunner(config.ConfigSet{})
	raw := "https://classical.music.apple.com/us/recording/abc-1873004116?l=en-US"
	if !r.handleClassicalURL(raw, "token") {
		t.Fatal("recording URL not handled")
	}
	if fetched.Language != "en-US" || *loaded != "en-US" {
		t.Fatalf("recording language = %q / %q, want en-US", fetched.Language, *loaded)
	}
	if r.Config.Language != "" {
		t.Fatalf("config language after recording = %q, want it empty again", r.Config.Language)
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

// stubStorefrontLanguage makes the storefront lookup return language and err,
// and counts the lookups.
func stubStorefrontLanguage(t *testing.T, language string, err error) *int {
	t.Helper()
	calls := new(int)
	orig := storefrontLanguage
	t.Cleanup(func() { storefrontLanguage = orig })
	storefrontLanguage = func(string, string) (string, error) {
		*calls++
		return language, err
	}
	return calls
}

func TestRecordingLanguageFallsBackToStorefront(t *testing.T) {
	const link = "https://classical.music.apple.com/jp/recording/abc-1873004116"
	cases := []struct {
		name, url, config string
		lookup            string
		lookupErr         error
		want              string
		lookups           int
	}{
		{"config language over the link", link + "?l=ja", "en-US", "ja", nil, "en-US", 0},
		{"link language without config", link + "?l=de-DE", "", "ja", nil, "de-DE", 0},
		{"config language", link, "en-US", "ja", nil, "en-US", 0},
		{"storefront default", link, "", "ja", nil, "ja", 1},
		{"lookup failed", link, "", "", errors.New("offline"), "", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetched, loaded := stubRecording(t)
			lookups := stubStorefrontLanguage(t, tc.lookup, tc.lookupErr)
			r := NewRunner(config.ConfigSet{Language: tc.config})

			r.handleClassicalURL(tc.url, "token")

			if fetched.Language != tc.want || *loaded != tc.want || *lookups != tc.lookups {
				t.Fatalf("classical %q, catalog %q, %d lookups; want %q and %d lookups",
					fetched.Language, *loaded, *lookups, tc.want, tc.lookups)
			}
		})
	}
}

func TestRipRecordingWarnsWhenCatalogUsesAnotherLanguage(t *testing.T) {
	origShow := showTrackQuality
	t.Cleanup(func() { showTrackQuality = origShow })
	showTrackQuality = func(*Runner, int, string, string, []string, string, string) {}
	pachelbel := func(language string) *classical.Recording {
		return &classical.Recording{
			Request:   classical.Request{Storefront: "cn", Language: language, RecordingID: "johann-pachelbel-1653-pp429-1452536848"},
			AlbumID:   "1452536848",
			WorkTitle: "D 大调卡农与吉格，P. 37",
			Tracks:    []classical.Track{{SongID: "1452537828", Title: "I. Canon", Position: 1, Count: 1}},
		}
	}
	beethoven := func(language string) *classical.Recording {
		rec := beethovenRecording("1873004347")
		rec.Request.Language = language
		return rec
	}
	cases := []struct {
		name  string
		album *model.Album // its href records the language the Catalog used
		rec   *classical.Recording
		warn  bool
	}{
		{"us served en-US for de-DE", beethovenAlbum(t), beethoven("de-DE"), true},
		{"us served en-US for en-GB", beethovenAlbum(t), beethoven("en-GB"), false},
		{"nothing requested", beethovenAlbum(t), beethoven(""), false},
		{"cn normalized zh-CN", catalogAlbum(t, "cn_pachelbel_zh-CN.json", "cn", "1452536848", "zh-CN"), pachelbel("zh-CN"), false},
		{"cn served zh-Hans-CN for en-US", catalogAlbum(t, "cn_pachelbel_zh-CN.json", "cn", "1452536848", "en-US"), pachelbel("en-US"), true},
		{"cn served Simplified for zh-TW", catalogAlbum(t, "cn_pachelbel_zh-CN.json", "cn", "1452536848", "zh-TW"), pachelbel("zh-TW"), true},
		{"cn served zh-Hans-CN for zh-Hans", catalogAlbum(t, "cn_pachelbel_zh-CN.json", "cn", "1452536848", "zh-Hans"), pachelbel("zh-Hans"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubAlbum(t, tc.album)
			r := NewRunner(config.ConfigSet{LimitMax: 200, AlacSaveFolder: t.TempDir()})
			r.Flags.Debug = true

			if err := r.ripRecording(tc.rec, "", ""); err != nil {
				t.Fatal(err)
			}
			if got := len(tc.rec.Warnings) > 0; got != tc.warn {
				t.Errorf("warnings = %v, want a warning = %v", tc.rec.Warnings, tc.warn)
			}
		})
	}
}
