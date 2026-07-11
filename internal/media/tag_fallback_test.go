package media

import "testing"

func TestIsCorruptMP4TagErr(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"mp4tag read failed: runtime error: makeslice: len out of range", true},
		{"box not present: meta", false},
		{"permission denied", false},
		{"slice bounds out of range [1:0]", true},
	}
	for _, c := range cases {
		err := errString(c.msg)
		if got := isCorruptMP4TagErr(err); got != c.want {
			t.Fatalf("isCorruptMP4TagErr(%q)=%v want %v", c.msg, got, c.want)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestParseNumberPair(t *testing.T) {
	n, total := parseNumberPair("3/12")
	if n != 3 || total != 12 {
		t.Fatalf("got %d/%d", n, total)
	}
	n, total = parseNumberPair("7")
	if n != 7 || total != 0 {
		t.Fatalf("got %d/%d", n, total)
	}
}
