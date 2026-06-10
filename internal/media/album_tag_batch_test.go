package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListAlbumTagFiles_direct(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"02 b.m4a", "01 a.m4a", "readme.txt"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := ListAlbumTagFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(paths))
	}
	if filepath.Base(paths[0]) != "01 a.m4a" {
		t.Fatalf("expected sorted paths, got %v", paths)
	}
}

func TestListAlbumTagFiles_includesMP4(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "track.mp4"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	paths, err := ListAlbumTagFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 track, got %d", len(paths))
	}
}

func TestReadAlbumTags_emptyFolder(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadAlbumTags(dir); err == nil {
		t.Fatal("expected error for empty folder")
	}
}

func TestReadAlbumTags_unreadableFileUsesFilename(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "01. Lifelike.m4a")
	if err := os.WriteFile(bad, []byte("not a real m4a"), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := ReadAlbumTags(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(res.Tracks))
	}
	if res.Tracks[0].Title != "Lifelike" {
		t.Fatalf("expected title from filename, got %q", res.Tracks[0].Title)
	}
	if res.Tracks[0].TrackNumber != 1 {
		t.Fatalf("expected track 1, got %d", res.Tracks[0].TrackNumber)
	}
	if len(res.Skipped) != 1 {
		t.Fatalf("expected 1 skipped entry, got %v", res.Skipped)
	}
}

func TestWriteAlbumBatch_empty(t *testing.T) {
	res := WriteAlbumBatch(TagAlbumBatchInput{})
	if res.Saved != 0 || res.Summary == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
