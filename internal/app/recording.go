package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"amdl/internal/classical"
	"amdl/internal/download"
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

// Seams for tests; production uses the real Classical client and Catalog.
var (
	fetchRecording = func(req classical.Request) (*classical.Recording, error) {
		return classical.NewClient(download.Client).Fetch(context.Background(), req)
	}
	loadRecordingAlbum = func(storefront, albumID, token, language string) (*model.Album, error) {
		album := model.NewAlbum(storefront, albumID)
		if err := album.GetResp(token, language); err != nil {
			return nil, err
		}
		return album, nil
	}
)

// isClassicalWorkURL reports links to Classical Work pages, which are not
// supported yet.
func isClassicalWorkURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Hostname(), classical.Host) {
		return false
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	return len(segments) >= 2 && segments[1] == "work"
}

// handleClassicalURL downloads Classical Recording links and rejects Work
// links. It returns false for every other URL.
func (r *Runner) handleClassicalURL(raw, token string) bool {
	if isClassicalWorkURL(raw) {
		fmt.Println("Classical Work links are not supported yet; use a Recording link.")
		r.State.Counter.Error++
		return true
	}
	req, ok, err := classical.ParseRecordingURL(raw, r.Config.Language)
	if !ok {
		return false
	}
	fmt.Println("Classical Recording")
	if err != nil {
		fmt.Println("Invalid classical recording URL:", err)
		r.State.Counter.Error++
		return true
	}
	rec, err := fetchRecording(req)
	if err != nil {
		fmt.Println("Failed to get classical recording:", err)
		r.State.Counter.Error++
		return true
	}
	if err := r.ripRecording(rec, token, r.Config.MediaUserToken); err != nil {
		fmt.Println("Failed to rip classical recording:", err)
		r.State.Counter.Error++
	}
	return true
}

func (r *Runner) ripRecording(rec *classical.Recording, token, mediaUserToken string) error {
	// The Recording's language applies to this Recording only.
	savedLanguage := r.Config.Language
	r.Config.Language = rec.Request.Language
	defer func() { r.Config.Language = savedLanguage }()

	album, err := loadRecordingAlbum(rec.Request.Storefront, rec.AlbumID, token, rec.Request.Language)
	if err != nil {
		return fmt.Errorf("load album %s: %w", rec.AlbumID, err)
	}
	tracks, err := buildRecordingTracks(rec, album)
	if err != nil {
		return err
	}
	fmt.Printf("%s - %s (%d tracks, source %s)\n", rec.Composer, rec.WorkTitle, len(tracks), rec.Source)
	for _, warning := range rec.Warnings {
		fmt.Println("Warning:", warning)
	}

	codec, root := "ALAC", r.Config.AlacSaveFolder
	if r.Flags.Atmos {
		codec, root = "ATMOS", r.Config.AtmosSaveFolder
	} else if r.Flags.AAC {
		codec, root = "AAC", r.Config.AacSaveFolder
	}
	dir, err := r.classicalFolder(root, &tracks[0])
	if err != nil {
		return err
	}
	if err := createDirectory(dir); err != nil {
		return err
	}
	covPath, err := r.writeCover(dir, "cover", album.Resp.Data[0].Attributes.Artwork.URL)
	if err != nil {
		fmt.Println("Failed to write cover.")
	}
	for i := range tracks {
		tracks[i].SaveDir = dir
		tracks[i].CoverPath = covPath
		tracks[i].Codec = codec
		r.ripTrack(&tracks[i], token, mediaUserToken)
	}
	return nil
}
