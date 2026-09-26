package classical

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL    = "https://" + Host
	apiPrefix  = "/api/classical/v10"
	maxBodyLen = 16 << 20
)

// errIdentity marks responses that describe a different or inconsistent
// Recording. Such errors are final: falling back to SSR must not mask them.
var errIdentity = errors.New("classical recording identity conflict")

// Client fetches public Classical Recording data without user credentials.
type Client struct {
	http *http.Client
}

// NewClient copies hc so the shared download client keeps its own settings.
// Cookies are never sent and redirects must stay on the same HTTPS page.
func NewClient(hc *http.Client) *Client {
	c := *hc
	c.Jar = nil
	if c.Timeout == 0 {
		c.Timeout = 60 * time.Second
	}
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" || !strings.EqualFold(req.URL.Host, Host) || req.URL.Path != via[0].URL.Path {
			return fmt.Errorf("refusing redirect to %s", req.URL.Redacted())
		}
		return nil
	}
	return &Client{http: &c}
}

// Fetch returns the Recording from the JSON API, falling back to the page's
// server-rendered data when the API cannot provide the core fields.
func (c *Client) Fetch(ctx context.Context, req Request) (*Recording, error) {
	rec, apiErr := c.fetchAPI(ctx, req)
	if apiErr != nil {
		if errors.Is(apiErr, errIdentity) {
			return nil, apiErr
		}
		var ssrErr error
		rec, ssrErr = c.fetchSSR(ctx, req)
		if ssrErr != nil {
			return nil, fmt.Errorf("classical recording %s: api: %v; page: %w", req.RecordingID, apiErr, ssrErr)
		}
		rec.Warnings = append(rec.Warnings, fmt.Sprintf("classical API unavailable, used page data: %v", apiErr))
	}
	if err := c.addMetadata(ctx, rec); err != nil {
		rec.Warnings = append(rec.Warnings, fmt.Sprintf("track metadata unavailable: %v", err))
	}
	return rec, nil
}

// get requests path (relative to the site root) with the selected language and
// returns the body only when the status and content type match.
func (c *Client) get(ctx context.Context, path, language, wantType string) ([]byte, error) {
	u := baseURL + path
	if language != "" {
		u += "?l=" + url.QueryEscape(language)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", wantType)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType != wantType {
		return nil, fmt.Errorf("unexpected content type %q", mediaType)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxBodyLen))
}

// newRecording validates the core fields shared by both adapters.
func newRecording(req Request, source, albumID, containerID, workTitle, composer string, ids []string, titles map[string]string) (*Recording, error) {
	if albumID == "" || albumID != containerID {
		return nil, fmt.Errorf("%w: album %q does not match container %q", errIdentity, albumID, containerID)
	}
	if len(ids) == 0 {
		return nil, errors.New("recording has no songs")
	}
	if workTitle == "" || composer == "" {
		return nil, errors.New("recording is missing work title or composer")
	}
	rec := &Recording{Request: req, AlbumID: albumID, WorkTitle: workTitle, Composer: composer, Source: source}
	seen := make(map[string]bool, len(ids))
	for i, id := range ids {
		if !isDecimalID(id) {
			return nil, fmt.Errorf("%w: invalid song id %q", errIdentity, id)
		}
		if seen[id] {
			return nil, fmt.Errorf("%w: duplicate song id %s", errIdentity, id)
		}
		seen[id] = true
		if titles[id] == "" {
			return nil, fmt.Errorf("no title for song %s", id)
		}
		rec.Tracks = append(rec.Tracks, Track{SongID: id, Title: titles[id], Position: i + 1, Count: len(ids)})
	}
	return rec, nil
}

func isDecimalID(id string) bool {
	if id == "" || id[0] == '0' {
		return false
	}
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
