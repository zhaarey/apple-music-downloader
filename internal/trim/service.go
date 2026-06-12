package trim

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"main/internal/events"
	"main/internal/media"
	"main/utils/structs"
)

const MinSelectionMs = 1000

// ProbeResult describes a local media file for the trim UI.
type ProbeResult struct {
	DurationMs int    `json:"duration_ms"`
	HasVideo   bool   `json:"has_video"`
	HasAudio   bool   `json:"has_audio"`
	Summary    string `json:"summary"`
	MediaKind  string `json:"media_kind"` // audio | video
}

// ExportInput is the trim export job payload.
type ExportInput struct {
	SourcePath string `json:"source_path"`
	OutputPath string `json:"output_path"`
	StartMs    int    `json:"start_ms"`
	EndMs      int    `json:"end_ms"`
	CopyTags   bool   `json:"copy_tags"`
	Overwrite  bool   `json:"overwrite"`
}

// Service runs async trim exports.
type Service struct {
	mu     sync.Mutex
	cfg    structs.ConfigSet
	emit   func(events.Event)
	ctx    context.Context
	cancel context.CancelFunc
	active bool
}

func NewService(cfg structs.ConfigSet, emit func(events.Event)) *Service {
	return &Service{cfg: cfg, emit: emit}
}

func (s *Service) SetConfig(cfg structs.ConfigSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func (s *Service) cfgSnapshot() structs.ConfigSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *Service) emitEvent(ev events.Event) {
	if s.emit != nil {
		s.emit(ev)
	}
}

func (s *Service) CancelExport() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Service) IsExporting() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Probe reads duration and stream info for a trim source file.
func Probe(ffmpegConfigured, path string) (ProbeResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return ProbeResult{}, fmt.Errorf("no file path")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ProbeResult{}, fmt.Errorf("file not found")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".m4a", ".mp4":
	default:
		return ProbeResult{}, fmt.Errorf("unsupported file type — use .m4a or .mp4")
	}

	out := ProbeResult{MediaKind: "audio"}
	if ext == ".mp4" {
		out.MediaKind = "video"
		out.HasVideo = true
		vinfo, err := media.ProbeVideoFile(ffmpegConfigured, path)
		if err != nil {
			return ProbeResult{}, err
		}
		out.DurationMs = vinfo.DurationMs
		out.HasAudio = vinfo.AudioChannels > 0
		out.Summary = fmt.Sprintf("Video · %s · %s", vinfo.DurationLabel, vinfo.VideoCodec)
		return out, nil
	}

	ainfo, err := media.ProbeSource(ffmpegConfigured, path)
	if err != nil {
		return ProbeResult{}, err
	}
	out.DurationMs = ainfo.DurationMs
	out.HasAudio = true
	out.Summary = ainfo.Summary
	return out, nil
}

// DefaultOutputPath suggests a trimmed filename beside the source.
func DefaultOutputPath(sourcePath string) string {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return ""
	}
	dir := filepath.Dir(sourcePath)
	base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	ext := filepath.Ext(sourcePath)
	return filepath.Join(dir, base+" [trimmed]"+ext)
}

// StartExport runs trim export asynchronously.
func (s *Service) StartExport(input ExportInput) error {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return fmt.Errorf("a trim export is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	s.active = true
	s.mu.Unlock()

	go func() {
		err := s.runExport(ctx, input)
		s.mu.Lock()
		s.active = false
		s.cancel = nil
		s.mu.Unlock()
		if err != nil && err != context.Canceled {
			s.emitEvent(events.Event{Type: events.EventTrimError, Message: err.Error()})
		}
	}()
	return nil
}

func (s *Service) runExport(ctx context.Context, input ExportInput) error {
	select {
	case <-ctx.Done():
		s.emitEvent(events.Event{Type: events.EventTrimComplete, Message: "Trim cancelled.", Phase: "cancelled"})
		return context.Canceled
	default:
	}

	src := strings.TrimSpace(input.SourcePath)
	if src == "" || !fileExists(src) {
		return fmt.Errorf("source file not found")
	}
	startMs := input.StartMs
	endMs := input.EndMs
	if endMs <= startMs {
		return fmt.Errorf("end must be after start")
	}
	if endMs-startMs < MinSelectionMs {
		return fmt.Errorf("selection must be at least 1 second")
	}

	probe, err := Probe(s.cfgSnapshot().FFmpegPath, src)
	if err != nil {
		return err
	}
	if endMs > probe.DurationMs {
		endMs = probe.DurationMs
	}
	if startMs < 0 {
		startMs = 0
	}

	outPath := strings.TrimSpace(input.OutputPath)
	if input.Overwrite {
		outPath = src
	}
	if outPath == "" {
		outPath = DefaultOutputPath(src)
	}

	cfg := s.cfgSnapshot()
	s.emitEvent(events.Event{
		Type:    events.EventTrimProgress,
		Message: "Trimming…",
		Current: 0,
		Total:   3,
	})

	tempPath := outPath
	if input.Overwrite {
		tempPath = src + "._trim.tmp" + filepath.Ext(src)
		defer os.Remove(tempPath)
	}

	isVideo := probe.MediaKind == "video" || strings.EqualFold(filepath.Ext(src), ".mp4")
	if isVideo && probe.HasVideo {
		if err := media.ExportVideoSlice(cfg.FFmpegPath, src, tempPath, startMs, endMs); err != nil {
			return err
		}
	} else {
		ainfo, err := media.ProbeSource(cfg.FFmpegPath, src)
		if err != nil {
			return err
		}
		if _, err := media.ExportSlice(cfg.FFmpegPath, src, tempPath, startMs, endMs, ainfo.SampleRate, ainfo.Channels); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		_ = os.Remove(tempPath)
		s.emitEvent(events.Event{Type: events.EventTrimComplete, Message: "Trim cancelled.", Phase: "cancelled"})
		return context.Canceled
	default:
	}

	s.emitEvent(events.Event{
		Type:    events.EventTrimProgress,
		Message: "Applying metadata…",
		Current: 2,
		Total:   3,
	})

	if input.CopyTags {
		if err := copyTagsFromSource(cfg.FFmpegPath, src, tempPath); err != nil {
			return fmt.Errorf("tag copy failed: %w", err)
		}
	}

	if input.Overwrite {
		backup := src + ".bak"
		if !fileExists(backup) {
			if err := os.Rename(src, backup); err != nil {
				return fmt.Errorf("backup failed: %w", err)
			}
		} else {
			_ = os.Remove(src)
		}
		if err := os.Rename(tempPath, src); err != nil {
			_ = os.Rename(backup, src)
			return fmt.Errorf("replace failed: %w", err)
		}
		outPath = src
	}

	s.emitEvent(events.Event{
		Type:       events.EventTrimComplete,
		Message:    fmt.Sprintf("Saved trimmed file to %s", outPath),
		Phase:      "success",
		OutputPath: outPath,
		Current:    3,
		Total:      3,
	})
	return nil
}

func copyTagsFromSource(ffmpegConfigured, src, dst string) error {
	info, err := media.ReadAudioTags(src)
	if err != nil {
		return err
	}
	input := media.WriteAudioTagsInput{
		Path:        dst,
		OutputPath:  dst,
		Title:       info.Title,
		Artist:      info.Artist,
		Album:       info.Album,
		AlbumArtist: info.AlbumArtist,
		Genre:       info.Genre,
		Year:        info.Year,
		TrackNumber: info.TrackNumber,
		TrackTotal:  info.TrackTotal,
		DiscNumber:  info.DiscNumber,
		DiscTotal:   info.DiscTotal,
	}
	return media.WriteAudioTags(input)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
