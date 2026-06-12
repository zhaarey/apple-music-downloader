package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"main/internal/events"
	"main/utils/runv3"
	"main/utils/task"
)

func formatArtworkURL(template string) string {
	if template == "" {
		return ""
	}
	size := Config.CoverSize
	if size == "" {
		size = "500x500"
	}
	return strings.Replace(template, "{w}x{h}", size, 1)
}

func resolveTrackSelection(total int, interactive func() []int) []int {
	arr := make([]int, total)
	for i := 0; i < total; i++ {
		arr[i] = i + 1
	}
	if !dl_select {
		return arr
	}
	if len(guiSelectedTracks) > 0 {
		return guiSelectedTracks
	}
	if interactive != nil {
		return interactive()
	}
	return arr
}

func formatDuration(ms int) string {
	if ms <= 0 {
		return ""
	}
	sec := ms / 1000
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func effectiveAacType() string {
	if Config.AacType == "" {
		return "aac-lc"
	}
	return Config.AacType
}

func useAacLCDownload() bool {
	return dl_aac && effectiveAacType() == "aac-lc"
}

func formatAACError(err error) string {
	if err == nil {
		return ""
	}
	return runv3.UserFacingError(err)
}

func resolveCoverPath(track *task.Track) (string, error) {
	if track.CoverPath != "" {
		if ok, err := fileExists(track.CoverPath); err == nil && ok {
			return track.CoverPath, nil
		}
	}
	coverExt := Config.CoverFormat
	if coverExt == "" || coverExt == "original" {
		coverExt = "jpg"
	}
	candidates := []string{
		filepath.Join(track.SaveDir, "cover."+coverExt),
		filepath.Join(track.SaveDir, "cover.jpg"),
		filepath.Join(track.SaveDir, "cover.png"),
	}
	for _, path := range candidates {
		if ok, err := fileExists(path); err == nil && ok {
			return path, nil
		}
	}
	artURL := track.Resp.Attributes.Artwork.URL
	if artURL == "" && track.AlbumData.Attributes.Artwork.URL != "" {
		artURL = track.AlbumData.Attributes.Artwork.URL
	}
	if artURL == "" {
		return "", fmt.Errorf("no artwork available")
	}
	return writeCover(track.SaveDir, "cover", artURL)
}

func loadCoverBytes(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("cover path empty")
	}
	return os.ReadFile(path)
}

func emitTrackDone(track *task.Track, trackLabel, outcome, message string) {
	ev := events.Event{
		Message: message,
		Track:   trackLabel,
		Current: int64(track.TaskNum),
		Total:   int64(track.TaskTotal),
		Phase:   outcome,
	}
	switch outcome {
	case "success", "skipped":
		ev.Type = events.EventTrackComplete
		if outcome == "skipped" {
			ev.Message = message
		} else {
			ev.Message = fmt.Sprintf("Completed: %s", trackLabel)
		}
	case "unavailable", "failed":
		ev.Type = events.EventTrackFailed
	default:
		ev.Type = events.EventTrackFailed
	}
	emitDownload(ev)
	emitDownload(events.Event{Type: events.EventLog, Message: message})
}
