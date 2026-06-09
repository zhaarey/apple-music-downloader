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

func TestWriteAlbumBatch_empty(t *testing.T) {
	res := WriteAlbumBatch(TagAlbumBatchInput{})
	if res.Saved != 0 || res.Summary == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
