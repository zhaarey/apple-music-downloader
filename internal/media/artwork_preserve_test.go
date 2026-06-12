package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/zhaarey/go-mp4tag"
)

func TestPrepareCoverForEmbed_preservesJPEGWhenOptimizeOff(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 64, 64))
	src.Set(0, 0, color.RGBA{R: 220, G: 40, B: 90, A: 255})
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, src, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), raw.Bytes()...)
	opt := false
	data, format, err := PrepareCoverForEmbed(TrackTags{CoverData: original, CoverOptimize: &opt})
	if err != nil {
		t.Fatal(err)
	}
	if format != mp4tag.ImageTypeJPEG {
		t.Fatalf("expected jpeg format, got %v", format)
	}
	if !bytes.Equal(data, original) {
		t.Fatal("expected original JPEG bytes unchanged when optimize is off")
	}
}

func TestNormalizeCoverWithOptions_passThroughWhenSmall(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 32, 32))
	src.Set(0, 0, color.RGBA{R: 10, G: 200, B: 30, A: 255})
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, src, nil); err != nil {
		t.Fatal(err)
	}
	original := raw.Bytes()
	out, err := NormalizeCoverWithOptions(original, CoverDownscaleOnlyOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, original) {
		t.Fatal("expected pass-through for small image")
	}
}
