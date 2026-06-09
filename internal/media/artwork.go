package media

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// MaxCoverEdgePx is the longest side Apple devices reliably show after iCloud Music Library sync.
const MaxCoverEdgePx = 3000

const coverJPEGQuality = 90

// NormalizeCoverForApple resizes oversized images and encodes as JPEG for iOS Music / iCloud sync.
func NormalizeCoverForApple(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty cover data")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode cover: %w", err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid cover dimensions")
	}

	scale := 1.0
	maxEdge := math.Max(float64(w), float64(h))
	if maxEdge > MaxCoverEdgePx {
		scale = MaxCoverEdgePx / maxEdge
	}

	var out image.Image = img
	if scale < 1.0 {
		nw := int(math.Round(float64(w) * scale))
		nh := int(math.Round(float64(h) * scale))
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
		out = resizeNearest(img, nw, nh)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: coverJPEGQuality}); err != nil {
		return nil, fmt.Errorf("encode cover jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func resizeNearest(src image.Image, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	for y := 0; y < height; y++ {
		sy := sb.Min.Y + y*sh/height
		for x := 0; x < width; x++ {
			sx := sb.Min.X + x*sw/width
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// PrepareCoverBytes loads, normalizes, and returns JPEG bytes for embedding in M4A/MP4.
func PrepareCoverBytes(tags TrackTags) ([]byte, error) {
	raw, _, err := resolveCover(tags)
	if err != nil {
		return nil, err
	}
	return NormalizeCoverForApple(raw)
}

// PrepareCoverFile normalizes cover bytes and writes a temp JPEG for ffmpeg attachment.
func PrepareCoverFile(tags TrackTags) (path string, cleanup func(), err error) {
	data, err := PrepareCoverBytes(tags)
	if err != nil {
		return "", nil, err
	}
	f, err := os.CreateTemp("", ".amd-cover-*.jpg")
	if err != nil {
		return "", nil, err
	}
	coverPath := f.Name()
	if _, werr := f.Write(data); werr != nil {
		f.Close()
		os.Remove(coverPath)
		return "", nil, werr
	}
	if err := f.Close(); err != nil {
		os.Remove(coverPath)
		return "", nil, err
	}
	return coverPath, func() { os.Remove(coverPath) }, nil
}
