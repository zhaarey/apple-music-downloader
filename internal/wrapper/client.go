package wrapper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"amdl/internal/download"
)

// ErrNotConfigured is returned when the wrapper-lite server URL is empty.
var ErrNotConfigured = errors.New("lite-server is not configured")

// Client interacts with the wrapper-lite HTTP API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New creates a new Client with default HTTP client.
func New(baseURL string) *Client {
	httpClient := download.Client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return NewWithHTTPClient(baseURL, httpClient)
}

// NewWithHTTPClient creates a new Client with a custom HTTP client.
func NewWithHTTPClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTPClient: httpClient,
	}
}

func (c *Client) get(endpoint string) ([]byte, error) {
	if c.BaseURL == "" {
		return nil, ErrNotConfigured
	}
	url := c.BaseURL + endpoint
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := c.HTTPClient
	if client == nil {
		client = download.Client
	}
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		trimmed := strings.TrimSpace(string(body))
		if resp.StatusCode == http.StatusNotFound && strings.HasPrefix(endpoint, "/lyrics") {
			var check struct {
				Code any `json:"code"`
				Data any `json:"data"`
			}
			if json.Unmarshal(body, &check) == nil {
				if isCode404(check.Code) || isCode404(check.Data) {
					return body, nil
				}
				if m, ok := check.Data.(map[string]any); ok && isCode404(m["code"]) {
					return body, nil
				}
			}
		}
		if trimmed != "" {
			return nil, fmt.Errorf("lite-server %s returned %s: %s", endpointPath(endpoint), resp.Status, trimmed)
		}
		return nil, fmt.Errorf("lite-server %s returned %s", endpointPath(endpoint), resp.Status)
	}

	return body, nil
}

func (c *Client) post(endpoint string, reqBody any) ([]byte, error) {
	if c.BaseURL == "" {
		return nil, ErrNotConfigured
	}
	url := c.BaseURL + endpoint

	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = download.Client
	}
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		trimmed := strings.TrimSpace(string(body))
		if trimmed != "" {
			return nil, fmt.Errorf("lite-server %s returned %s: %s", endpointPath(endpoint), resp.Status, trimmed)
		}
		return nil, fmt.Errorf("lite-server %s returned %s", endpointPath(endpoint), resp.Status)
	}

	return body, nil
}

func endpointPath(endpoint string) string {
	if idx := strings.IndexByte(endpoint, '?'); idx != -1 {
		return endpoint[:idx]
	}
	return endpoint
}

func decodeEnvelope[T any](body []byte, endpoint string) (T, error) {
	var zero T
	var envelope Response[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return zero, err
	}
	if envelope.Code != 0 {
		return zero, fmt.Errorf("lite-server %s returned code=%d msg=%s", endpointPath(endpoint), envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

func isCode404(v any) bool {
	switch val := v.(type) {
	case float64:
		return int(val) == 404
	case int:
		return val == 404
	case string:
		return strings.TrimSpace(val) == "404"
	default:
		return false
	}
}
