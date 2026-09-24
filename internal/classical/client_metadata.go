package classical

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// conductorRoles lists the role labels observed for conductors. Roles are
// localized display strings, so unknown labels are left unclassified.
var conductorRoles = map[string]bool{
	"Conductor": true,
	"指挥":        true,
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
	var missing int
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
			}
		}
	}
	if missing > 0 {
		return fmt.Errorf("%d of %d tracks have no matching metadata", missing, len(rec.Tracks))
	}
	return nil
}
