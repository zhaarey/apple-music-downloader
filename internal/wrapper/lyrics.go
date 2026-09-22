package wrapper

import (
	"encoding/json"
	"errors"
	"net/url"
)

// ErrLyricsNotFound is returned when wrapper-lite indicates no lyrics are available (code 404).
var ErrLyricsNotFound = errors.New("no lyrics available for this song")

// ErrNoLyrics is an alias for ErrLyricsNotFound.
var ErrNoLyrics = ErrLyricsNotFound

// Lyrics queries the wrapper-lite /lyrics endpoint for the given adamID, language, and syllable setting.
func (c *Client) Lyrics(adamID, language string, syllable bool) (string, error) {
	isSyllable := "0"
	if syllable {
		isSyllable = "1"
	}
	endpoint := "/lyrics?adamId=" + url.QueryEscape(adamID) + "&language=" + url.QueryEscape(language) + "&syllable=" + isSyllable
	body, err := c.get(endpoint)
	if err != nil {
		return "", err
	}

	var raw struct {
		Code any `json:"code"`
		Msg  string `json:"msg"`
		Data any `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err == nil {
		if isCode404(raw.Code) || isCode404(raw.Data) {
			return "", ErrLyricsNotFound
		}
		if dataMap, ok := raw.Data.(map[string]any); ok {
			if isCode404(dataMap["code"]) {
				return "", ErrLyricsNotFound
			}
		}
	}

	data, err := decodeEnvelope[LyricsData](body, endpoint)
	if err != nil {
		return "", err
	}
	if data.Code == 404 {
		return "", ErrLyricsNotFound
	}
	return data.Lyrics, nil
}

// GetLyrics queries wrapper-lite's /lyrics endpoint using baseURL.
func GetLyrics(baseURL, adamID, language string, syllable bool) (string, error) {
	return New(baseURL).Lyrics(adamID, language, syllable)
}
