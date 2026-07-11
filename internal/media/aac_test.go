package media

import (
	"path/filepath"
	"testing"
)

func TestIsConvertibleAudioExt(t *testing.T) {
	for _, ext := range []string{".mp3", ".FLAC", ".wav", ".m4a", ".ogg"} {
		if !IsConvertibleAudioExt(ext) {
			t.Fatalf("expected convertible: %s", ext)
		}
	}
	if IsConvertibleAudioExt(".txt") {
		t.Fatal("txt should not be convertible")
	}
}

func TestDefaultAppleAACOutputPath(t *testing.T) {
	src := filepath.Join("C:", "Music", "song.mp3")
	got := DefaultAppleAACOutputPath(src)
	want := filepath.Join("C:", "Music", "song.m4a")
	if got != want {
		t.Fatalf("DefaultAppleAACOutputPath(%q) = %q, want %q", src, got, want)
	}

	srcM4A := filepath.Join("C:", "Music", "already.m4a")
	got = DefaultAppleAACOutputPath(srcM4A)
	want = filepath.Join("C:", "Music", "already - AAC.m4a")
	if got != want {
		t.Fatalf("DefaultAppleAACOutputPath(%q) = %q, want %q", srcM4A, got, want)
	}
}
