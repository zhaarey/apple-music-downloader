package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"main/internal/appstate"
	"main/internal/logging"
	"main/internal/media"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TagDropResolve tells the Tag Editor how to open a drag-and-drop payload.
type TagDropResolve struct {
	Mode    string `json:"mode"` // "single" or "album"
	Path    string `json:"path"`
	Message string `json:"message,omitempty"`
}

func isTagAudioExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".m4a", ".mp4", ".m4b":
		return true
	default:
		return false
	}
}

// TagResolveDrop classifies OS file-drop paths into single-file or album-folder mode.
func (a *App) TagResolveDrop(paths []string) (TagDropResolve, error) {
	var dirs []string
	var audios []string

	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		stat, err := os.Stat(p)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			dirs = append(dirs, p)
			continue
		}
		if isTagAudioExt(filepath.Ext(p)) {
			audios = append(audios, p)
		}
	}

	if len(dirs) == 0 && len(audios) == 0 {
		return TagDropResolve{}, fmt.Errorf("drop an .m4a/.mp4 file or an album folder")
	}

	if len(dirs) > 0 {
		msg := ""
		if len(dirs) > 1 {
			msg = fmt.Sprintf("Using first folder (%d dropped).", len(dirs))
		}
		return TagDropResolve{Mode: "album", Path: dirs[0], Message: msg}, nil
	}

	if len(audios) == 1 {
		return TagDropResolve{Mode: "single", Path: audios[0]}, nil
	}

	parent := filepath.Dir(audios[0])
	sameFolder := true
	for _, p := range audios[1:] {
		if filepath.Dir(p) != parent {
			sameFolder = false
			break
		}
	}
	if sameFolder {
		return TagDropResolve{
			Mode:    "album",
			Path:    parent,
			Message: fmt.Sprintf("Opened album folder from %d dropped tracks.", len(audios)),
		}, nil
	}

	return TagDropResolve{
		Mode:    "single",
		Path:    audios[0],
		Message: "Tracks are from different folders — opened the first file. Drop one folder for album bulk.",
	}, nil
}

func (a *App) TagPickAudioFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select audio or music video",
		Filters: []runtime.FileFilter{
			{DisplayName: "Apple Music media", Pattern: "*.m4a;*.mp4;*.m4b"},
			{DisplayName: "Music video (MP4)", Pattern: "*.mp4"},
			{DisplayName: "Audio (M4A)", Pattern: "*.m4a;*.m4b"},
		},
	})
}

func (a *App) TagPickSaveMediaFile(sourcePath string) (string, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return "", fmt.Errorf("no source file")
	}
	ext := strings.ToLower(filepath.Ext(sourcePath))
	if ext == "" {
		ext = ".m4a"
	}
	base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save tagged file as",
		DefaultFilename: base + ext,
		Filters: []runtime.FileFilter{
			{DisplayName: "Media file", Pattern: "*" + ext},
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

func (a *App) TagFindAlbumCover(folder string) (string, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return "", fmt.Errorf("no folder path")
	}
	if sidecar := media.FindAlbumCoverFile(folder); sidecar != "" {
		return sidecar, nil
	}
	return "", fmt.Errorf("no cover.jpg or folder.jpg in this folder")
}

func (a *App) TagAnalyzeArtwork(path string) (media.ArtworkAccentAnalysis, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return media.ArtworkAccentAnalysis{}, fmt.Errorf("no image path")
	}
	return media.AnalyzeArtworkFilePath(path, true)
}

func (a *App) TagAnalyzeEmbeddedArtwork(audioPath string) (media.ArtworkAccentAnalysis, error) {
	audioPath = strings.TrimSpace(audioPath)
	if audioPath == "" {
		return media.ArtworkAccentAnalysis{}, fmt.Errorf("no audio path")
	}
	return media.AnalyzeEmbeddedArtworkAccent(audioPath, true)
}

func (a *App) TagPreviewOptimizedArtwork(path string) (media.ArtworkAccentAnalysis, error) {
	return a.TagAnalyzeArtwork(path)
}

func (a *App) TagAnalyzeArtworkSource(coverPath, coverURL string) (media.ArtworkAccentAnalysis, error) {
	return media.AnalyzeArtworkSource(strings.TrimSpace(coverPath), strings.TrimSpace(coverURL), true)
}

func (a *App) TagApplyOptimizedArtwork(coverPath, coverURL string) (media.PreparedArtworkResult, error) {
	return media.PrepareOptimizedCoverToTemp(strings.TrimSpace(coverPath), strings.TrimSpace(coverURL))
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
	stat, err := os.Stat(path)
	if err != nil {
		return media.AudioTagInfo{}, err
	}
	if stat.IsDir() {
		return media.AudioTagInfo{}, fmt.Errorf("path is a folder, not a file")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".m4a" && ext != ".mp4" && ext != ".m4b" {
		return media.AudioTagInfo{}, fmt.Errorf("unsupported file type %s (use .m4a)", ext)
	}
	tags, err := media.ReadAudioTags(path)
	if err != nil {
		return media.AudioTagInfo{}, err
	}
	if strings.EqualFold(ext, ".mp4") {
		tags = media.EnrichVideoTagInfo(a.eng.GetConfig().FFmpegPath, path, tags)
	}
	appstate.RememberRecentFile(path)
	return tags, nil
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
	outPath := strings.TrimSpace(input.OutputPath)
	if outPath == "" {
		outPath = input.Path
	}
	logging.Info("TagWriteFile updated %s", outPath)
	appstate.RememberRecentFile(outPath)
	tags, err := media.ReadAudioTags(outPath)
	if err != nil {
		return media.AudioTagInfo{}, err
	}
	if strings.EqualFold(filepath.Ext(outPath), ".mp4") {
		tags = media.EnrichVideoTagInfo(a.eng.GetConfig().FFmpegPath, outPath, tags)
	}
	return tags, nil
}

func (a *App) TagReadAlbumFolder(folder string) (result media.AlbumFolderReadResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.LogPanic("TagReadAlbumFolder", r)
			err = fmt.Errorf("failed to read album folder: %v", r)
			result = media.AlbumFolderReadResult{}
		}
	}()
	folder = filepath.Clean(strings.TrimSpace(folder))
	if folder == "" {
		return result, fmt.Errorf("no folder selected")
	}
	result, err = media.ReadAlbumTags(folder)
	if err != nil {
		return result, err
	}
	for _, info := range result.Tracks {
		appstate.RememberRecentFile(info.Path)
	}
	return result, nil
}

func (a *App) TagWriteAlbumBatch(input media.TagAlbumBatchInput) (media.TagAlbumBatchResult, error) {
	defer func() {
		if r := recover(); r != nil {
			logging.LogPanic("TagWriteAlbumBatch", r)
		}
	}()
	if len(input.Tracks) == 0 {
		return media.TagAlbumBatchResult{}, fmt.Errorf("no tracks to save")
	}
	res := media.WriteAlbumBatch(input)
	for _, tr := range input.Tracks {
		if tr.Path != "" && res.Saved > 0 {
			appstate.RememberRecentFile(tr.Path)
		}
	}
	if res.Saved == 0 && len(res.Errors) > 0 {
		return res, fmt.Errorf("%s", res.Summary)
	}
	logging.Info("TagWriteAlbumBatch folder=%s saved=%d errors=%d", input.Folder, res.Saved, len(res.Errors))
	return res, nil
}

// TagLocalFileURL returns a webview-safe URL for local file preview (audio or images).
func (a *App) TagLocalFileURL(path string) string {
	return localMediaURL(path)
}
