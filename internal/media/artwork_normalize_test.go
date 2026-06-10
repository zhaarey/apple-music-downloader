package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestNormalizeCoverWithOptions_squareCrop(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 200, G: 40, B: 80, A: 255})
		}
	}
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, src, nil); err != nil {
		t.Fatal(err)
	}
	out, err := NormalizeCoverWithOptions(raw.Bytes(), DefaultCoverNormalizeOptions())
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w != h {
		t.Fatalf("expected square output, got %dx%d", w, h)
	}
}

func TestAnalyzeArtworkAccent_lowSaturation(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	gray := color.RGBA{R: 120, G: 120, B: 120, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, gray)
		}
	}
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, src, nil); err != nil {
		t.Fatal(err)
	}
	analysis, err := AnalyzeArtworkAccent(raw.Bytes(), false)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.AccentReady {
		t.Fatal("expected accent warnings for gray image")
	}
}
