package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	appconfig "main/internal/config"
	"main/internal/events"
	"main/utils/structs"
)

type ytDlpEntry struct {
	Type      string       `json:"_type"`
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Uploader  string       `json:"uploader"`
	Channel   string       `json:"channel"`
	Duration  float64      `json:"duration"`
	Thumbnail string       `json:"thumbnail"`
	URL       string       `json:"url"`
	WebpageURL string      `json:"webpage_url"`
	Entries   []ytDlpEntry `json:"entries"`
}

func IsYouTubeURL(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return false
	}
	return strings.Contains(raw, "youtube.com/") ||
		strings.Contains(raw, "youtu.be/") ||
		strings.Contains(raw, "youtube.com?") ||
		strings.HasPrefix(raw, "youtu.be")
}

func ytDlpPath() string {
	return appconfig.YtDlpPath(Config.YtDlpPath)
}

func youtubeOutputDir() string {
	dir := Config.YouTubeSaveFolder
	if dir == "" {
		dir = Config.AacSaveFolder
	}
	return dir
}

func (e *Engine) previewYouTube(raw string) PreviewResult {
	raw = normalizeYouTubeHost(raw)
	out := PreviewResult{
		URL:          raw,
		Type:         e.detectYouTubeType(raw),
		OutputFolder: youtubeOutputDir(),
	}
	if !IsYouTubeURL(raw) {
		out.Error = "Enter a YouTube video or playlist link (youtube.com or youtu.be)"
		return out
	}
	if err := e.ensureYtDlp(); err != nil {
		out.Error = err.Error()
		return out
	}

	info, err := e.fetchYouTubeInfo(raw)
	if err != nil {
		out.Error = err.Error()
		return out
	}

	if len(info.Entries) > 0 {
		return e.previewYouTubePlaylist(raw, info, out)
	}
	return e.previewYouTubeVideo(raw, info, out)
}

func (e *Engine) detectYouTubeType(raw string) string {
	if strings.Contains(raw, "list=") {
		return "YouTube Playlist"
	}
	return "YouTube Video"
}

