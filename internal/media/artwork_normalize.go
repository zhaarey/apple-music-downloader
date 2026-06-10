package media

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"math"
)

// CoverNormalizeOptions controls Apple Music / iOS album-view artwork preparation.
type CoverNormalizeOptions struct {
	OptimizeForAppleUI bool
	SquareCrop         bool
	TrimLetterbox      bool
	SaturationBoost    float64
	TargetEdgePx       int
}

// DefaultCoverNormalizeOptions is used when the user opts into Apple Music UI optimization.
func DefaultCoverNormalizeOptions() CoverNormalizeOptions {
	return CoverNormalizeOptions{
		OptimizeForAppleUI: true,
		SquareCrop:         true,
		TrimLetterbox:      true,
		SaturationBoost:    0,
		TargetEdgePx:       TargetCoverEdgePx,
	}
}

// CoverDownscaleOnlyOptions shrinks only when above MaxCoverEdgePx — no crop or color changes.
func CoverDownscaleOnlyOptions() CoverNormalizeOptions {
	return CoverNormalizeOptions{
		TargetEdgePx: MaxCoverEdgePx,
	}
}

// NormalizeCoverWithOptions decodes, optionally optimizes for iOS album UI, and encodes baseline JPEG.
func NormalizeCoverWithOptions(data []byte, opts CoverNormalizeOptions) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty cover data")
	}
	if !opts.OptimizeForAppleUI {
		if ok, err := coverWithinEdgeLimit(data, opts.targetEdge()); err == nil && ok {
			return data, nil
		}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode cover: %w", err)
	}
	img = toRGBA(img)
	if opts.OptimizeForAppleUI {
		if opts.TrimLetterbox {
			img = trimUniformBorders(img)
		}
		if opts.SquareCrop {
			img = centerSquareCrop(img)
		}
	}
	quality := coverJPEGQuality
	if !opts.OptimizeForAppleUI {
		quality = 95
	}
	return encodeCoverJPEG(resizeCoverImage(img, opts.targetEdge()), quality)
}

func coverWithinEdgeLimit(data []byte, maxEdge int) (bool, error) {
	if maxEdge <= 0 {
		maxEdge = MaxCoverEdgePx
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return false, err
	}
	w, h := cfg.Width, cfg.Height
	if w <= 0 || h <= 0 {
		return false, fmt.Errorf("invalid cover dimensions")
	}
	long := w
	if h > long {
		long = h
	}
	return long <= maxEdge, nil
}

func (o CoverNormalizeOptions) targetEdge() int {
	if o.TargetEdgePx > 0 {
		return o.TargetEdgePx
	}
	return TargetCoverEdgePx
}

func encodeCoverJPEG(img image.Image, quality int) ([]byte, error) {
	if quality <= 0 {
		quality = coverJPEGQuality
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("encode cover jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func resizeCoverImage(img image.Image, targetEdge int) image.Image {
	if targetEdge <= 0 {
		targetEdge = TargetCoverEdgePx
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return img
	}
	maxEdge := math.Max(float64(w), float64(h))
	scale := 1.0
	if maxEdge > float64(MaxCoverEdgePx) {
		scale = float64(MaxCoverEdgePx) / maxEdge
	} else if maxEdge > float64(targetEdge) {
		scale = float64(targetEdge) / maxEdge
	}
	if scale >= 1.0 {
		return img
	}
	nw := int(math.Round(float64(w) * scale))
	nh := int(math.Round(float64(h) * scale))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return resizeNearest(img, nw, nh)
}

func centerSquareCrop(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == h {
		return img
	}
	side := w
	if h < w {
		side = h
	}
	x0 := b.Min.X + (w-side)/2
	y0 := b.Min.Y + (h-side)/2
	crop := image.Rect(0, 0, side, side)
	out := image.NewRGBA(crop)
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			out.Set(x, y, img.At(x0+x, y0+y))
		}
	}
	return out
}

func trimUniformBorders(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 8 || h < 8 {
		return img
	}
	top := 0
	for top < h/3 && rowIsUniform(img, b.Min.X, b.Min.Y+top, w) {
		top++
	}
	bottom := 0
	for bottom < h/3 && rowIsUniform(img, b.Min.X, b.Min.Y+h-1-bottom, w) {
		bottom++
	}
	left := 0
	for left < w/3 && colIsUniform(img, b.Min.X+left, b.Min.Y, h) {
		left++
	}
	right := 0
	for right < w/3 && colIsUniform(img, b.Min.X+w-1-right, b.Min.Y, h) {
		right++
	}
	nw := w - left - right
	nh := h - top - bottom
	if nw < 32 || nh < 32 {
		return img
	}
	if top == 0 && bottom == 0 && left == 0 && right == 0 {
		return img
	}
	out := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			out.Set(x, y, img.At(b.Min.X+left+x, b.Min.Y+top+y))
		}
	}
	return out
}

func rowIsUniform(img image.Image, x0, y, width int) bool {
	r0, g0, b0, _ := img.At(x0, y).RGBA()
	for x := 1; x < width; x++ {
		r, g, b, _ := img.At(x0+x, y).RGBA()
		if colorDist(r0, g0, b0, r, g, b) > 1200 {
			return false
		}
	}
	return true
}

func colIsUniform(img image.Image, x, y0, height int) bool {
	r0, g0, b0, _ := img.At(x, y0).RGBA()
	for y := 1; y < height; y++ {
		r, g, b, _ := img.At(x, y0+y).RGBA()
		if colorDist(r0, g0, b0, r, g, b) > 1200 {
			return false
		}
	}
	return true
}

func colorDist(r0, g0, b0, r1, g1, b1 uint32) int64 {
	dr := int64(r0) - int64(r1)
	dg := int64(g0) - int64(g1)
	db := int64(b0) - int64(b1)
	return dr*dr + dg*dg + db*db
}

func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, img, b.Min, draw.Src)
	return out
}
