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
        <translations>
          <translation type="subtitle">
            <text for="L1">说不上为什么 我变得很主动</text><text for="L2">若爱上一个人 什么都会值得去做</text>
          </translation>
        </translations>
        <transliterations>
          <transliteration>
            <text for="L1">shuo bu shang wei shen me wo bian de hen zhu dong</text><text for="L2">ruo ai shang yi ge ren shen me du hui zhi de qu zuo</text>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div begin="26.663" end="46.313" itunes:songPart="Verse"><p begin="26.663" end="31.467" itunes:key="L1" ttm:agent="v1">說不上為什麼 我變得很主動</p><p begin="31.673" end="36.360" itunes:key="L2" ttm:agent="v1">若愛上一個人 什麼都會值得去做</p>
    </div>
  </body>
</tt>`

	lineReplacementTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Line">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations>
          <translation type="replacement">
            <text for="L1">说不上为什么 我变得很主动</text><text for="L2">若爱上一个人 什么都会值得去做</text>
          </translation>
        </translations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div begin="26.663" end="46.313" itunes:songPart="Verse"><p begin="26.663" end="31.467" itunes:key="L1" ttm:agent="v1">說不上為什麼 我變得很主動</p><p begin="31.673" end="36.360" itunes:key="L2" ttm:agent="v1">若愛上一個人 什麼都會值得去做</p>
    </div>
  </body>
</tt>`

	syllableTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Word">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations>
          <translation type="replacement">
            <text for="L1"><span begin="26.663" end="28.242" xmlns="http://www.w3.org/ns/ttml">说不上为什</span><span begin="28.242" end="29.013" xmlns="http://www.w3.org/ns/ttml">么</span> <span begin="29.193" end="30.794" xmlns="http://www.w3.org/ns/ttml">我变得很主</span><span begin="30.794" end="31.467" xmlns="http://www.w3.org/ns/ttml">动</span></text><text for="L2"><span begin="31.673" end="33.552" xmlns="http://www.w3.org/ns/ttml">若爱上一个人</span> <span begin="33.552" end="33.850" xmlns="http://www.w3.org/ns/ttml">什么</span><span begin="33.850" end="35.105" xmlns="http://www.w3.org/ns/ttml">都会值得</span><span begin="35.105" end="36.360" xmlns="http://www.w3.org/ns/ttml">去做</span></text>
          </translation>
        </translations>
        <transliterations>
          <transliteration>
            <text for="L1"><span begin="26.663" end="28.242" xmlns="http://www.w3.org/ns/ttml">shuo bu shang wei shen</span> <span begin="28.242" end="29.013" xmlns="http://www.w3.org/ns/ttml">me</span> <span begin="29.193" end="30.794" xmlns="http://www.w3.org/ns/ttml">wo bian de hen zhu</span> <span begin="30.794" end="31.467" xmlns="http://www.w3.org/ns/ttml">dong</span></text><text for="L2"><span begin="31.673" end="33.552" xmlns="http://www.w3.org/ns/ttml">ruo ai shang yi ge ren</span> <span begin="33.552" end="33.850" xmlns="http://www.w3.org/ns/ttml">shen me</span> <span begin="33.850" end="35.105" xmlns="http://www.w3.org/ns/ttml">du hui zhi de</span> <span begin="35.105" end="36.360" xmlns="http://www.w3.org/ns/ttml">qu zuo</span></text>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div begin="26.663" end="46.313" itunes:songPart="Verse"><p begin="26.663" end="31.467" itunes:key="L1" ttm:agent="v1"><span begin="26.663" end="28.242">說不上為什</span><span begin="28.242" end="29.013">麼</span> <span begin="29.193" end="30.794">我變得很主</span><span begin="30.794" end="31.467">動</span></p><p begin="31.673" end="36.360" itunes:key="L2" ttm:agent="v1"><span begin="31.673" end="33.552">若愛上一個人</span> <span begin="33.552" end="33.850">什麼</span><span begin="33.850" end="35.105">都會值得</span><span begin="35.105" end="36.360">去做</span></p>
    </div>
  </body>
