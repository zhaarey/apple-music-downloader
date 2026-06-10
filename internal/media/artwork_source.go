package media

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PreparedArtworkResult is a normalized JPEG written to disk plus accent analysis.
type PreparedArtworkResult struct {
	Path string `json:"path"`
	ArtworkAccentAnalysis
}

// LoadCoverBytes reads artwork from a local path or HTTP(S) URL.
func LoadCoverBytes(coverPath, coverURL string) ([]byte, error) {
	coverPath = strings.TrimSpace(coverPath)
	if coverPath != "" {
		data, err := os.ReadFile(coverPath)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("empty cover file")
		}
		return data, nil
	}
	coverURL = strings.TrimSpace(coverURL)
	if coverURL == "" {
		return nil, fmt.Errorf("no artwork path or URL")
	}
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Get(coverURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover URL returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty cover from URL")
	}
	return data, nil
}

// AnalyzeArtworkSource inspects cover bytes from path or URL.
func AnalyzeArtworkSource(coverPath, coverURL string, includeOptimizedPreview bool) (ArtworkAccentAnalysis, error) {
	data, err := LoadCoverBytes(coverPath, coverURL)
	if err != nil {
		return ArtworkAccentAnalysis{}, err
	}
	return AnalyzeArtworkAccent(data, includeOptimizedPreview)
}

// PrepareOptimizedCoverToTemp normalizes cover bytes and writes a temp JPEG for reuse.
func PrepareOptimizedCoverToTemp(coverPath, coverURL string) (PreparedArtworkResult, error) {
	data, err := LoadCoverBytes(coverPath, coverURL)
	if err != nil {
		return PreparedArtworkResult{}, err
	}
	norm, err := NormalizeCoverForApple(data)
	if err != nil {
		return PreparedArtworkResult{}, err
	}
	f, err := os.CreateTemp("", "aura-cover-*.jpg")
	if err != nil {
		return PreparedArtworkResult{}, err
	}
	path := f.Name()
	if _, err := f.Write(norm); err != nil {
		f.Close()
		os.Remove(path)
		return PreparedArtworkResult{}, err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return PreparedArtworkResult{}, err
	}
	analysis, err := AnalyzeArtworkAccent(norm, false)
	if err != nil {
		os.Remove(path)
		return PreparedArtworkResult{}, err
	}
	return PreparedArtworkResult{
		Path:                  filepath.Clean(path),
		ArtworkAccentAnalysis: analysis,
	}, nil
}
