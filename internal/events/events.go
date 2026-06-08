package events

import "sync"

type EventType string

const (
	EventLog           EventType = "log"
	EventProgress      EventType = "progress"
	EventJobStart      EventType = "job_start"
	EventTrackStart    EventType = "track_start"
	EventTrackComplete EventType = "track_complete"
	EventTrackFailed   EventType = "track_failed"
	EventJobComplete   EventType = "job_complete"
	EventError         EventType = "error"
	EventSpliceProgress EventType = "splice_progress"
	EventSpliceComplete EventType = "splice_complete"
	EventSpliceError    EventType = "splice_error"
)

// SpliceHandoff prefills the Split mix tab after a download.
type SpliceHandoff struct {
	MasterPath  string `json:"master_path"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Artist      string `json:"artist"`
	Year        string `json:"year"`
	Genre       string `json:"genre"`
	ArtURL      string `json:"art_url,omitempty"`
}

type Event struct {
	Type    EventType `json:"type"`
	Message string    `json:"message,omitempty"`
	Phase   string    `json:"phase,omitempty"`
	Current int64     `json:"current,omitempty"`
	Total   int64     `json:"total,omitempty"`
	Track   string    `json:"track,omitempty"`
	Success int       `json:"success,omitempty"`
	Error   int       `json:"error,omitempty"`
	Total_  int       `json:"total_count,omitempty"`
	MasterPath string        `json:"master_path,omitempty"`
	Handoff    *SpliceHandoff `json:"handoff,omitempty"`
}

type Emitter interface {
	Emit(ev Event)
}

type FuncEmitter func(ev Event)

func (f FuncEmitter) Emit(ev Event) { f(ev) }

type MultiEmitter struct {
	mu       sync.RWMutex
	emitters []Emitter
}

func (m *MultiEmitter) Add(e Emitter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emitters = append(m.emitters, e)
}

func (m *MultiEmitter) Emit(ev Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.emitters {
		e.Emit(ev)
	}
}

type CLIEmitter struct{}

func (CLIEmitter) Emit(ev Event) {
	switch ev.Type {
	case EventLog, EventTrackStart, EventTrackComplete, EventTrackFailed, EventError:
		if ev.Message != "" {
			println(ev.Message)
		}
	case EventProgress:
		// CLI uses progressbar in runv2
	case EventJobComplete:
		if ev.Message != "" {
			println(ev.Message)
		}
	}
}
