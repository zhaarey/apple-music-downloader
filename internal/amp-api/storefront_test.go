package ampapi

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"amdl/internal/download"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// serveStorefront makes download.Client answer every request with status and
// body, and returns the requests it received.
func serveStorefront(t *testing.T, status int, body string) *[]*http.Request {
	t.Helper()
	var requests []*http.Request
	orig := download.Client
	t.Cleanup(func() { download.Client = orig })
	download.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r)
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}
	return &requests
}

func TestGetDefaultLanguage(t *testing.T) {
	// Shape of the real answer for jp, 2026-09-26.
	requests := serveStorefront(t, http.StatusOK, `{"data":[{"id":"jp","type":"storefronts","href":"/v1/storefronts/jp",`+
		`"attributes":{"supportedLanguageTags":["ja","en-US"],"defaultLanguageTag":"ja","name":"Japan","explicitContentPolicy":"allowed"}}]}`)

	language, err := GetDefaultLanguage("jp", "dev-token")
	if err != nil || language != "ja" {
		t.Fatalf("language = %q, %v; want ja", language, err)
	}
	req := (*requests)[0]
	if req.URL.String() != "https://amp-api.music.apple.com/v1/storefronts/jp" || req.Header.Get("Authorization") != "Bearer dev-token" {
		t.Fatalf("request = %s with Authorization %q", req.URL, req.Header.Get("Authorization"))
	}
}

func TestGetDefaultLanguageFailsWithoutATag(t *testing.T) {
	cases := []struct {
		status int
		body   string
	}{
		{http.StatusNotFound, `{"errors":[{"status":"404"}]}`},
		{http.StatusOK, `{"data":[]}`},
		{http.StatusOK, `{"data":[{"id":"jp","attributes":{"defaultLanguageTag":""}}]}`},
	}
	for _, tc := range cases {
		serveStorefront(t, tc.status, tc.body)
		if language, err := GetDefaultLanguage("jp", "dev-token"); err == nil {
			t.Errorf("status %d body %s: got %q, want an error", tc.status, tc.body, language)
		}
	}
}
