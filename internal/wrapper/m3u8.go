package wrapper

import (
	"net/url"
)

// M3U8 queries the wrapper-lite /m3u8 endpoint for the given adamID and returns the playlist URL.
func (c *Client) M3U8(adamID string) (string, error) {
	endpoint := "/m3u8?adamId=" + url.QueryEscape(adamID)
	body, err := c.get(endpoint)
	if err != nil {
		return "", err
	}
	data, err := decodeEnvelope[M3U8Data](body, endpoint)
	if err != nil {
		return "", err
	}
	return data.M3u8, nil
}

// GetM3U8 queries wrapper-lite's /m3u8 endpoint using baseURL for the given adamID.
func GetM3U8(baseURL, adamID string) (string, error) {
	return New(baseURL).M3U8(adamID)
}
