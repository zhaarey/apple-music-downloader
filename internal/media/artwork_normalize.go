package media

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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

// DefaultCoverNormalizeOptions is used for Tag Editor saves and download embeds.
func DefaultCoverNormalizeOptions() CoverNormalizeOptions {
	return CoverNormalizeOptions{
		OptimizeForAppleUI: true,
		SquareCrop:         true,
		TrimLetterbox:      true,
		SaturationBoost:    0.08,
		TargetEdgePx:       TargetCoverEdgePx,
	}
}

// LegacyCoverNormalizeOptions only downscales oversized images (no square crop).
func LegacyCoverNormalizeOptions() CoverNormalizeOptions {
	return CoverNormalizeOptions{
		TargetEdgePx: TargetCoverEdgePx,
	}
}

// NormalizeCoverWithOptions decodes, optionally optimizes for iOS album UI, and encodes baseline JPEG.
func NormalizeCoverWithOptions(data []byte, opts CoverNormalizeOptions) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty cover data")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode cover: %w", err)
	}
	if opts.OptimizeForAppleUI {
		if opts.TrimLetterbox {
			img = trimUniformBorders(img)
		}
		if opts.SquareCrop {
			img = centerSquareCrop(img)
		}
		if opts.SaturationBoost > 0 {
			img = boostSaturation(toRGBA(img), opts.SaturationBoost)
		}
	}
	return encodeCoverJPEG(resizeCoverImage(img, opts.targetEdge()), coverJPEGQuality)
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
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}
	return out
}

func boostSaturation(img *image.RGBA, amount float64) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			h, s, l := rgbToHSL(float64(r)/65535, float64(g)/65535, float64(bl)/65535)
			s = math.Min(1, s*(1+amount))
			nr, ng, nb := hslToRGB(h, s, l)
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(nr * 255)),
				G: uint8(math.Round(ng * 255)),
				B: uint8(math.Round(nb * 255)),
				A: uint8(a >> 8),
			})
		}
	}
	return out
}

func rgbToHSL(r, g, b float64) (h, s, l float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2
	if max == min {
		return 0, 0, l
	}
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}
	switch max {
	case r:
		h = (g-b)/d + func() float64 {
			if g < b {
				return 6
			}
			return 0
		}()
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h /= 6
	return h, s, l
}

func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	return hueToRGB(p, q, h+1/3), hueToRGB(p, q, h), hueToRGB(p, q, h-1/3)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}
