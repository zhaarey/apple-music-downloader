package engine

import (
	"errors"
	"testing"
)

func TestIsMissingMP4MetaBox(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("MP4Box failed"), false},
		{errors.New("moov.udta.meta box not present"), true},
		{errors.New("moov.udta box not present"), true},
		{errors.New("ilst box not present, implement me"), true},
	}
	for _, tc := range cases {
		if got := isMissingMP4MetaBox(tc.err); got != tc.want {
			t.Fatalf("isMissingMP4MetaBox(%q) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestMp4boxITagValue_escapesColon(t *testing.T) {
	if got := mp4boxITagValue("a:b"); got != `a\:b` {
		t.Fatalf("got %q", got)
	}
}
