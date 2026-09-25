package classical

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// conductorRoles lists the role labels the Classical API serves for a
// conductor. Roles are localized display strings with no language-independent
// id, so labels in other languages are left unclassified.
var conductorRoles = map[string]bool{
	"Conductor":             true, // en
	"指揮者":                   true, // ja
	"指挥":                    true, // zh-Hans
	"指揮":                    true, // zh-Hant
	"Dirigent:in":           true, // de
	"Direction d’orchestre": true, // fr
	"지휘자":                   true, // ko
	"Dirección":             true, // es
	"Direzione":             true, // it
	"Regência":              true, // pt-BR
}

// conductorLanguages holds the primary language subtags of conductorRoles.
var conductorLanguages = map[string]bool{
	"en": true, "ja": true, "zh": true, "de": true, "fr": true,
	"ko": true, "es": true, "it": true, "pt": true,
}

type trackMetadata struct {
	TrackMetadataMap map[string]struct {
		ID      string `json:"id"`
		Artists []struct {
			Title string `json:"title"`
			Role  string `json:"role"`
		} `json:"artists"`
	} `json:"trackMetadataMap"`
}

// addMetadata fills optional per-track details. It never changes the song
// list or titles, and callers treat its error as a warning.
func (c *Client) addMetadata(ctx context.Context, rec *Recording) error {
	req := rec.Request
	body, err := c.get(ctx, apiPrefix+"/query/view/"+req.Storefront+"/recording/"+url.PathEscape(req.RecordingID)+"/tracksMetadata", req.Language, "application/json")
	if err != nil {
		return err
	}
	var meta trackMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return fmt.Errorf("decode track metadata: %w", err)
	}
	var missing, conductors int
	for i := range rec.Tracks {
		track := &rec.Tracks[i]
		entry, ok := meta.TrackMetadataMap[track.SongID]
		if !ok || entry.ID != track.SongID {
			missing++
			continue
		}
		for _, artist := range entry.Artists {
			if conductorRoles[artist.Role] && artist.Title != "" {
				track.Conductors = append(track.Conductors, artist.Title)
				conductors++
			}
		}
	}
	language, _, _ := strings.Cut(strings.ToLower(req.Language), "-")
	if conductors == 0 && language != "" && !conductorLanguages[language] {
		rec.Warnings = append(rec.Warnings, fmt.Sprintf("conductor role names in language %q are unknown; a conductor would not be tagged", req.Language))
	}
	if missing > 0 {
		return fmt.Errorf("%d of %d tracks have no matching metadata", missing, len(rec.Tracks))
	}
	return nil
}
