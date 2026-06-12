package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appconfig "main/internal/config"
)

// ReembedCoverMP4Box re-embeds cover JPEG via MP4Box (optional Windows/macOS sync hardening).
func ReembedCoverMP4Box(trackPath string, coverJPEG []byte) error {
	trackPath = strings.TrimSpace(trackPath)
	if trackPath == "" {
		return fmt.Errorf("no track path")
	}
	if len(coverJPEG) == 0 {
		return fmt.Errorf("no cover data")
	}
	mp4box := appconfig.MP4BoxPath()
	if _, err := exec.LookPath(mp4box); err != nil {
		if _, statErr := os.Stat(mp4box); statErr != nil {
			return fmt.Errorf("MP4Box not found: %w", err)
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(trackPath), ".aura-cover-*.jpg")
	if err != nil {
		return err
	}
	coverPath := tmp.Name()
	defer os.Remove(coverPath)
	if _, err := tmp.Write(coverJPEG); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cmd := exec.Command(mp4box, "-itags", "cover="+coverPath, trackPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("MP4Box cover re-embed: %s", msg)
	}
	return nil
}
