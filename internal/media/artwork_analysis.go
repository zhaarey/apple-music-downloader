package media

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// ArtworkAccentAnalysis reports whether cover pixels are likely to get iOS album accent UI.
type ArtworkAccentAnalysis struct {
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	IsSquare        bool    `json:"is_square"`
	MinEdgePx       int     `json:"min_edge_px"`
	AvgSaturation   float64 `json:"avg_saturation"`
	AvgLuminance    float64 `json:"avg_luminance"`
	AccentReady     bool    `json:"accent_ready"`
	Warnings        []string `json:"warnings"`
	Summary         string  `json:"summary"`
	OptimizedB64    string  `json:"optimized_b64,omitempty"`
	OptimizedMime   string  `json:"optimized_mime,omitempty"`
}

// AnalyzeArtworkAccent inspects raw image bytes for iOS album-view accent readiness.
func AnalyzeArtworkAccent(data []byte, includeOptimizedPreview bool) (ArtworkAccentAnalysis, error) {
	out := ArtworkAccentAnalysis{Warnings: []string{}}
	if len(data) == 0 {
		return out, fmt.Errorf("empty image")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return out, err
	}
	out.Width = cfg.Width
	out.Height = cfg.Height
	out.IsSquare = cfg.Width == cfg.Height
	out.MinEdgePx = cfg.Width
	if cfg.Height < out.MinEdgePx {
		out.MinEdgePx = cfg.Height
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return out, err
	}
	var satSum, lumSum float64
	var n float64
	b := img.Bounds()
	step := 1
	pixels := b.Dx() * b.Dy()
	if pixels > 250_000 {
		step = int(math.Ceil(math.Sqrt(float64(pixels) / 250_000)))
	}
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			r, g, bl, _ := img.At(x, y).RGBA()
			_, s, l := rgbToHSL(float64(r)/65535, float64(g)/65535, float64(bl)/65535)
			satSum += s
			lumSum += l
			n++
		}
	}
	if n > 0 {
		out.AvgSaturation = satSum / n
		out.AvgLuminance = lumSum / n
	}

	if !out.IsSquare {
		out.Warnings = append(out.Warnings, "Image is not square — iOS may sample letterbox edges and skip accent colors")
	}
	if out.MinEdgePx < 600 {
		out.Warnings = append(out.Warnings, "Resolution is low (<600px) — album art may look soft on device")
	}
	if out.AvgSaturation < 0.12 {
		out.Warnings = append(out.Warnings, "Very low saturation — iOS often uses a flat background instead of accent colors")
	}
	if out.AvgLuminance < 0.12 || out.AvgLuminance > 0.88 {
		out.Warnings = append(out.Warnings, "Very dark or very bright cover — accent UI may be disabled for readability")
	}

	out.AccentReady = len(out.Warnings) == 0
	if out.AccentReady {
		out.Summary = "Artwork looks suitable for iOS album accent colors."
	} else if len(out.Warnings) == 1 {
		out.Summary = out.Warnings[0] + " Enable optimize on save to improve odds."
	} else {
		out.Summary = fmt.Sprintf("%d issue(s) may prevent iOS accent colors — enable optimize on save.", len(out.Warnings))
	}

	if includeOptimizedPreview {
		if norm, err := NormalizeCoverWithOptions(data, DefaultCoverNormalizeOptions()); err == nil && len(norm) > 0 {
			out.OptimizedB64 = base64.StdEncoding.EncodeToString(norm)
			out.OptimizedMime = "image/jpeg"
		}
	}
	return out, nil
}

// AnalyzeArtworkFilePath analyzes an image file on disk.
func AnalyzeArtworkFilePath(path string, includeOptimizedPreview bool) (ArtworkAccentAnalysis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ArtworkAccentAnalysis{}, err
	}
	return AnalyzeArtworkAccent(data, includeOptimizedPreview)
}

// AnalyzeEmbeddedArtworkAccent analyzes raw embedded artwork in an M4A.
func AnalyzeEmbeddedArtworkAccent(m4aPath string, includeOptimizedPreview bool) (ArtworkAccentAnalysis, error) {
	raw, err := ReadEmbeddedCoverRaw(m4aPath)
	if err != nil {
		return ArtworkAccentAnalysis{}, err
	}
	return AnalyzeArtworkAccent(raw, includeOptimizedPreview)
}
