package crossdevice

import "testing"

func TestTrackFilename(t *testing.T) {
	got := TrackFilename(1, "Porter Robinson Live", AudioExtension)
	want := "01. Porter Robinson Live.m4a"
	if got != want {
		t.Fatalf("TrackFilename = %q, want %q", got, want)
	}
}

func TestVideoFilename(t *testing.T) {
	got := VideoFilename(1, "Porter Robinson Live")
	want := "01. Porter Robinson Live [video].mp4"
	if got != want {
		t.Fatalf("VideoFilename = %q, want %q", got, want)
	}
}

func TestSanitizePathPart(t *testing.T) {
	if got := SanitizePathPart(`a/b:c`); got != "a_b_c" {
		t.Fatalf("sanitize = %q", got)
	}
	if got := SanitizePathPart(""); got != "Unknown" {
		t.Fatalf("empty = %q", got)
	}
}

func TestIsVideoTrackFilename(t *testing.T) {
	if !IsVideoTrackFilename("01. Set [video].mp4") {
		t.Fatal("expected video")
	}
	if IsVideoTrackFilename("01. Set.m4a") {
		t.Fatal("expected audio")
	}
}

func TestAudioStemFromVideo(t *testing.T) {
	got := AudioStemFromVideo("01. Set [video].mp4")
	if got != "01. Set.m4a" {
		t.Fatalf("stem = %q", got)
	}
}

func TestAlbumRelativePath(t *testing.T) {
	if got := AlbumRelativePath(""); got != "YouTube Downloads" {
		t.Fatalf("empty album = %q", got)
	}
	if got := AlbumRelativePath("EDC 2026"); got != "EDC 2026" {
		t.Fatalf("album = %q", got)
	}
}
