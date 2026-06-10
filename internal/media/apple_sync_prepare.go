package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhaarey/go-mp4tag"
)

func isMissingMetaBoxErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "box not present") {
		return false
	}
	return strings.Contains(msg, "udta") ||
		strings.Contains(msg, "meta") ||
		strings.Contains(msg, "ilst") ||
		strings.Contains(msg, "stco")
}

// readMP4TagsSafe reads tags via go-mp4tag and converts internal panics into errors.
func readMP4TagsSafe(path string) (tags *mp4tag.MP4Tags, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("mp4tag read failed: %v", r)
		}
	}()
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return nil, err
	}
	defer mp4.Close()
	return mp4.Read()
}

// HasWritableAppleTags reports whether go-mp4tag can read/write iTunes metadata atoms.
func HasWritableAppleTags(path string) bool {
	_, err := readMP4TagsSafe(path)
	return err == nil
}

// WriteNormalizedCoverSidecar writes cover.jpg (iOS-safe JPEG) beside album tracks.
func WriteNormalizedCoverSidecar(dir, coverPath string) (string, error) {
	dir = strings.TrimSpace(dir)
	coverPath = strings.TrimSpace(coverPath)
	if dir == "" || coverPath == "" {
		return "", fmt.Errorf("missing folder or cover path")
	}
	data, err := os.ReadFile(coverPath)
	if err != nil {
		return "", err
	}
	norm, err := NormalizeCoverForApple(data)
	if err != nil {
		return "", err
	}
	sidecar := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(sidecar, norm, 0644); err != nil {
		return "", err
	}
	return sidecar, nil
}
