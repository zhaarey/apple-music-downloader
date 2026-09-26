package classical

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type playParam struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type apiRecording struct {
	ScreenType string `json:"screenType"`
	ScreenID   string `json:"screenId"`
	Header     struct {
		WorkTitle string `json:"workTitle"`
		Composer  string `json:"composer"`
		AlbumID   string `json:"albumId"`
	} `json:"header"`
	PrimaryButton struct {
		Action struct {
			PlayParams         []playParam `json:"playParams"`
			ContainerPlayParam playParam   `json:"containerPlayParam"`
		} `json:"action"`
	} `json:"primaryButton"`
	Sections []struct {
		Type       string `json:"type"`
		Components []struct {
			Items []struct {
				Type  string `json:"type"`
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"items"`
		} `json:"components"`
	} `json:"sections"`
}

func (c *Client) fetchAPI(ctx context.Context, req Request) (*Recording, error) {
	body, err := c.get(ctx, apiPrefix+"/query/view/"+req.Storefront+"/recording/"+url.PathEscape(req.RecordingID), req.Language, "application/json")
	if err != nil {
		return nil, err
	}
	var resp apiRecording
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode recording: %w", err)
	}
	if resp.ScreenType != "recording" {
		return nil, fmt.Errorf("unexpected screen type %q", resp.ScreenType)
	}
	if resp.ScreenID != req.RecordingID {
		return nil, fmt.Errorf("%w: response is for %q", errIdentity, resp.ScreenID)
	}
	action := resp.PrimaryButton.Action
	if action.ContainerPlayParam.Kind != "album" {
		return nil, fmt.Errorf("%w: container is %q", errIdentity, action.ContainerPlayParam.Kind)
	}
	ids := make([]string, 0, len(action.PlayParams))
	for _, p := range action.PlayParams {
		if p.Kind != "song" {
			return nil, fmt.Errorf("%w: play item %s is %q", errIdentity, p.ID, p.Kind)
		}
		ids = append(ids, p.ID)
	}
	titles := map[string]string{}
	for _, section := range resp.Sections {
		if section.Type != "track-list" {
			continue
		}
		for _, component := range section.Components {
			for _, item := range component.Items {
				if item.Type != "track" {
					continue
				}
				if prev, ok := titles[item.ID]; ok && prev != item.Title {
					return nil, fmt.Errorf("%w: song %s has conflicting titles", errIdentity, item.ID)
				}
				titles[item.ID] = item.Title
			}
		}
	}
	return newRecording(req, "api", resp.Header.AlbumID, action.ContainerPlayParam.ID, resp.Header.WorkTitle, resp.Header.Composer, ids, titles)
}