func (e *Engine) fetchYouTubeInfo(raw string) (ytDlpEntry, error) {
	args := []string{
		"--no-warnings",
		"--no-playlist",
		"-J",
		raw,
	}
	if strings.Contains(raw, "list=") {
		args = []string{
			"--no-warnings",
			"--flat-playlist",
			"-J",
			raw,
		}
	}
	out, err := e.runYtDlp(args...)
	if err != nil {
		return ytDlpEntry{}, fmt.Errorf("could not read YouTube metadata: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(out))
	var info ytDlpEntry
	if err := decoder.Decode(&info); err != nil {
		// Flat playlist returns one JSON object per line.
		scanner := bufio.NewScanner(bytes.NewReader(out))
		var entries []ytDlpEntry
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var entry ytDlpEntry
			if json.Unmarshal([]byte(line), &entry) != nil {
				continue
			}
			if entry.Title == "" {
				entry.Title = entry.ID
			}
			entries = append(entries, entry)
		}
		if len(entries) == 0 {
			return ytDlpEntry{}, fmt.Errorf("could not parse YouTube metadata")
		}
		return ytDlpEntry{Type: "playlist", Entries: entries, Title: "YouTube playlist"}, nil
	}
	return info, nil
}

func (e *Engine) previewYouTubeVideo(raw string, info ytDlpEntry, out PreviewResult) PreviewResult {
	title := info.Title
	if title == "" {
		title = "YouTube video"
	}
	subtitle := info.Uploader
	if subtitle == "" {
		subtitle = info.Channel
	}
	out.Title = title
	out.Subtitle = subtitle
	out.ArtURL = info.Thumbnail
	out.TrackCount = 1
	out.CanSelectTracks = false
	out.TotalDuration = formatDuration(int(info.Duration * 1000))
	out.Tracks = []PreviewTrack{{
		Num:        1,
		ID:         info.ID,
		Name:       title,
		Artist:     subtitle,
		Type:       "youtube",
		Duration:   out.TotalDuration,
		DurationMs: int(info.Duration * 1000),
		URL:        raw,
	}}
	return out
}

func (e *Engine) previewYouTubePlaylist(raw string, info ytDlpEntry, out PreviewResult) PreviewResult {
	title := info.Title
	if title == "" {
		title = "YouTube playlist"
	}
	out.Title = title
	out.Subtitle = fmt.Sprintf("%d videos", len(info.Entries))
	out.CanSelectTracks = true
	tracks := make([]PreviewTrack, 0, len(info.Entries))
	totalMs := 0
	for i, entry := range info.Entries {
		name := entry.Title
		if name == "" {
			name = fmt.Sprintf("Video %d", i+1)
		}
		artist := entry.Uploader
		if artist == "" {
			artist = entry.Channel
		}
		durMs := int(entry.Duration * 1000)
		totalMs += durMs
		tracks = append(tracks, PreviewTrack{
			Num:        i + 1,
			ID:         entry.ID,
			Name:       name,
			Artist:     artist,
			Type:       "youtube",
			Duration:   formatDuration(durMs),
			DurationMs: durMs,
			URL:        entry.URL,
		})
	}
	out.Tracks = tracks
	out.TrackCount = len(tracks)
	out.TotalDuration = formatDuration(totalMs)
	if out.ArtURL == "" && len(info.Entries) > 0 {
		out.ArtURL = info.Entries[0].Thumbnail
	}
	return out
}

func (e *Engine) ensureYtDlp() error {
	bin := ytDlpPath()
	if _, err := exec.LookPath(bin); err != nil {
		if _, statErr := os.Stat(bin); statErr != nil {
			return fmt.Errorf("yt-dlp not found at %q — install from https://github.com/yt-dlp/yt-dlp/releases and add to PATH or dist/tools/", bin)
		}
	}
	return nil
}

func (e *Engine) ensureFFmpegForYouTube() error {
	bin := appconfig.FFmpegPath(Config.FFmpegPath)
	if _, err := exec.LookPath(bin); err != nil {
		if _, statErr := os.Stat(bin); statErr != nil {
			return fmt.Errorf("ffmpeg not found — YouTube audio extraction requires ffmpeg on PATH or in dist/tools/")
		}
	}
	return nil
}

func (e *Engine) validateYouTubeDownload(opts RunOptions) error {
	if err := e.ensureYtDlp(); err != nil {
		return err
	}
	if err := e.ensureFFmpegForYouTube(); err != nil {
		return err
	}
	saveFolder := youtubeOutputDir()
	if saveFolder == "" {
		return fmt.Errorf("YouTube output folder is not configured in Settings")
	}
	if err := os.MkdirAll(saveFolder, 0755); err != nil {
		return fmt.Errorf("cannot create YouTube output folder %q: %w", saveFolder, err)
	}
	for _, raw := range opts.URLs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return fmt.Errorf("download URL is empty")
		}
		if !IsYouTubeURL(raw) {
			return fmt.Errorf("YouTube mode requires a YouTube link (got %q)", raw)
		}
	}
	return nil
}

