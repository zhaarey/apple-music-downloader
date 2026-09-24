package classical

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"golang.org/x/net/html"
)

type ssrItem struct {
	Type        string `json:"type"`
	Identifiers struct {
		ID string `json:"id"`
	} `json:"identifiers"`
}

type ssrDocument struct {
	Data []struct {
		Intent struct {
			Storefront string `json:"storefront"`
			PageType   string `json:"pageType"`
			ID         string `json:"id"`
		} `json:"intent"`
		Data struct {
			ScreenType string `json:"screenType"`
			ScreenID   string `json:"screenId"`
			Header     struct {
				WorkTitle     string `json:"workTitle"`
				Composer      string `json:"composer"`
				AlbumID       string `json:"albumId"`
				PrimaryButton struct {
					Action struct {
						Items         []ssrItem `json:"items"`
						ContainerItem ssrItem   `json:"containerItem"`
					} `json:"action"`
				} `json:"primaryButton"`
			} `json:"header"`
			Sections []struct {
				ItemKind string `json:"itemKind"`
				Items    []struct {
					Title             string `json:"title"`
					ContextMenuAction struct {
						URL string `json:"url"`
					} `json:"contextMenuAction"`
				} `json:"items"`
			} `json:"sections"`
		} `json:"data"`
	} `json:"data"`
}

func (c *Client) fetchSSR(ctx context.Context, req Request) (*Recording, error) {
	page, err := c.get(ctx, "/"+req.Storefront+"/recording/"+url.PathEscape(req.RecordingID), req.Language, "text/html")
	if err != nil {
		return nil, err
	}
	raw, err := serializedServerData(page)
	if err != nil {
		return nil, err
	}
	var doc ssrDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode page data: %w", err)
	}
	for _, entry := range doc.Data {
		intent, data := entry.Intent, entry.Data
		if intent.PageType != "recording" || intent.ID != req.RecordingID || intent.Storefront != req.Storefront {
			continue
		}
		if data.ScreenType != "recording" || data.ScreenID != req.RecordingID {
			return nil, fmt.Errorf("%w: page data is for %q", errIdentity, data.ScreenID)
		}
		action := data.Header.PrimaryButton.Action
		if action.ContainerItem.Type != "album" {
			return nil, fmt.Errorf("%w: container is %q", errIdentity, action.ContainerItem.Type)
		}
		ids := make([]string, 0, len(action.Items))
		for _, item := range action.Items {
			if item.Type != "song" {
				return nil, fmt.Errorf("%w: play item %s is %q", errIdentity, item.Identifiers.ID, item.Type)
			}
			ids = append(ids, item.Identifiers.ID)
		}
		// Each lockup's play action repeats the whole list, so the song a title
		// belongs to comes from its context-menu entity id instead.
		titles := map[string]string{}
		for _, section := range data.Sections {
			if section.ItemKind != "trackLockup" {
				continue
			}
			for _, item := range section.Items {
				menu, err := url.Parse(item.ContextMenuAction.URL)
				if err != nil || menu.Query().Get("entityType") != "track" {
					continue
				}
				titles[menu.Query().Get("entityId")] = item.Title
			}
		}
		return newRecording(req, "ssr", data.Header.AlbumID, action.ContainerItem.Identifiers.ID, data.Header.WorkTitle, data.Header.Composer, ids, titles)
	}
	return nil, errors.New("page has no data for this recording")
}

// serializedServerData returns the text of the script element whose id is
// serialized-server-data, regardless of attribute order.
func serializedServerData(page []byte) ([]byte, error) {
	z := html.NewTokenizer(bytes.NewReader(page))
	inTarget := false
	for {
		switch z.Next() {
		case html.ErrorToken:
			return nil, errors.New("page has no serialized-server-data script")
		case html.StartTagToken:
			name, hasAttr := z.TagName()
			if string(name) != "script" {
				continue
			}
			for hasAttr {
				var key, val []byte
				key, val, hasAttr = z.TagAttr()
				if string(key) == "id" && string(val) == "serialized-server-data" {
					inTarget = true
				}
			}
		case html.TextToken:
			if inTarget {
				return z.Text(), nil
			}
		case html.EndTagToken:
			inTarget = false
		}
	}
}
