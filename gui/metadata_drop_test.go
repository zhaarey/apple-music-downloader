package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTagResolveDrop(t *testing.T) {
	app := &App{}
	dir := t.TempDir()
	sub := filepath.Join(dir, "Album")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	f1 := filepath.Join(sub, "01 Song.m4a")
	f2 := filepath.Join(sub, "02 Song.m4a")
	for _, p := range []string{f1, f2} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	other := filepath.Join(dir, "other.m4a")
	if err := os.WriteFile(other, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("folder", func(t *testing.T) {
		res, err := app.TagResolveDrop([]string{sub})
		if err != nil {
			t.Fatal(err)
		}
		if res.Mode != "album" || res.Path != sub {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("single file", func(t *testing.T) {
		res, err := app.TagResolveDrop([]string{f1})
		if err != nil {
			t.Fatal(err)
		}
		if res.Mode != "single" || res.Path != f1 {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("sibling files", func(t *testing.T) {
		res, err := app.TagResolveDrop([]string{f1, f2})
		if err != nil {
			t.Fatal(err)
		}
		if res.Mode != "album" || res.Path != sub {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("different folders", func(t *testing.T) {
		res, err := app.TagResolveDrop([]string{f1, other})
		if err != nil {
			t.Fatal(err)
		}
		if res.Mode != "single" || res.Path != f1 {
			t.Fatalf("got %+v", res)
		}
		if res.Message == "" {
			t.Fatal("expected guidance message")
		}
	})

	t.Run("empty", func(t *testing.T) {
		_, err := app.TagResolveDrop([]string{filepath.Join(dir, "nope.txt")})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
