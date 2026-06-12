package media

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteNormalizedCoverSidecar(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "raw.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := WriteNormalizedCoverSidecar(dir, src)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(out) != "cover.jpg" {
		t.Fatalf("got %q", out)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 3 || data[0] != 0xff || data[1] != 0xd8 {
		t.Fatal("expected JPEG sidecar")
	}
}

func TestHasWritableAppleTags_missingFile(t *testing.T) {
	if HasWritableAppleTags(filepath.Join(t.TempDir(), "nope.m4a")) {
		t.Fatal("expected false for missing file")
	}
}
