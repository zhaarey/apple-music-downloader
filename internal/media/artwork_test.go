package media

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"testing"
)

func TestNormalizeCoverForApple_JPEGPassthrough(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, src, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	out, err := NormalizeCoverForApple(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected jpeg output")
	}
}

func TestNormalizeCoverForApple_DownscalesLarge(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4000, 2000))
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, src, nil); err != nil {
		t.Fatal(err)
	}
	out, err := NormalizeCoverForApple(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w > MaxCoverEdgePx || h > MaxCoverEdgePx {
		t.Fatalf("expected max edge %d, got %dx%d", MaxCoverEdgePx, w, h)
	}
}
