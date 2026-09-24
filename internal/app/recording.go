package app

import (
	"errors"
	"fmt"
	"strings"

	"amdl/internal/classical"
	"amdl/internal/model"
)

// buildRecordingTracks picks the Recording's songs out of its fully loaded
// parent Album, in Recording order. Catalog numbering is kept for the tags.
func buildRecordingTracks(rec *classical.Recording, album *model.Album) ([]model.Track, error) {
	if album.ID != rec.AlbumID {
		return nil, fmt.Errorf("recording album %s does not match loaded album %s", rec.AlbumID, album.ID)
	}
	catalog := make(map[string]*model.Track, len(album.Tracks))
	for i := range album.Tracks {
		catalog[album.Tracks[i].ID] = &album.Tracks[i]
	}
	var missing []string
	for _, movement := range rec.Tracks {
		if track, ok := catalog[movement.SongID]; !ok || track.Type != "songs" {
			missing = append(missing, movement.SongID)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("recording songs missing from album %s: %s", rec.AlbumID, strings.Join(missing, ", "))
	}
	if len(rec.Tracks) == 0 {
		return nil, errors.New("recording has no songs")
	}

	tracks := make([]model.Track, len(rec.Tracks))
	for i, movement := range rec.Tracks {
		track := *catalog[movement.SongID]
		track.TaskNum = i + 1
		track.TaskTotal = len(rec.Tracks)
		track.PreType = "albums"
		track.PreID = rec.AlbumID
		track.Classical = &model.ClassicalContext{
			RecordingID:   rec.Request.RecordingID,
			WorkTitle:     rec.WorkTitle,
			Composer:      rec.Composer,
			MovementTitle: movement.Title,
			Position:      movement.Position,
			Count:         movement.Count,
			Conductor:     joinUnique(movement.Conductors),
		}
		tracks[i] = track
	}
	return tracks, nil
}

// joinUnique joins non-empty names in first-seen order without duplicates.
func joinUnique(names []string) string {
	seen := make(map[string]bool, len(names))
	var unique []string
	for _, name := range names {
		if name != "" && !seen[name] {
			seen[name] = true
			unique = append(unique, name)
		}
	}
	return strings.Join(unique, "; ")
}
