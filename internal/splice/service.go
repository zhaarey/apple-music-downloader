package splice

import (
	"sync"

	"main/internal/events"
	"main/utils/structs"
)

// Service is the splice suite backend (project math + export).
type Service struct {
	mu       sync.Mutex
	cfg      structs.ConfigSet
	emit     func(events.Event)
	exporter *Exporter
}

func NewService(cfg structs.ConfigSet, emit func(events.Event)) *Service {
	s := &Service{cfg: cfg, emit: emit}
	s.exporter = NewExporter(cfg, emit)
	return s
}

func (s *Service) SetConfig(cfg structs.ConfigSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	s.exporter.SetConfig(cfg)
}

func (s *Service) cfgSnapshot() structs.ConfigSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// ComputeTimings returns resolved boundaries for a project copy.
func (s *Service) ComputeTimings(project Project) [][3]int {
	if project.MasterDurationMs <= 0 && project.MasterPath != "" {
		if probe, err := ProbeMaster(s.cfgSnapshot().FFmpegPath, project.MasterPath); err == nil {
			project.MasterDurationMs = probe.DurationMs
		}
	}
	return ComputeTrackTimings(project.Tracks, project.MasterDurationMs)
}

// SetBoundary adjusts a cut boundary and returns updated tracks.
func (s *Service) SetBoundary(project Project, boundaryIndex, positionMs int) Project {
	SetBoundary(project.Tracks, boundaryIndex, positionMs, project.MasterDurationMs, nil)
	return project
}

// SetTrackStartProject updates a track start and reflows surrounding tracks.
func (s *Service) SetTrackStartProject(project Project, row, startMs int) Project {
	SetTrackStart(project.Tracks, row, startMs, project.MasterDurationMs, nil)
	return project
}

// SetTrackDurationProject updates a track duration and reflows downstream tracks.
func (s *Service) SetTrackDurationProject(project Project, row, durationMs int) Project {
	SetTrackDuration(project.Tracks, row, durationMs, project.MasterDurationMs)
	return project
}

// DistributeDriftProject spreads drift and returns updated project.
func (s *Service) DistributeDriftProject(project Project) Project {
	DistributeDrift(project.Tracks, project.MasterDurationMs)
	return project
}

// StartExport begins export in the background.
func (s *Service) StartExport(project Project) error {
	go func() {
		_ = s.exporter.Export(project)
	}()
	return nil
}

// CancelExport stops export.
func (s *Service) CancelExport() {
	s.exporter.Cancel()
}

// IsExporting reports export state.
func (s *Service) IsExporting() bool {
	return s.exporter.IsRunning()
}
