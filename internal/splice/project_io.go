package splice

import (
	"encoding/json"
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/*.json
var templateFS embed.FS

// LoadProject reads a splice project JSON file.
func LoadProject(path string) (Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Project{}, err
	}
	return ParseProjectJSON(data)
}

// ParseProjectJSON unmarshals project JSON (audio-splicer compatible).
func ParseProjectJSON(data []byte) (Project, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Project{}, err
	}

	albumRaw, _ := raw["album"].(map[string]interface{})
	album := AlbumMetadata{
		Album:       strVal(albumRaw, "album"),
		AlbumArtist: strVal(albumRaw, "album_artist"),
		Artist:      strVal(albumRaw, "artist"),
		Year:        strVal(albumRaw, "year"),
		Genre:       strVal(albumRaw, "genre"),
	}
	if v, ok := albumRaw["artwork_path"].(string); ok && v != "" {
		album.ArtworkPath = &v
	}
	if v, ok := intVal(albumRaw, "total_tracks"); ok {
		album.TotalTracks = &v
	}

	tracksRaw, _ := raw["tracks"].([]interface{})
	tracks := make([]Track, 0, len(tracksRaw))
	for _, item := range tracksRaw {
		m, _ := item.(map[string]interface{})
		durationMs := 0
		if v, ok := intVal(m, "duration_ms"); ok {
			durationMs = v
		} else if s, ok := m["duration"].(string); ok {
			durationMs = ParseDuration(s)
		}
		var startMs *int
		if v, ok := intVal(m, "start_ms"); ok {
			startMs = &v
		}
		var trackNum *int
		if v, ok := intVal(m, "track_number"); ok {
			trackNum = &v
		}
		var discNum *int
		if v, ok := intVal(m, "disc_number"); ok {
			discNum = &v
		}
		var discTotal *int
		if v, ok := intVal(m, "disc_total"); ok {
			discTotal = &v
		}
		tracks = append(tracks, Track{
			Title:       strVal(m, "title"),
			DurationMs:  durationMs,
			StartMs:     startMs,
			Artist:      strVal(m, "artist"),
			TrackNumber: trackNum,
			Duration:    strVal(m, "duration"),
			Album:       strVal(m, "album"),
			AlbumArtist: strVal(m, "album_artist"),
			Genre:       strVal(m, "genre"),
			Year:        strVal(m, "year"),
			DiscNumber:  discNum,
			DiscTotal:   discTotal,
		})
	}

	return Project{
		MasterPath: strVal(raw, "master_path"),
		OutputDir:  defaultStr(strVal(raw, "output_dir"), "./output"),
		Album:      album,
		Tracks:     tracks,
	}, nil
}

// SaveProject writes project JSON.
func SaveProject(project Project, path string) error {
	data := projectToJSON(project)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func projectToJSON(project Project) []byte {
	album := map[string]interface{}{
		"album":        project.Album.Album,
		"album_artist": project.Album.AlbumArtist,
		"artist":       project.Album.Artist,
		"year":         project.Album.Year,
		"genre":        project.Album.Genre,
		"artwork_path": project.Album.ArtworkPath,
		"total_tracks": project.Album.TotalTracks,
	}
	trackItems := make([]map[string]interface{}, 0, len(project.Tracks))
	for _, track := range project.Tracks {
		item := map[string]interface{}{
			"title":  track.Title,
			"artist": track.Artist,
		}
		if track.Album != "" {
			item["album"] = track.Album
		}
		if track.AlbumArtist != "" {
			item["album_artist"] = track.AlbumArtist
		}
		if track.Genre != "" {
			item["genre"] = track.Genre
		}
		if track.Year != "" {
			item["year"] = track.Year
		}
		if track.DurationMs > 0 {
			item["duration_ms"] = track.DurationMs
			item["duration"] = FormatDuration(track.DurationMs, track.DurationMs >= 3600000)
		} else if track.Duration != "" {
			item["duration"] = track.Duration
		}
		if track.StartMs != nil {
			item["start_ms"] = *track.StartMs
		}
		if track.TrackNumber != nil {
			item["track_number"] = *track.TrackNumber
		}
		if track.DiscNumber != nil {
			item["disc_number"] = *track.DiscNumber
		}
		if track.DiscTotal != nil {
			item["disc_total"] = *track.DiscTotal
		}
		trackItems = append(trackItems, item)
	}
	payload := map[string]interface{}{
		"master_path": project.MasterPath,
		"output_dir":  project.OutputDir,
		"album":       album,
		"tracks":      trackItems,
	}
	out, _ := json.MarshalIndent(payload, "", "  ")
	return out
}

// ListTemplates returns bundled template names.
func ListTemplates() ([]string, error) {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) == ".json" {
			names = append(names, name[:len(name)-5])
		}
	}
	return names, nil
}

// LoadTemplate loads a bundled template by name (without .json).
func LoadTemplate(name string) (Project, error) {
	data, err := templateFS.ReadFile("templates/" + name + ".json")
	if err != nil {
		return Project{}, fmt.Errorf("template %q not found", name)
	}
	return ParseProjectJSON(data)
}

func strVal(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func intVal(m map[string]interface{}, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	switch v := m[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
