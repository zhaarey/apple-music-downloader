package spotify

import "testing"

func TestParseLink(t *testing.T) {
	cases := []struct {
		raw      string
		kind     LinkKind
		id       string
	}{
		{"https://open.spotify.com/track/abc123", LinkTrack, "abc123"},
		{"https://open.spotify.com/playlist/xyz789?si=foo", LinkPlaylist, "xyz789"},
		{"spotify:album:albumid", LinkAlbum, "albumid"},
	}
	for _, c := range cases {
		kind, id, ok := ParseLink(c.raw)
		if !ok {
			t.Fatalf("expected ok for %q", c.raw)
		}
		if kind != c.kind || id != c.id {
			t.Fatalf("ParseLink(%q) = %s %s, want %s %s", c.raw, kind, id, c.kind, c.id)
		}
	}
}
