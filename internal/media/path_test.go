package media

import "testing"

func TestParseTrackFilename(t *testing.T) {
	num, title := ParseTrackFilename(`C:\Music\01. Lifelike.m4a`)
	if num != 1 || title != "Lifelike" {
		t.Fatalf("got track %d title %q", num, title)
	}
}
