package splice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	appconfig "main/internal/config"
	"main/internal/events"
	"main/internal/media"
	"main/utils/structs"
)

// Exporter slices a master file into tagged AAC tracks.
type Exporter struct {
	mu      sync.Mutex
	cfg     structs.ConfigSet
	emit    func(events.Event)
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

func NewExporter(cfg structs.ConfigSet, emit func(events.Event)) *Exporter {
	return &Exporter{cfg: cfg, emit: emit}
}

func (x *Exporter) SetConfig(cfg structs.ConfigSet) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.cfg = cfg
}

func (x *Exporter) emitEvent(ev events.Event) {
	if x.emit != nil {
		x.emit(ev)
	}
}

// Cancel stops an in-progress export.
func (x *Exporter) Cancel() {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.cancel != nil {
		x.cancel()
	}
}

// IsRunning reports whether export is active.
func (x *Exporter) IsRunning() bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.running
}

func (x *Exporter) cfgSnapshot() structs.ConfigSet {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.cfg
}

// Export runs the splice export job.
func (x *Exporter) Export(project Project) error {
	x.mu.Lock()
	if x.running {
		x.mu.Unlock()
		return fmt.Errorf("a splice export is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	x.ctx = ctx
	x.cancel = cancel
	x.running = true
	x.mu.Unlock()

	defer func() {
		x.mu.Lock()
		x.running = false
		x.cancel = nil
		x.mu.Unlock()
	}()

	if project.MasterPath == "" || !fileExists(project.MasterPath) {
		x.emitEvent(events.Event{Type: events.EventSpliceError, Message: "Source audio file not found."})
		return fmt.Errorf("source not found")
	}
	if len(project.Tracks) == 0 {
		x.emitEvent(events.Event{Type: events.EventSpliceError, Message: "No tracks to export."})
		return fmt.Errorf("no tracks")
	}
	for i, t := range project.Tracks {
		if t.TagTitle() == "" {
			msg := fmt.Sprintf("Track %d has an empty title.", i+1)
			x.emitEvent(events.Event{Type: events.EventSpliceError, Message: msg})
			return fmt.Errorf("%s", msg)
		}
	}

	info, err := media.ProbeSource(x.cfgSnapshot().FFmpegPath, project.MasterPath)
	if err != nil {
		x.emitEvent(events.Event{Type: events.EventSpliceError, Message: err.Error()})
		return err
	}
	project.MasterDurationMs = info.DurationMs
	DistributeDrift(project.Tracks, project.MasterDurationMs)

	outputDir := project.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(appconfig.AppDataDir(), "splice-output")
	}
	albumFolder := media.SanitizePathPart(project.Album.Album)
	if albumFolder == "" {
		albumFolder = "Spliced Album"
	}
	outRoot := filepath.Join(outputDir, albumFolder)
	if err := os.MkdirAll(outRoot, 0755); err != nil {
		return err
	}

	timings := project.ComputeStartEnd()
	total := len(project.Tracks)
	totalTracks := project.Album.EffectiveTotalTracks(total)
	encoderLabel := media.DetectAACEncoder(x.cfgSnapshot().FFmpegPath).Label

	x.emitEvent(events.Event{
		Type:    events.EventSpliceProgress,
		Message: "Starting export…",
		Current: 0,
		Total:   int64(total + 1),
		Phase:   encoderLabel,
	})

	var coverPath string
	if project.Album.ArtworkPath != nil {
		coverPath = *project.Album.ArtworkPath
	}

	for idx, track := range project.Tracks {
		select {
		case <-x.ctx.Done():
			x.emitEvent(events.Event{Type: events.EventSpliceComplete, Message: "Export cancelled.", Phase: "cancelled"})
			return context.Canceled
		default:
		}

		start, end, _ := timings[idx][0], timings[idx][1], timings[idx][2]
		trackNum := track.EffectiveTrackNumber(idx)
		filename := media.TrackFilename(trackNum, track.TagTitle(), ".m4a")
		dst := filepath.Join(outRoot, filename)

		x.emitEvent(events.Event{
			Type:    events.EventSpliceProgress,
			Message: fmt.Sprintf("Exporting: %s", track.TagTitle()),
			Track:   track.TagTitle(),
			Current: int64(idx + 1),
			Total:   int64(total),
		})

		if _, err := media.ExportSlice(x.cfgSnapshot().FFmpegPath, project.MasterPath, dst, start, end, info.SampleRate, info.Channels); err != nil {
			x.emitEvent(events.Event{Type: events.EventSpliceError, Message: err.Error()})
			return err
		}

		artist := track.EffectiveArtist(project.Album.Artist)
		if artist == "" {
			artist = project.Album.AlbumArtist
		}
		album := track.EffectiveAlbum(project.Album.Album)
		albumArtist := track.EffectiveAlbumArtist(project.Album.AlbumArtist)
		genre := track.EffectiveGenre(project.Album.Genre)
		year := track.EffectiveYear(project.Album.Year)
		discNumber := track.EffectiveDiscNumber(1)
		discTotal := track.EffectiveDiscTotal(1)
		tags := media.TrackTags{
			Title:       track.TagTitle(),
			Artist:      artist,
			Album:       album,
			AlbumArtist: albumArtist,
			Genre:       genre,
			Year:        year,
			TrackNumber: int16(trackNum),
			TrackTotal:  int16(totalTracks),
			DiscNumber:  int16(discNumber),
			DiscTotal:   int16(discTotal),
			SortTags:    x.cfgSnapshot().TagSortOrder,
			CoverPath:   coverPath,
		}
		if err := media.WriteTrackTags(dst, tags); err != nil {
			x.emitEvent(events.Event{Type: events.EventSpliceError, Message: "Tagging failed: " + err.Error()})
			return err
		}
	}

	x.emitEvent(events.Event{
		Type:       events.EventSpliceComplete,
		Message:    fmt.Sprintf("Exported %d tracks to %s", total, outRoot),
		Phase:      "success",
		MasterPath: outRoot,
		Success:    total,
		Total_:     total,
	})
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
