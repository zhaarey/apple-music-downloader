package ampapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"amdl/internal/download"
)

// GetDefaultLanguage returns the storefront's default language tag: the
// language the Catalog answers in when no l parameter is sent.
func GetDefaultLanguage(storefront, token string) (string, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return "", err
		}
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com/v1/storefronts/%s", storefront), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	do, err := download.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return "", errors.New(do.Status)
	}
	var resp struct {
		Data []struct {
			Attributes struct {
				DefaultLanguageTag string `json:"defaultLanguageTag"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(do.Body).Decode(&resp); err != nil {
		return "", err
	}
	if len(resp.Data) == 0 || resp.Data[0].Attributes.DefaultLanguageTag == "" {
		return "", fmt.Errorf("storefront %s has no default language", storefront)
	}
	return resp.Data[0].Attributes.DefaultLanguageTag, nil
}
