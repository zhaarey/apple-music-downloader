package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"main/internal/logging"
	"main/internal/media"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) TagPickAudioFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select audio file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Audio", Pattern: "*.m4a;*.mp4;*.m4b"},
		},
	})
}

func (a *App) TagPickArtworkFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select artwork image",
		Filters: []runtime.FileFilter{
			{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png"},
		},
	})
}

func (a *App) TagReadFile(path string) (media.AudioTagInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			logging.LogPanic("TagReadFile", r)
		}
	}()
	if path == "" {
		return media.AudioTagInfo{}, fmt.Errorf("no file selected")
	}
	info, err := os.Stat(path)
	if err != nil {
		return media.AudioTagInfo{}, err
	}
	if info.IsDir() {
		return media.AudioTagInfo{}, fmt.Errorf("path is a folder, not a file")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".m4a" && ext != ".mp4" && ext != ".m4b" {
		return media.AudioTagInfo{}, fmt.Errorf("unsupported file type %s (use .m4a)", ext)
	}
	return media.ReadAudioTags(path)
}

func (a *App) TagWriteFile(input media.WriteAudioTagsInput) (media.AudioTagInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			logging.LogPanic("TagWriteFile", r)
		}
	}()
	if input.Path == "" {
		return media.AudioTagInfo{}, fmt.Errorf("no file selected")
	}
	if !input.SortTags {
		input.SortTags = true
	}
	if err := media.WriteAudioTags(input); err != nil {
		return media.AudioTagInfo{}, err
	}
	logging.Info("TagWriteFile updated %s", input.Path)
	return media.ReadAudioTags(input.Path)
}

// TagLocalFileURL returns a webview-safe URL for local file preview (audio or images).
func (a *App) TagLocalFileURL(path string) string {
	return spliceMediaURL(path)
}
