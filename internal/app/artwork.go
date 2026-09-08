package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (r *Runner) saveAnimatedArtwork(dir string, squareVideoURL string, tallVideoURL string) {
	if squareVideoURL != "" {
		if err := r.downloadAnimatedArtwork(dir, "square", squareVideoURL); err != nil {
			fmt.Printf("no motion video square: %v\n", err)
		}
		if r.Config.EmbyAnimatedArtwork {
			cmd := exec.Command("ffmpeg", "-i", filepath.Join(dir, "square_animated_artwork.mp4"), "-vf", "scale=440:-1", "-r", "24", "-f", "gif", filepath.Join(dir, "folder.jpg"))
			if err := cmd.Run(); err != nil {
				fmt.Printf("animated artwork square to gif err: %v\n", err)
			}
		}
	}
	if tallVideoURL != "" {
		if err := r.downloadAnimatedArtwork(dir, "tall", tallVideoURL); err != nil {
			fmt.Printf("no motion video tall: %v\n", err)
		}
	}
}

func (r *Runner) downloadAnimatedArtwork(dir string, orientation string, videoURL string) error {
	videoURL, err := r.extractVideo(videoURL)
	if err != nil {
		return err
	}
	output := filepath.Join(dir, orientation+"_animated_artwork.mp4")
	exists, err := fileExists(output)
	if err != nil {
		return fmt.Errorf("check existing artwork: %w", err)
	}
	if exists {
		fmt.Printf("Animated artwork %s already exists locally.\n", orientation)
		return nil
	}
	fmt.Printf("Animation Artwork %s Downloading...\n", orientation)
	cmd := exec.Command("ffmpeg", "-loglevel", "quiet", "-y", "-i", videoURL, "-c", "copy", output)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download animated artwork: %w", err)
	}
	fmt.Printf("Animation Artwork %s Downloaded\n", orientation)
	return nil
}

func sanitizeFolderName(name string) string {
	if name == "" {
		return ""
	}
	if name[len(name)-1] == '.' {
		name = name[:len(name)-1]
	}
	return forbiddenNames.ReplaceAllString(name, "_")
}

func (r *Runner) prepareCollectionFolder(base string, displayName string) (string, error) {
	if strings.HasSuffix(displayName, ".") {
		displayName = displayName[:len(displayName)-1]
	}
	displayName = strings.TrimSpace(displayName)
	folderPath := filepath.Join(base, sanitizeFolderName(displayName))
	if err := createDirectory(folderPath); err != nil {
		return "", err
	}
	return folderPath, nil
}

func (r *Runner) saveM3UPlaylist(dir string, name string) {
	startIdx := len(r.State.AddedTracks)
	if len(r.State.AddedTracks) > startIdx {
		if err := r.writeM3UPlaylist(dir, name, r.State.AddedTracks[startIdx:]); err != nil {
			fmt.Printf("Failed to write M3U8 playlist: %v\n", err)
		}
	}
}

func createDirectory(path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("create directory %s: %w", path, err)
	}
	return nil
}
