package media

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhaarey/go-mp4tag"
)

// MaxCoverEdgePx is the longest side Apple devices reliably show after iCloud Music Library sync.
const MaxCoverEdgePx = 3000

// TargetCoverEdgePx is the preferred longest side when normalizing for Apple Music import.
const TargetCoverEdgePx = 1200

var albumCoverNames = []string{"cover.jpg", "cover.jpeg", "cover.png", "folder.jpg", "Folder.jpg", "Folder.png"}

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
	target := float64(TargetCoverEdgePx)
	if maxEdge > MaxCoverEdgePx {
		scale = MaxCoverEdgePx / maxEdge
	} else if maxEdge > target {
		scale = target / maxEdge
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

// CoverHash returns a short SHA256 prefix for comparing embedded artwork across tracks.
func CoverHash(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

// FindAlbumCoverFile returns a sidecar cover path in dir if present.
func FindAlbumCoverFile(dir string) string {
	for _, name := range albumCoverNames {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Size() > 0 {
			return p
		}
	}
	return ""
}

// ReadNormalizedEmbeddedCover returns normalized JPEG bytes from a track's primary embedded cover.
func ReadNormalizedEmbeddedCover(path string) ([]byte, error) {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return nil, err
	}
	defer mp4.Close()
	tags, err := mp4.Read()
	if err != nil || tags == nil || len(tags.Pictures) == 0 {
		return nil, os.ErrNotExist
	}
	pic := tags.Pictures[0]
	if len(tags.Pictures) > 1 {
		pic = tags.Pictures[len(tags.Pictures)-1]
	}
	return NormalizeCoverForApple(pic.Data)
}

// TrackArtworkAlreadyPrepared reports whether the file already has the target cover as a single JPEG.
func TrackArtworkAlreadyPrepared(path string, targetCover []byte) bool {
	if len(targetCover) == 0 {
		return false
	}
	targetHash := CoverHash(targetCover)
	info, err := ReadAudioTags(path)
	if err != nil {
		return false
	}
	if info.ArtworkCount != 1 {
		return false
	}
	mime := strings.ToLower(info.ArtworkMime)
	if mime != "image/jpeg" && mime != "image/jpg" {
		return false
	}
	h, err := EmbeddedCoverHash(path)
	if err != nil {
		return false
	}
	return h == targetHash
}

// WriteTrackArtworkOnly replaces embedded cover with normalized JPEG and leaves text tags unchanged.
func WriteTrackArtworkOnly(path string, coverData []byte) error {
	if len(coverData) == 0 {
		return fmt.Errorf("no cover data")
	}
	norm, err := NormalizeCoverForApple(coverData)
	if err != nil {
		return err
	}
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return err
	}
	defer mp4.Close()
	existing, err := mp4.Read()
	if err != nil {
		return err
	}
	if existing == nil {
		existing = &mp4tag.MP4Tags{}
	}
	existing.Pictures = []*mp4tag.MP4Picture{{Format: mp4tag.ImageTypeJPEG, Data: norm}}
	return mp4.Write(existing, []string{"allpictures"})
}

// EmbeddedCoverHash reads normalized cover bytes from an M4A for album consistency checks.
func EmbeddedCoverHash(path string) (string, error) {
	norm, err := ReadNormalizedEmbeddedCover(path)
	if err != nil {
		return "", err
	}
	return CoverHash(norm), nil
}
