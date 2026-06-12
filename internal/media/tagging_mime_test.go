package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/zhaarey/go-mp4tag"
)

func TestDetectImageMIME_sniffsBytes(t *testing.T) {
	var jpg bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 200, G: 40, B: 80, A: 255})
	if err := jpeg.Encode(&jpg, img, nil); err != nil {
		t.Fatal(err)
	}
	if got := DetectImageMIME(jpg.Bytes(), 0); got != "image/jpeg" {
		t.Fatalf("expected jpeg, got %q", got)
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	if got := DetectImageMIME(pngBuf.Bytes(), mp4tag.ImageTypeJPEG); got != "image/png" {
		t.Fatalf("expected png sniff to win over format enum, got %q", got)
	}
}