func (e *Engine) runYouTubeDownload(opts RunOptions) {
	currentEmitter = e.emitter
	defer func() { currentEmitter = nil }()

	counter = structs.Counter{}
	AddedTracks = nil
	okDict = make(map[string][]int)

	e.emit(events.Event{
		Type:    events.EventJobStart,
		Message: "Starting YouTube audio download (best quality)",
		Phase:   "youtube",
	})

	urls := opts.URLs
	if len(opts.ChildURLs) > 0 {
		urls = opts.ChildURLs
	}

	for i, raw := range urls {
		select {
		case <-e.ctx.Done():
			e.finishYouTubeJob("cancelled", fmt.Sprintf("Cancelled — %d completed before stop", counter.Success))
			return
		default:
		}
		e.log(fmt.Sprintf("YouTube %d of %d", i+1, len(urls)))
		if err := e.downloadYouTubeURL(raw, opts.SelectedTrackNums); err != nil {
			e.logError(err.Error())
			if Config.ExitOnError {
				break
			}
		}
	}

	msg := fmt.Sprintf("Finished: %d succeeded, %d failed (of %d attempted)",
		counter.Success, counter.Error, counter.Total)
	e.emit(events.Event{
		Type:    events.EventJobComplete,
		Message: msg,
		Phase:   jobCompletePhase(),
		Success: counter.Success,
		Error:   counter.Error + counter.Unavailable,
		Total_:  counter.Total,
	})
}

func (e *Engine) finishYouTubeJob(phase, msg string) {
	e.emit(events.Event{
		Type:    events.EventJobComplete,
		Message: msg,
		Phase:   phase,
		Success: counter.Success,
		Error:   counter.Error + counter.Unavailable,
		Total_:  counter.Total,
	})
}

func (e *Engine) downloadYouTubeURL(raw string, selectedNums []int) error {
	saveDir := youtubeOutputDir()
	outputTemplate := filepath.Join(saveDir, "%(title)s.%(ext)s")
	if strings.Contains(raw, "list=") || len(selectedNums) > 1 {
		outputTemplate = filepath.Join(saveDir, "%(playlist_index)02d - %(title)s.%(ext)s")
	}

	args := []string{
		"--newline",
		"--no-warnings",
		"--ffmpeg-location", appconfig.FFmpegPath(Config.FFmpegPath),
		"-f", "ba/b",
		"--extract-audio",
		"--audio-quality", "0",
		"--audio-format", "m4a",
		"--embed-thumbnail",
		"--embed-metadata",
		"--retries", "10",
		"--fragment-retries", "10",
		"-o", outputTemplate,
	}

	if len(selectedNums) > 0 {
		parts := make([]string, len(selectedNums))
		for i, n := range selectedNums {
			parts[i] = strconv.Itoa(n)
		}
		args = append(args, "--playlist-items", strings.Join(parts, ","))
	} else if !strings.Contains(raw, "list=") {
		args = append(args, "--no-playlist")
	}

	args = append(args, raw)

	counter.Total++
	trackLabel := e.DetectURLType(raw)
	e.emit(events.Event{
		Type:    events.EventTrackStart,
		Message: "Downloading audio…",
		Track:   trackLabel,
		Current: 1,
		Total:   1,
	})

	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytDlpPath(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		counter.Error++
		e.emitTrackFailed(trackLabel, 1, 1, err.Error())
		return err
	}
	if err := cmd.Start(); err != nil {
		counter.Error++
		e.emitTrackFailed(trackLabel, 1, 1, err.Error())
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "[download]") {
			e.log(line)
		}
	}
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		counter.Error++
		e.emitTrackFailed(trackLabel, 1, 1, msg)
		return fmt.Errorf("%s", msg)
	}

	counter.Success++
	e.emit(events.Event{
		Type:    events.EventTrackComplete,
		Message: "Saved to " + saveDir,
		Track:   trackLabel,
		Current: 1,
		Total:   1,
	})
	return nil
}

func (e *Engine) emitTrackFailed(label string, current, total int64, msg string) {
	e.emit(events.Event{
		Type:    events.EventTrackFailed,
		Message: msg,
		Track:   label,
		Current: current,
		Total:   total,
		Phase:   "failed",
	})
}

func (e *Engine) runYtDlp(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ytDlpPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return out, nil
}

func normalizeYouTubeHost(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	if parsed.Host == "youtu.be" && parsed.Path != "" {
		return "https://www.youtube.com/watch?v=" + strings.TrimPrefix(parsed.Path, "/")
	}
	return raw
}
