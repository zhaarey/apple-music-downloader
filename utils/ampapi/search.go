package ampapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SearchResp represents the top-level response from the search API.
type SearchResp struct {
	Results SearchResults `json:"results"`
}

// SearchResults contains the different types of search results.
type SearchResults struct {
	Songs   *SongResults   `json:"songs,omitempty"`
	Albums  *AlbumResults  `json:"albums,omitempty"`
	Artists *ArtistResults `json:"artists,omitempty"`
}

// SongResults contains a list of song search results.
type SongResults struct {
	Href string         `json:"href"`
	Next string         `json:"next"`
	Data []SongRespData `json:"data"`
}

// AlbumResults contains a list of album search results.
type AlbumResults struct {
	Href string          `json:"href"`
	Next string          `json:"next"`
	Data []AlbumRespData `json:"data"`
}

// ArtistResults contains a list of artist search results.
type ArtistResults struct {
	Href string `json:"href"`
	Next string `json:"next"`
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Href       string `json:"href"`
		Attributes struct {
			Name       string   `json:"name"`
			GenreNames []string `json:"genreNames"`
			URL        string   `json:"url"`
		} `json:"attributes"`
	} `json:"data"`
}

// Search performs a search query against the Apple Music API.
func Search(storefront, term, types, language, token string, limit, offset int) (*SearchResp, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/search", storefront), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")

	query := url.Values{}
	query.Set("term", term)
	query.Set("types", types)
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("offset", fmt.Sprintf("%d", offset))
	query.Set("l", language)
	req.URL.RawQuery = query.Encode()

	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer do.Body.Close()

	if do.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", do.Status)
	}

	obj := new(SearchResp)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
