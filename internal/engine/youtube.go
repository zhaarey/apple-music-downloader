package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	appconfig "main/internal/config"
	"main/internal/events"
	"main/internal/media"
	"main/internal/youtube"
	"main/utils/structs"
)

type ytDlpEntry struct {
	Type       string       `json:"_type"`
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	Uploader   string       `json:"uploader"`
	Channel    string       `json:"channel"`
	Playlist   string       `json:"playlist"`
	Duration   float64      `json:"duration"`
	Thumbnail  string       `json:"thumbnail"`
	URL        string       `json:"url"`
	WebpageURL string       `json:"webpage_url"`
	Thumbnails []struct {
		URL string `json:"url"`
	} `json:"thumbnails"`
	Entries []ytDlpEntry `json:"entries"`
}

var (
	reYtDownloadPct  = regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`)
	reYtDownloadItem = regexp.MustCompile(`\[info\]\s+Downloading item (\d+) of (\d+)`)
	reYtPostProcess  = regexp.MustCompile(`\[(ExtractAudio|Merger|EmbedThumbnail|Metadata|FixupM3u8)\]`)
)

func IsYouTubeURL(raw string) bool {
	return youtube.IsURL(raw)
}

func useYouTubePipeline(opts RunOptions) bool {
	if opts.Quality == "youtube" {
		return true
	}
	for _, raw := range opts.URLs {
		if IsYouTubeURL(strings.TrimSpace(raw)) {
			return true
		}
	}
	for _, raw := range opts.ChildURLs {
		if IsYouTubeURL(strings.TrimSpace(raw)) {
			return true
		}
	}
	return false
}

func ytDlpPath() string {
	return appconfig.YtDlpPath(Config.YtDlpPath)
}

func youtubeOutputDir() string {
	return youtube.OutputDir(Config)
}

func ytDlpFFmpegArgs() []string {
	return youtube.FFmpegArgs(Config)
}

func bestThumbnail(entry ytDlpEntry) string {
	if entry.Thumbnail != "" {
		return entry.Thumbnail
	}
	if n := len(entry.Thumbnails); n > 0 {
		return entry.Thumbnails[n-1].URL
	}
	return ""
}

func entryUploader(entry ytDlpEntry) string {
	if entry.Uploader != "" {
		return entry.Uploader
	}
	return entry.Channel
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
	args := []string{"--no-warnings", "--no-playlist", "-J", raw}
	if strings.Contains(raw, "list=") {
		args = []string{"--no-warnings", "--flat-playlist", "-J", raw}
	}
	out, err := e.runYtDlpWithTimeout(300*time.Second, args...)
	if err != nil {
		return ytDlpEntry{}, fmt.Errorf("could not read YouTube metadata: %w", err)
	}

	var info ytDlpEntry
	if err := json.Unmarshal(out, &info); err != nil {
		return ytDlpEntry{}, fmt.Errorf("could not parse YouTube metadata: %w", err)
	}
	if len(info.Entries) > 0 {
		if info.Title == "" {
			info.Title = info.Playlist
		}
		if info.Title == "" {
			info.Title = "YouTube playlist"
		}
		return info, nil
	}
	if info.Title == "" && info.ID != "" {
		info.Title = info.ID
	}
	return info, nil
}

func (e *Engine) previewYouTubeVideo(raw string, info ytDlpEntry, out PreviewResult) PreviewResult {
	title := info.Title
	if title == "" {
		title = "YouTube video"
	}
	subtitle := entryUploader(info)
	if subtitle == "" {
		subtitle = "YouTube"
	}
	out.Title = title
	out.Subtitle = subtitle
	out.ArtURL = bestThumbnail(info)
	out.TrackCount = 1
	out.CanSelectTracks = false
	out.TotalDuration = formatDuration(int(info.Duration * 1000))
	year := youtube.DefaultYear()
	out.Tracks = []PreviewTrack{{
		Num:         1,
		ID:          info.ID,
		Name:        title,
		Artist:      subtitle,
		Type:        "youtube",
		Duration:    out.TotalDuration,
		DurationMs:  int(info.Duration * 1000),
		URL:         raw,
		ArtURL:      out.ArtURL,
		Album:       title,
		AlbumArtist: subtitle,
		Genre:       "DJ Mix",
		Year:        year,
		TrackNumber: 1,
		DiscNumber:  1,
	}}
	return out
}

func (e *Engine) previewYouTubePlaylist(raw string, info ytDlpEntry, out PreviewResult) PreviewResult {
	title := info.Title
	if title == "" {
		title = "YouTube playlist"
	}
	channel := entryUploader(info)
	subtitle := fmt.Sprintf("%d videos", len(info.Entries))
	if channel != "" {
		subtitle = fmt.Sprintf("%s · %s", channel, subtitle)
	}
	out.Title = title
	out.Subtitle = subtitle
	out.CanSelectTracks = true
	out.ArtURL = bestThumbnail(info)

	tracks := make([]PreviewTrack, 0, len(info.Entries))
	totalMs := 0
	year := youtube.DefaultYear()
	for i, entry := range info.Entries {
		name := entry.Title
		if name == "" {
			name = fmt.Sprintf("Video %d", i+1)
		}
		artist := entryUploader(entry)
		if artist == "" {
			artist = channel
		}
		durMs := int(entry.Duration * 1000)
		totalMs += durMs
		durationLabel := formatDuration(durMs)
		if durMs == 0 {
			durationLabel = "—"
		}
		tracks = append(tracks, PreviewTrack{
			Num:         i + 1,
			ID:          entry.ID,
			Name:        name,
			Artist:      artist,
			Type:        "youtube",
			Duration:    durationLabel,
			DurationMs:  durMs,
			URL:         entry.URL,
			ArtURL:      bestThumbnail(entry),
			Album:       title,
			AlbumArtist: channel,
			Genre:       "DJ Mix",
			Year:        year,
			TrackNumber: i + 1,
			DiscNumber:  1,
		})
	}
	out.Tracks = tracks
	out.TrackCount = len(tracks)
	if totalMs > 0 {
		out.TotalDuration = formatDuration(totalMs)
	}
	if out.ArtURL == "" && len(tracks) > 0 {
		out.ArtURL = tracks[0].ArtURL
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
	return appconfig.ValidateFFmpegForYouTube(Config.FFmpegPath)
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
		if IsAppleMusicURL(raw) {
			return fmt.Errorf("this is an Apple Music link — use the Apple Music tab to download it (got %q)", raw)
		}
		if !IsYouTubeURL(raw) {
			return fmt.Errorf("YouTube downloads require a youtube.com or youtu.be link (got %q)", raw)
		}
	}
	return nil
}

func (e *Engine) runYouTubeDownload(opts RunOptions) {
	currentEmitter = e.emitter
	defer func() { currentEmitter = nil }()

	counter = structs.Counter{}
	AddedTracks = nil
	lastYouTubeOutput = ""
	okDict = make(map[string][]int)

	var lastHandoff *youtube.HandoffPayload

	delivery := NormalizeYouTubeDelivery(opts.YouTubeDeliveryMode)

	jobMsg := delivery.JobStartMessage()
	e.emit(events.Event{
		Type:    events.EventJobStart,
		Message: jobMsg,
		Phase:   "youtube",
	})

	urls := opts.URLs
	if len(opts.ChildURLs) > 0 {
		urls = opts.ChildURLs
	}

	for i, raw := range urls {
		select {
		case <-e.ctx.Done():
			e.finishYouTubeJob("cancelled", fmt.Sprintf("Cancelled — %d completed before stop", counter.Success), nil)
			return
		default:
		}
		e.log(fmt.Sprintf("YouTube %d of %d", i+1, len(urls)))
		handoff, err := e.downloadYouTubeURL(raw, opts.SelectedTrackNums, delivery, opts.YouTubeMeta)
		if handoff != nil {
			lastHandoff = handoff
		}
		if err != nil {
			e.logError(err.Error())
			if Config.ExitOnError {
				break
			}
		}
	}

	msg := fmt.Sprintf("Finished: %d succeeded, %d failed (of %d attempted)",
		counter.Success, counter.Error, counter.Total)
	var eventHandoff *events.SpliceHandoff
	if lastHandoff != nil {
		eventHandoff = &events.SpliceHandoff{
			MasterPath:  lastHandoff.MasterPath,
			Album:       lastHandoff.Album,
			AlbumArtist: lastHandoff.AlbumArtist,
			Artist:      lastHandoff.Artist,
			Year:        lastHandoff.Year,
			Genre:       lastHandoff.Genre,
			ArtURL:      lastHandoff.ArtURL,
		}
	}
	master := lastHandoffPath(lastHandoff)
	e.emit(events.Event{
		Type:       events.EventJobComplete,
		Message:    msg,
		Phase:      jobCompletePhase(),
		Success:    counter.Success,
		Error:      counter.Error + counter.Unavailable,
		Total_:     counter.Total,
		MasterPath: master,
		OutputPath: lastJobOutputPath(master),
		Handoff:    eventHandoff,
	})
}

func lastHandoffPath(h *youtube.HandoffPayload) string {
	if h == nil {
		return ""
	}
	return h.MasterPath
}

func handoffFromMeta(outPath string, meta YouTubeDownloadMeta) *youtube.HandoffPayload {
	return &youtube.HandoffPayload{
		MasterPath:  outPath,
		Album:       meta.Album,
		AlbumArtist: meta.AlbumArtist,
		Artist:      meta.Artist,
		Year:        meta.Year,
		Genre:       meta.Genre,
		ArtURL:      meta.ArtURL,
	}
}

func (e *Engine) finishYouTubeJob(phase, msg string, handoff *youtube.HandoffPayload) {
	var eventHandoff *events.SpliceHandoff
	if handoff != nil {
		eventHandoff = &events.SpliceHandoff{
			MasterPath:  handoff.MasterPath,
			Album:       handoff.Album,
			AlbumArtist: handoff.AlbumArtist,
			Artist:      handoff.Artist,
			Year:        handoff.Year,
			Genre:       handoff.Genre,
			ArtURL:      handoff.ArtURL,
		}
	}
	master := lastHandoffPath(handoff)
	e.emit(events.Event{
		Type:       events.EventJobComplete,
		Message:    msg,
		Phase:      phase,
		Success:    counter.Success,
		Error:      counter.Error + counter.Unavailable,
		Total_:     counter.Total,
		MasterPath: master,
		OutputPath: lastJobOutputPath(master),
		Handoff:    eventHandoff,
	})
}

func appendYouTubePlaylistArgs(args []string, raw string, selectedNums []int) []string {
	if len(selectedNums) > 0 {
		parts := make([]string, len(selectedNums))
		for i, n := range selectedNums {
			parts[i] = strconv.Itoa(n)
		}
		return append(args, "--playlist-items", strings.Join(parts, ","))
	}
	if !strings.Contains(raw, "list=") {
		return append(args, "--no-playlist")
	}
	return args
}

func youtubeTempTemplate(tempDir string, isPlaylist bool) string {
	if isPlaylist {
		return filepath.Join(tempDir, "%(playlist_index)03d_%(id)s.%(ext)s")
	}
	return filepath.Join(tempDir, "%(id)s.%(ext)s")
}

func buildYouTubeAudioArgs(tempDir, raw string, selectedNums []int, isPlaylist bool) []string {
	args := []string{
		"--newline",
		"--no-warnings",
		"--progress",
	}
	args = append(args, ytDlpFFmpegArgs()...)
	args = append(args,
		// Audio-only: do NOT fall back to "b"/best (muxed video) — that breaks AAC step on DJ sets.
		"-f", "bestaudio/ba/ba.*",
		"--extract-audio",
		"--audio-quality", "0",
		"--audio-format", "best",
		"--embed-thumbnail",
		"--embed-metadata",
		"--retries", "10",
		"--fragment-retries", "10",
		"-o", youtubeTempTemplate(tempDir, isPlaylist),
	)
	return appendYouTubePlaylistArgs(args, raw, selectedNums)
}

func buildYouTubeVideoArgs(tempDir, raw string, selectedNums []int, isPlaylist bool) []string {
	args := []string{
		"--newline",
		"--no-warnings",
		"--progress",
	}
	args = append(args, ytDlpFFmpegArgs()...)
	args = append(args,
		// Prefer H.264 MP4 video + best audio; re-encoded to AAC stereo after download.
		"-f", "bv*[vcodec^=avc1][ext=mp4]+ba/b[ext=mp4]/bv*+ba/b",
		"-S", "vcodec:h264,res,acodec",
		"--merge-output-format", "mp4",
		"--embed-thumbnail",
		"--embed-metadata",
		"--retries", "10",
		"--fragment-retries", "10",
		"-o", youtubeTempTemplate(tempDir, isPlaylist),
	)
	return appendYouTubePlaylistArgs(args, raw, selectedNums)
}

func convertMetas(metas []YouTubeDownloadMeta) []youtube.DownloadMeta {
	out := make([]youtube.DownloadMeta, len(metas))
	for i, m := range metas {
		out[i] = youtube.DownloadMeta(m)
	}
	return out
}

func (e *Engine) downloadYouTubeURL(raw string, selectedNums []int, delivery YouTubeDeliveryMode, metas []YouTubeDownloadMeta) (*youtube.HandoffPayload, error) {
	saveDir := youtubeOutputDir()
	trackLabel := e.DetectURLType(raw)
	isPlaylist := strings.Contains(raw, "list=") || len(selectedNums) > 1
	multiTrack := isPlaylist || len(selectedNums) > 1
	metaMap := youtube.MetaByNum(convertMetas(metas))
	checkVideo := delivery.ExistingMediaIsVideo()

	resolveMeta := func(num int) youtube.DownloadMeta {
		if m, ok := metaMap[num]; ok {
			if m.TrackTotal == 0 && len(selectedNums) > 0 {
				m.TrackTotal = len(selectedNums)
			} else if m.TrackTotal == 0 && isPlaylist {
				m.TrackTotal = len(metas)
			}
			return m
		}
		return youtube.DownloadMeta{Num: num, TrackNumber: num, DiscNumber: 1, TrackTotal: 1}
	}

	if !isPlaylist {
		meta := resolveMeta(1)
		if num := 1; len(selectedNums) > 0 {
			num = selectedNums[0]
			meta = resolveMeta(num)
		}
		if path, rootHint, ok := youtubeExistingLocation(saveDir, meta, multiTrack, checkVideo); ok {
			counter.Success++
			counter.Total++
			msg := fmt.Sprintf("Already on disk: %s", meta.Title)
			if rootHint != "output folder" {
				msg = fmt.Sprintf("Already on disk (%s): %s", rootHint, meta.Title)
			}
			e.emitTrackSkipped(meta.Title, 1, 1, msg)
			e.log(fmt.Sprintf("Skipped (exists): %s", path))
			noteDownloadOutput(path)
			return handoffFromMeta(path, YouTubeDownloadMeta(meta)), nil
		}
	} else if len(selectedNums) > 0 {
		missing := filterMissingYouTubeTracks(saveDir, selectedNums, multiTrack, metaMap, delivery)
		for _, num := range selectedNums {
			if containsInt(missing, num) {
				continue
			}
			meta := resolveMeta(num)
			if path, rootHint, ok := youtubeExistingLocation(saveDir, meta, multiTrack, checkVideo); ok {
				counter.Success++
				msg := fmt.Sprintf("Already on disk: %s", meta.Title)
				if rootHint != "output folder" {
					msg = fmt.Sprintf("Already on disk (%s): %s", rootHint, meta.Title)
				}
				e.emitTrackSkipped(meta.Title, int64(num), int64(len(selectedNums)), msg)
				e.log(fmt.Sprintf("Skipped (exists): %s", path))
				noteDownloadOutput(path)
			}
		}
		if len(missing) == 0 {
			counter.Total++
			e.emit(events.Event{
				Type:    events.EventTrackComplete,
				Message: "All selected tracks already on disk",
				Track:   trackLabel,
				Phase:   "skipped",
				Current: 1,
				Total:   1,
			})
			return nil, nil
		}
		selectedNums = missing
	}

	counter.Total++
	e.emit(events.Event{
		Type:    events.EventTrackStart,
		Message: "Preparing download…",
		Track:   trackLabel,
		Current: 1,
		Total:   1,
	})

	tempDir, err := os.MkdirTemp(saveDir, "yt-dl-*")
	if err != nil {
		counter.Error++
		e.emitTrackFailed(trackLabel, 1, 1, err.Error())
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	var handoff *youtube.HandoffPayload

	if delivery.SaveAudio() {
		audioArgs := buildYouTubeAudioArgs(tempDir, raw, selectedNums, isPlaylist)
		if err := e.runYtDlpDownload(raw, audioArgs, trackLabel, "audio"); err != nil {
			counter.Error++
			e.emitTrackFailed(trackLabel, 1, 1, err.Error())
			return nil, err
		}

		downloaded, err := listTempDownloads(tempDir)
		if err != nil || len(downloaded) == 0 {
			counter.Error++
			msg := "no audio files produced by yt-dlp"
			if err != nil {
				msg = err.Error()
			}
			e.emitTrackFailed(trackLabel, 1, 1, msg)
			return nil, fmt.Errorf("%s", msg)
		}

		e.emit(events.Event{
			Type:    events.EventProgress,
			Message: "Converting to AAC (256 kbps) and writing Apple Music tags…",
			Track:   trackLabel,
			Phase:   "postprocess",
			Current: 850,
			Total:   1000,
		})

		if isPlaylist {
			for i, src := range downloaded {
				num := parseDownloadIndex(filepath.Base(src))
				if len(selectedNums) > 0 && i < len(selectedNums) {
					num = selectedNums[i]
				}
				meta := resolveMeta(num)
				if path, rootHint, ok := youtubeExistingLocation(saveDir, meta, multiTrack, false); ok {
					counter.Success++
					msg := fmt.Sprintf("Already on disk: %s", meta.Title)
					if rootHint != "output folder" {
						msg = fmt.Sprintf("Already on disk (%s): %s", rootHint, meta.Title)
					}
					e.emitTrackSkipped(meta.Title, int64(num), int64(len(downloaded)), msg)
					e.log(fmt.Sprintf("Skipped (exists): %s", path))
					noteDownloadOutput(path)
					continue
				}
				e.emit(events.Event{
					Type:    events.EventTrackStart,
					Message: fmt.Sprintf("Processing: %s", meta.Title),
					Track:   meta.Title,
					Current: int64(num),
					Total:   int64(len(downloaded)),
				})
				outPath, err := youtube.FinalizeAudio(Config, saveDir, src, meta, multiTrack)
				if err != nil {
					counter.Error++
					e.emitTrackFailed(meta.Title, int64(num), int64(len(downloaded)), err.Error())
					return nil, err
				}
				e.log(fmt.Sprintf("Saved AAC: %s", outPath))
				noteDownloadOutput(outPath)
			}
		} else {
			meta := resolveMeta(1)
			outPath, err := youtube.FinalizeAudio(Config, saveDir, downloaded[0], meta, multiTrack)
			if err != nil {
				counter.Error++
				e.emitTrackFailed(trackLabel, 1, 1, err.Error())
				return nil, err
			}
			e.log(fmt.Sprintf("Saved AAC: %s", outPath))
			noteDownloadOutput(outPath)
			handoff = handoffFromMeta(outPath, YouTubeDownloadMeta(meta))
		}
	}

	if delivery.SaveVideo() {
		e.emit(events.Event{
			Type:    events.EventProgress,
			Message: "Downloading MP4 video (H.264)…",
			Track:   trackLabel,
			Phase:   "video",
			Current: 0,
			Total:   1000,
		})
		videoTemp, err := os.MkdirTemp(saveDir, "yt-vid-*")
		if err != nil {
			counter.Error++
			e.emitTrackFailed(trackLabel, 1, 1, "Video temp dir: "+err.Error())
			return handoff, err
		}
		defer os.RemoveAll(videoTemp)
		videoArgs := buildYouTubeVideoArgs(videoTemp, raw, selectedNums, isPlaylist)
		if err := e.runYtDlpDownload(raw, videoArgs, trackLabel, "video"); err != nil {
			counter.Error++
			e.emitTrackFailed(trackLabel, 1, 1, "Video copy failed: "+err.Error())
			return handoff, err
		}
		videos, err := listTempDownloads(videoTemp)
		if err != nil || len(videos) == 0 {
			msg := "no video files produced by yt-dlp"
			if err != nil {
				msg = err.Error()
			}
			counter.Error++
			e.emitTrackFailed(trackLabel, 1, 1, msg)
			return handoff, fmt.Errorf("%s", msg)
		}
		for i, src := range videos {
			num := 1
			if isPlaylist {
				num = parseDownloadIndex(filepath.Base(src))
				if len(selectedNums) > 0 && i < len(selectedNums) {
					num = selectedNums[i]
				}
			}
			meta := resolveMeta(num)
			e.emit(events.Event{
				Type:    events.EventProgress,
				Message: "Muxing H.264 + AAC stereo for Apple Music…",
				Track:   meta.Title,
				Phase:   "video",
				Current: 750,
				Total:   1000,
			})
			outPath, err := youtube.FinalizeVideo(Config, src, meta, multiTrack)
			if err != nil {
				e.logError(fmt.Sprintf("Video save failed: %v", err))
				e.emit(events.Event{
					Type:    events.EventLog,
					Message: fmt.Sprintf("Video copy failed for %s: %v", meta.Title, err),
				})
				continue
			}
			e.log(fmt.Sprintf("Saved video: %s", outPath))
			noteDownloadOutput(outPath)
		}
	}

	counter.Success++
	e.emit(events.Event{
		Type:    events.EventTrackComplete,
		Message: delivery.CompleteMessage(),
		Track:   trackLabel,
		Current: 1,
		Total:   1,
	})
	return handoff, nil
}

func mediaWriteTrackTags(path string, tags media.TrackTags) error {
	return media.WriteTrackTags(path, tags)
}

func listTempDownloads(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(ent.Name()))
		switch ext {
		case ".m4a", ".opus", ".webm", ".mp3", ".mp4", ".m4b", ".ogg":
			files = append(files, filepath.Join(dir, ent.Name()))
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no media files in %s", dir)
	}
	sortStrings(files)
	return files, nil
}

func parseDownloadIndex(filename string) int {
	base := filepath.Base(filename)
	idx := strings.Index(base, "_")
	if idx <= 0 {
		return 1
	}
	n, err := strconv.Atoi(base[:idx])
	if err != nil || n <= 0 {
		return 1
	}
	return n
}

func sortStrings(items []string) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func (e *Engine) runYtDlpDownload(raw string, args []string, trackLabel, phase string) error {
	args = append(append([]string(nil), args...), raw)

	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytDlpPath(), args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	consume := func(r io.Reader, captureErr bool) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if captureErr {
				stderrBuf.WriteString(line)
				stderrBuf.WriteByte('\n')
			}
			e.handleYtDlpLine(line, trackLabel, phase)
		}
	}
	wg.Add(2)
	go consume(stdout, false)
	go consume(stderr, true)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderrBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (e *Engine) handleYtDlpLine(line, trackLabel, phase string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if m := reYtDownloadItem.FindStringSubmatch(line); len(m) == 3 {
		item, _ := strconv.Atoi(m[1])
		total, _ := strconv.Atoi(m[2])
		e.emit(events.Event{
			Type:    events.EventTrackStart,
			Message: fmt.Sprintf("Downloading item %d of %d", item, total),
			Track:   trackLabel,
			Current: int64(item),
			Total:   int64(total),
		})
	}
	if m := reYtDownloadPct.FindStringSubmatch(line); len(m) == 2 {
		pct, _ := strconv.ParseFloat(m[1], 64)
		e.emit(events.Event{
			Type:    events.EventProgress,
			Message: line,
			Track:   trackLabel,
			Phase:   phase,
			Current: int64(pct * 10),
			Total:   1000,
		})
	}
	if reYtPostProcess.MatchString(line) {
		label := "Processing audio…"
		if phase == "video" {
			label = "Merging video for Apple Music…"
		}
		e.emit(events.Event{
			Type:    events.EventProgress,
			Message: label,
			Track:   trackLabel,
			Phase:   "postprocess",
			Current: 950,
			Total:   1000,
		})
	}
	if strings.HasPrefix(line, "[download]") || strings.HasPrefix(line, "[info]") {
		e.log(line)
	}
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

func (e *Engine) emitTrackSkipped(label string, current, total int64, msg string) {
	e.emit(events.Event{
		Type:    events.EventTrackComplete,
		Message: msg,
		Track:   label,
		Current: current,
		Total:   total,
		Phase:   "skipped",
	})
}

func containsInt(nums []int, want int) bool {
	for _, n := range nums {
		if n == want {
			return true
		}
	}
	return false
}

func (e *Engine) runYtDlp(args ...string) ([]byte, error) {
	return e.runYtDlpWithTimeout(120*time.Second, args...)
}

func (e *Engine) runYtDlpWithTimeout(timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
