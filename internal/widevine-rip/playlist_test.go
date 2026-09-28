package widevinerip

import "testing"

func TestFindKeyURIByFormat(t *testing.T) {
	const playlist = `#EXTM3U
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="skd://itunes.apple.com/p996363250/c2",KEYFORMAT="com.apple.streamingkeydelivery",KEYFORMATVERSIONS="2"
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="data:text/plain;charset=UTF-16;base64,playready",KEYFORMAT="com.microsoft.playready",KEYFORMATVERSIONS="2"
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="data:text/plain;base64,widevine",KEYFORMAT="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed",KEYFORMATVERSIONS="2"
`

	if got := findKeyURIByFormat(playlist, "com.microsoft.playready"); got != "data:text/plain;charset=UTF-16;base64,playready" {
		t.Fatalf("unexpected PlayReady key URI: %q", got)
	}
	if got := findKeyURIByFormat(playlist, "missing"); got != "" {
		t.Fatalf("expected missing key format to return empty string, got %q", got)
	}
}

func TestSplitKeyURI(t *testing.T) {
	prefix, payload, err := splitKeyURI("data:text/plain;charset=UTF-16;base64,abc")
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "data:text/plain;charset=UTF-16;base64" || payload != "abc" {
		t.Fatalf("unexpected split result: %q %q", prefix, payload)
	}

	if _, _, err := splitKeyURI("not-a-data-uri"); err == nil {
		t.Fatal("expected invalid key URI to fail")
	}
}

func TestParseMediaPlaylist(t *testing.T) {
	const playlist = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-TARGETDURATION:6
#EXT-X-MAP:URI="init.mp4"
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="data:text/plain;base64,a2lk",KEYFORMAT="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed",KEYFORMATVERSIONS="1"
#EXTINF:6.0,
seg0.m4s
#EXTINF:6.0,
https://cdn.example.com/abs/seg1.m4s
#EXT-X-ENDLIST
`
	got, err := ParseMediaPlaylist("https://example.com/path/media.m3u8", []byte(playlist), DefaultKeyFormat)
	if err != nil {
		t.Fatal(err)
	}
	if got.KeyURIPrefix != "data:text/plain;base64" || got.KeyPayload != "a2lk" {
		t.Fatalf("unexpected key: %q %q", got.KeyURIPrefix, got.KeyPayload)
	}
	if got.KeyURI() != "data:text/plain;base64,a2lk" {
		t.Fatalf("unexpected key URI: %q", got.KeyURI())
	}
	want := []string{
		"https://example.com/path/init.mp4",
		"https://example.com/path/seg0.m4s",
		"https://cdn.example.com/abs/seg1.m4s",
	}
	urls := got.StreamURLs()
	if len(urls) != len(want) {
		t.Fatalf("StreamURLs() = %q, want %q", urls, want)
	}
	for i := range want {
		if urls[i] != want[i] {
			t.Fatalf("StreamURLs()[%d] = %q, want %q", i, urls[i], want[i])
		}
	}
}

func TestParseMediaPlaylistRejectsMasterPlaylist(t *testing.T) {
	const playlist = `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=256000
variant.m3u8
`
	if _, err := ParseMediaPlaylist("https://example.com/master.m3u8", []byte(playlist), DefaultKeyFormat); err == nil {
		t.Fatal("expected master playlist to be rejected")
	}
}
