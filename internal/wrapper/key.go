package wrapper

import (
	"encoding/json"
	"net/url"
)

// Key queries the wrapper-lite /key endpoint for adamID and uri, returning the KeyTemplate.
// If adamID is "0" and uri matches PrefetchKey, the embedded prefetch template is returned without network IO.
func (c *Client) Key(adamID, uri string) (*KeyTemplate, error) {
	if adamID == "0" && uri == PrefetchKey {
		var tmpl KeyTemplate
		if err := json.Unmarshal([]byte(PrefetchTemplateJSON), &tmpl); err != nil {
			return nil, err
		}
		return &tmpl, nil
	}

	endpoint := "/key?adamId=" + url.QueryEscape(adamID) + "&uri=" + url.QueryEscape(uri)
	body, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}
	data, err := decodeEnvelope[KeyTemplate](body, endpoint)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// KeyJSON queries the wrapper-lite /key endpoint and returns the JSON-serialized template bytes
// expected by temari.FromJSON.
// If adamID is "0" and uri matches PrefetchKey, the embedded prefetch template bytes are returned.
func (c *Client) KeyJSON(adamID, uri string) ([]byte, error) {
	if adamID == "0" && uri == PrefetchKey {
		return []byte(PrefetchTemplateJSON), nil
	}

	tmpl, err := c.Key(adamID, uri)
	if err != nil {
		return nil, err
	}
	return json.Marshal(tmpl)
}

// GetKeyTemplate queries wrapper-lite's /key endpoint using baseURL.
func GetKeyTemplate(baseURL, adamID, uri string) (*KeyTemplate, error) {
	return New(baseURL).Key(adamID, uri)
}

// GetKeyTemplateJSON queries wrapper-lite's /key endpoint using baseURL and returns JSON bytes.
func GetKeyTemplateJSON(baseURL, adamID, uri string) ([]byte, error) {
	return New(baseURL).KeyJSON(adamID, uri)
}
