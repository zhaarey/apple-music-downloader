package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	ampapi "amdl/internal/amp-api"
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
	var conductors []string
	for _, movement := range rec.Tracks {
		conductors = append(conductors, movement.Conductors...)
	}
	recordingConductor := joinUnique(conductors)
	for i, movement := range rec.Tracks {
		track := *catalog[movement.SongID]
		track.TaskNum = i + 1
		track.TaskTotal = len(rec.Tracks)
		track.PreType = "albums"
		track.PreID = rec.AlbumID
		track.Classical = &model.ClassicalContext{
			RecordingID:        rec.Request.RecordingID,
			WorkTitle:          rec.WorkTitle,
			Composer:           rec.Composer,
			MovementTitle:      movement.Title,
			Position:           movement.Position,
			Count:              movement.Count,
			Conductor:          joinUnique(movement.Conductors),
			RecordingConductor: recordingConductor,
		}
		tracks[i] = track
	}
	return tracks, nil
}

// catalogLanguage returns the language the Catalog used for album, which its
// href records, or "" when that is unknown.
func catalogLanguage(album *model.Album) string {
	if len(album.Resp.Data) == 0 {
		return ""
	}
	u, err := url.Parse(album.Resp.Data[0].Href)
	if err != nil {
		return ""
	}
	return u.Query().Get("l")
}

// languageKey returns the lower-case language subtag of a tag, such as "de"
// for "de-DE". Chinese keeps its script, since Simplified and Traditional
// names differ: "zh-hans" for zh-Hans-CN or zh-CN, "zh-hant" for zh-Hant-HK
// or zh-TW.
func languageKey(tag string) string {
	subtags := strings.Split(strings.ToLower(tag), "-")
	if subtags[0] != "zh" {
		return subtags[0]
	}
	for _, subtag := range subtags[1:] {
		switch subtag {
		case "hans", "cn", "sg":
			return "zh-hans"
		case "hant", "tw", "hk", "mo":
			return "zh-hant"
		}
	}
	return "zh"
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
	showTrackQuality   = (*Runner).printTrackQuality
	storefrontLanguage = ampapi.GetDefaultLanguage
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
		// Not counted as an error, like an unknown link type: retrying the
		// queue cannot make it succeed.
		fmt.Println("Classical Work links are not supported yet; use a Recording link.")
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
	if req.Language == "" {
		// Without l the Classical API answers in English but the Catalog in
		// the storefront's language, so ask both for the latter.
		language, err := storefrontLanguage(req.Storefront, token)
		if err != nil {
			fmt.Println("Warning: storefront language unknown, classical titles may be in English:", err)
		}
		req.Language = language
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
	if served := catalogLanguage(album); rec.Request.Language != "" && served != "" && languageKey(served) != languageKey(rec.Request.Language) {
		rec.Warnings = append(rec.Warnings, fmt.Sprintf("the Catalog answered in %s, not %s: album and artist names will not match the classical titles; use a language the storefront offers", served, rec.Request.Language))
	}
	tracks, err := buildRecordingTracks(rec, album)
	if err != nil {
		return err
	}
	fmt.Printf("%s - %s (%d tracks, source %s)\n", rec.Composer, rec.WorkTitle, len(tracks), rec.Source)
	for _, warning := range rec.Warnings {
		fmt.Println("Warning:", warning)
	}
	if r.Flags.Select {
		fmt.Println("--select does not apply to classical recordings; all their tracks are used.")
	}
	if r.Flags.Debug {
		// Debug mode only inspects: nothing is written or counted.
		for i := range tracks {
			fmt.Printf("\nTrack %d of %d:\n", i+1, len(tracks))
			fmt.Printf("%02d. %s\n", i+1, tracks[i].Classical.MovementTitle)
			showTrackQuality(r, i+1, rec.Request.Storefront, tracks[i].ID, tracks[i].Resp.Attributes.AudioTraits, rec.Request.Language, token)
		}
		return nil
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
	// Check every file name before anything is written.
	for i := range tracks {
		if _, err := r.classicalFileName(&tracks[i]); err != nil {
			return err
		}
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
