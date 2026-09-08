package app

import (
	"amdl/internal/config"
	"os"
	"regexp"
)

var (
	forbiddenNames = regexp.MustCompile(`[/\\<>:"|?*]`)
	alac_max       *int
	atmos_max      *int
	mv_max         *int
	mv_audio_type  *string
	aac_type       *string
)

type Flags struct {
	Atmos          bool
	AAC            bool
	Select         bool
	ArtistSelect   bool
	Debug          bool
	PrintJSON      bool
	SaveM3U8       bool
	LiteServerFlag string
}

type State struct {
	Counter     config.Counter
	OKDict      map[string][]int
	AddedTracks []AddedTrack
}

type Runner struct {
	Config config.ConfigSet
	Flags  Flags
	State  State
}

func NewRunner(cfg config.ConfigSet) *Runner {
	return &Runner{Config: cfg, State: State{OKDict: make(map[string][]int)}}
}

type AddedTrack struct {
	Path     string `json:"path"`
	Artist   string `json:"artist"`
	ArtistID string `json:"artist_id"`
	Album    string `json:"album"`
	Song     string `json:"song"`
}

// topLevelKeys returns the set of top-level YAML keys in data.

// contains reports whether item is present in slice.
func contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func (r *Runner) LimitString(s string) string {
	if len([]rune(s)) > r.Config.LimitMax {
		return string([]rune(s)[:r.Config.LimitMax])
	}
	return s
}

func fileExists(path string) (bool, error) {
	f, err := os.Stat(path)
	if err == nil {
		return !f.IsDir(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