</tt>`

	subtitleSyllableTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Word">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations>
          <translation type="subtitle">
            <text for="L1">别着急 <span xmlns:ttm="http://www.w3.org/ns/ttml#metadata" ttm:role="x-bg" xmlns="http://www.w3.org/ns/ttml">(别着急)</span></text><text for="L2">教你我如何互相信任 不必偷偷摸摸 <span xmlns:ttm="http://www.w3.org/ns/ttml#metadata" ttm:role="x-bg" xmlns="http://www.w3.org/ns/ttml">(答应我)</span></text>
          </translation>
        </translations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div begin="7.439" end="23.760" itunes:songPart="Verse"><p begin="7.439" end="9.794" itunes:key="L1" ttm:agent="v1"><span begin="7.439" end="7.988">Don't</span> <span begin="7.988" end="8.700">rush</span> <span ttm:role="x-bg"><span begin="8.435" end="8.921">(Don't</span> <span begin="8.921" end="9.794">rush)</span></span></p><p begin="9.304" end="14.834" itunes:key="L2" ttm:agent="v1"><span begin="9.304" end="9.915">Teaching</span> <span begin="9.915" end="10.399">you,</span> <span begin="10.399" end="10.900">teaching</span> <span begin="10.900" end="11.427">me</span> <span begin="11.427" end="11.688">how</span> <span begin="11.688" end="11.919">to</span> <span begin="11.919" end="12.386">trust,</span> <span begin="12.386" end="12.652">don't</span> <span begin="12.652" end="12.884">have</span> <span begin="12.884" end="13.126">to</span> <span begin="13.126" end="13.402">be</span> <span begin="13.402" end="13.822">under</span><span begin="13.822" end="14.125">co</span><span begin="14.125" end="14.834">ver</span></p>
    </div>
  </body>
</tt>`

	backgroundReplacementSyllableTTML = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" xmlns:ttm="http://www.w3.org/ns/ttml#metadata" itunes:timing="Word">
  <head>
    <metadata>
      <iTunesMetadata>
        <translations>
          <translation type="replacement">
            <text for="L1"><span begin="00:00:01.250" end="00:00:01.500">替换</span> <span ttm:role="x-bg"><span begin="00:00:01.500" end="00:00:02.000">背景</span></span></text>
          </translation>
        </translations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div><p begin="00:00:01.250" end="00:00:02.000" itunes:key="L1"><span begin="00:00:01.250" end="00:00:01.500">Original</span></p></div>
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
			want: "[00:26.66]說不上為什麼 我變得很主動\n" +
				"[00:31.67]若愛上一個人 什麼都會值得去做",
		},
		{
			name:        "pronunciation line",
			lyricsExtra: "pronunciation",
			want: "[00:26.66]說不上為什麼 我變得很主動\n" +
				"[00:26.66]shuo bu shang wei shen me wo bian de hen zhu dong\n" +
				"[00:31.67]若愛上一個人 什麼都會值得去做\n" +
				"[00:31.67]ruo ai shang yi ge ren shen me du hui zhi de qu zuo",
		},
		{
			name:        "subtitle translation line",
			lyricsExtra: "translation",
			want: "[00:26.66]說不上為什麼 我變得很主動\n" +
				"[00:26.66]说不上为什么 我变得很主动\n" +
				"[00:31.67]若愛上一個人 什麼都會值得去做\n" +
				"[00:31.67]若爱上一个人 什么都会值得去做",
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

	want := "[00:26.66]说不上为什么 我变得很主动\n" +
		"[00:31.67]若爱上一个人 什么都会值得去做"
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

	want := "[00:26.66]<00:26.66>说不上为什<00:28.24>么 <00:29.19>我变得很主<00:30.79>动<00:31.46>\n" +
		"[00:31.67]<00:31.67>若爱上一个人 <00:33.55>什么<00:33.85>都会值得<00:35.10>去做<00:36.36>"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestConventSyllableTTMLToLRCWithPronunciation(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(syllableTTML, "pronunciation")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:26.66]<00:26.66>说不上为什<00:28.24>么 <00:29.19>我变得很主<00:30.79>动<00:31.46>\n" +
		"[00:26.66]<00:26.66>shuo bu shang wei shen <00:28.24>me <00:29.19>wo bian de hen zhu <00:30.79>dong<00:31.46>\n" +
		"[00:31.67]<00:31.67>若爱上一个人 <00:33.55>什么<00:33.85>都会值得<00:35.10>去做<00:36.36>\n" +
		"[00:31.67]<00:31.67>ruo ai shang yi ge ren <00:33.55>shen me <00:33.85>du hui zhi de <00:35.10>qu zuo<00:36.36>"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestConventSyllableTTMLToLRCWithSubtitleTranslation(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(subtitleSyllableTTML, "translation")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:07.43]<00:07.43>Don't <00:07.98>rush <00:08.70>\n" +
		"[00:07.43]别着急 (别着急)\n" +
		"[00:09.30]<00:09.30>Teaching <00:09.91>you, <00:10.39>teaching <00:10.90>me <00:11.42>how <00:11.68>to <00:11.91>trust, <00:12.38>don't <00:12.65>have <00:12.88>to <00:13.12>be <00:13.40>under<00:13.82>co<00:14.12>ver<00:14.83>\n" +
		"[00:09.30]教你我如何互相信任 不必偷偷摸摸 (答应我)"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestConventSyllableTTMLToLRCWithReplacement(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(syllableTTML, "translation")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:26.66]<00:26.66>说不上为什<00:28.24>么 <00:29.19>我变得很主<00:30.79>动<00:31.46>\n" +
		"[00:31.67]<00:31.67>若爱上一个人 <00:33.55>什么<00:33.85>都会值得<00:35.10>去做<00:36.36>"
	if got != want {
		t.Fatalf("conventSyllableTTMLToLRC()\n got: %q\nwant: %q", got, want)
	}
}

func TestConventSyllableTTMLToLRCWithBackgroundReplacement(t *testing.T) {
	got, err := conventSyllableTTMLToLRC(backgroundReplacementSyllableTTML, "")
	if err != nil {
		t.Fatalf("conventSyllableTTMLToLRC() error = %v", err)
	}

	want := "[00:01.25]<00:01.25>替换<00:01.50>"
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
