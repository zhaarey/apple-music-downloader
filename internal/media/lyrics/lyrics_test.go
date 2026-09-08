package lyrics

import (
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

const (
	lineTimedTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Line">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations type="subtitle">
          <translation for="line-1" text="你好，世界"/>
        </translations>
        <transliterations>
          <transliteration for="line-1" text="hello world"/>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div>
      <p itunes:key="line-1" begin="00:01:02.250" end="00:01:04.500" text="Hello world">
      </p>
    </div>
  </body>
</tt>`

	lineReplacementTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Line">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations type="replacement">
          <translation for="line-1" text="替换文本"/>
        </translations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div>
      <p itunes:key="line-1" begin="01:02.500" end="01:03.000" text="Original">
      </p>
    </div>
  </body>
</tt>`

	syllableTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Word">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations type="replacement">
          <translation for="line-1">
            <span begin="00:00:01.250">替换</span>
            <span begin="00:00:01.500">文本</span>
          </translation>
        </translations>
        <transliterations>
          <transliteration for="line-1">
            <span begin="00:00:01.250">HEH</span>
            <span begin="00:00:01.500">LOH</span>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div>
      <p itunes:key="line-1"><span begin="00:00:01.250" end="00:00:01.500" text="Hel"/> <span begin="00:00:01.500" end="00:00:02.000" text="lo"/></p>
    </div>
  </body>
</tt>`

	subtitleSyllableTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Word">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations type="subtitle">
          <translation for="line-1" text="你好"/>
        </translations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div>
      <p itunes:key="line-1"><span begin="00:00:01.250" end="00:00:02.000" text="Hello"/></p>
    </div>
  </body>
</tt>`

	unTimedTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="None">
  <body>
    <div>
      <p>First line</p>
      <p>  </p>
      <p>Second line</p>
    </div>
  </body>
</tt>`

	validFallbackSongID = "1624945512"
)

var lrcLineTagPattern = regexp.MustCompile(`^\[\d{2}:\d{2}\.\d{2}\]`)

func TestTtmlToLrcLineTimed(t *testing.T) {
	tests := []struct {
		name        string
		lyricsExtra string
		want        string
	}{
		{
			name: "default line",
			want: "[01:02.25]Hello world",
		},
		{
			name:        "pronunciation line",
			lyricsExtra: "pronunciation",
			want:        "[01:02.25]Hello world\n[01:02.25]hello world",
		},
		{
			name:        "subtitle translation line",
			lyricsExtra: "translation",
			want:        "[01:02.25]Hello world\n[01:02.25]你好，世界",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := TtmlToLrc(lineTimedTTML, test.lyricsExtra)
			if err != nil {
				t.Fatalf("TtmlToLrc() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("TtmlToLrc()\n got: %q\nwant: %q", got, test.want)
			}
		})
	}
}

func TestTtmlToLrcReplacementLine(t *testing.T) {
	got, err := TtmlToLrc(lineReplacementTTML, "")
	if err != nil {
		t.Fatalf("TtmlToLrc() error = %v", err)
	}

	want := "[01:02.50]替换文本"
	if got != want {
		t.Fatalf("TtmlToLrc() = %q, want %q", got, want)
	}
}

func TestTtmlToLrcUnTimed(t *testing.T) {
	got, err := TtmlToLrc(unTimedTTML, "")
	if err != nil {
		t.Fatalf("TtmlToLrc() error = %v", err)
	}

	want := "First line\nSecond line"
	if got != want {
		t.Fatalf("TtmlToLrc() = %q, want %q", got, want)
	}
}

func TestConventSyllableTTMLToLRC(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(syllableTTML, "")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:01.25]<00:01.25>替换 <00:01.50>文本"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestConventSyllableTTMLToLRCWithPronunciation(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(syllableTTML, "pronunciation")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:01.25]<00:01.25>替换 <00:01.50>文本\n" +
		"[00:01.25]<00:01.25>HEH <00:01.50>LOH"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestConventSyllableTTMLToLRCWithSubtitleTranslation(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(subtitleSyllableTTML, "translation")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:01.25]<00:01.25>Hello[00:02.00]\n[00:01.25]你好"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestConventSyllableTTMLToLRCWithReplacement(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(syllableTTML, "translation")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:01.25]<00:01.25>替换 <00:01.50>文本"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestParseTTMLTime(t *testing.T) {
	tests := []struct {
		value string
		want  lrcTime
	}{
		{value: "00:01:02.250", want: lrcTime{minutes: 1, seconds: 2, centiseconds: 25}},
		{value: "01:02.250", want: lrcTime{minutes: 1, seconds: 2, centiseconds: 25}},
		{value: "01:02", want: lrcTime{minutes: 1, seconds: 2}},
		{value: "02.250", want: lrcTime{seconds: 2, centiseconds: 25}},
		{value: "101.046s", want: lrcTime{minutes: 1, seconds: 41, centiseconds: 4}},
		{value: "1.2345s", want: lrcTime{seconds: 1, centiseconds: 23}},
	}

	for _, test := range tests {
		got, err := parseTTMLTime(test.value)
		if err != nil {
			t.Fatalf("parseTTMLTime(%q) error = %v", test.value, err)
		}
		if got != test.want {
			t.Fatalf("parseTTMLTime(%q) = %+v, want %+v", test.value, got, test.want)
		}
	}
}

func TestResolveLiteServerPrefersEnvironment(t *testing.T) {
	t.Setenv("LITE_SERVER", "http://127.0.0.1:3999/")
	if got := resolveLiteServer(); got != "http://127.0.0.1:3999" {
		t.Fatalf("resolveLiteServer() = %q, want %q", got, "http://127.0.0.1:3999")
	}
}

func TestWrapperLiteRandomSongIDIntegration(t *testing.T) {
	liteServer := resolveLiteServer()
	randomSongID := rand.Int63n(9000000000) + 1000000000
	randomSongIDText := strconv.FormatInt(randomSongID, 10)

	ttml, err := getSongLyrics(randomSongIDText, liteServer, "syllable-lyrics", "en-US")
	if err != nil {
		t.Logf("random song id %s was not available: %v", randomSongIDText, err)
		ttml, err = getSongLyrics(validFallbackSongID, liteServer, "syllable-lyrics", "en-US")
	}
	if err != nil {
		t.Skipf("wrapper-lite is unavailable: %v", err)
	}
	if strings.TrimSpace(ttml) == "" {
		t.Fatal("wrapper-lite returned empty TTML")
	}

	assertLRCFromTTML(t, ttml)
}

func assertLRCFromTTML(t *testing.T, ttml string) {
	t.Helper()

	for name, lyricsExtra := range map[string]string{
		"default":       "",
		"pronunciation": "pronunciation",
		"translation":   "translation",
	} {
		t.Run(name, func(t *testing.T) {
			lrc, err := TtmlToLrc(ttml, lyricsExtra)
			if err != nil {
				t.Fatalf("TtmlToLrc(%q) error = %v", lyricsExtra, err)
			}
			if strings.TrimSpace(lrc) == "" {
				t.Fatal("TtmlToLrc returned empty LRC")
			}

			for number, line := range strings.Split(lrc, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				if !lrcLineTagPattern.MatchString(line) {
					t.Fatalf("LRC line %d does not start with a time tag: %q", number+1, line)
				}
			}
		})
	}
}

func resolveLiteServer() string {
	if value := normalizeLiteServer(os.Getenv("LITE_SERVER")); value != "" {
		return value
	}
	if value := liteServerFromConfig(); value != "" {
		return value
	}
	return "http://127.0.0.1:12340"
}

func liteServerFromConfig() string {
	path, err := findConfigFile()
	if err != nil {
		return ""
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var config struct {
		LiteServer string `yaml:"lite-server"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return ""
	}
	return normalizeLiteServer(config.LiteServer)
}

func findConfigFile() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(current, "config.yaml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
		current = parent
	}
}

func normalizeLiteServer(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "s")
	return strings.TrimRight(value, "/")
}
