package wrapper

import (
	"errors"
	"net/url"
)

// Webplayback queries the wrapper-lite /webplayback endpoint for the given adamID and returns the m3u8 URL.
func (c *Client) Webplayback(adamID string) (string, error) {
	endpoint := "/webplayback?adamId=" + url.QueryEscape(adamID)
	body, err := c.get(endpoint)
	if err != nil {
		return "", err
	}
	data, err := decodeEnvelope[WebplaybackData](body, endpoint)
	if err != nil {
		return "", err
	}
	if data.M3u8 == "" {
		return "", errors.New("Unavailable")
	}
	return data.M3u8, nil
}

// GetWebplayback queries wrapper-lite's /webplayback endpoint using baseURL.
func GetWebplayback(baseURL, adamID string) (string, error) {
	return New(baseURL).Webplayback(adamID)
}
