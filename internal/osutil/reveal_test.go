package osutil

import (
	"runtime"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		got, err := normalizePath(`C:/Users/test/Music/Artist/Album/track.m4a`)
		if err != nil {
			t.Fatal(err)
		}
		if got != `C:\Users\test\Music\Artist\Album\track.m4a` {
			t.Fatalf("got %q", got)
		}
	}
	got, err := normalizePath(`  "relative/path"  `)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected non-empty abs path")
	}
}

func TestNormalizePath_empty(t *testing.T) {
	if _, err := normalizePath("  "); err == nil {
		t.Fatal("expected error for empty path")
	}
}
