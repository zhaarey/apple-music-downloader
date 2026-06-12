package ampapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// LookupSongsByISRC finds catalog songs with an exact ISRC (case-insensitive).
func LookupSongsByISRC(storefront, isrc, language, token string) ([]SongRespData, error) {
	isrc = strings.TrimSpace(isrc)
	if isrc == "" {
		return nil, fmt.Errorf("empty isrc")
	}
	if token == "" {
		var err error
		token, err = GetToken()
		if err != nil {
			return nil, err
		}
	}
	if language == "" {
		language = "en-US"
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/songs", storefront), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	q := url.Values{}
	q.Set("filter[isrc]", isrc)
	q.Set("l", language)
	req.URL.RawQuery = q.Encode()

	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("isrc lookup: %s", do.Status)
	}
	var resp SongResp
	if err := json.NewDecoder(do.Body).Decode(&resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
