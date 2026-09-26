package classical

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const Host = "classical.music.apple.com"

// ParseRecordingURL reports whether raw is a Classical Recording link and, if
// so, resolves it. Links on other hosts or of other kinds return false so the
// caller keeps its existing handling.
func ParseRecordingURL(raw, configLanguage string) (Request, bool, error) {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Hostname(), Host) {
		return Request{}, false, nil
	}
	segments := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(segments) < 2 || segments[1] != "recording" {
		return Request{}, false, nil
	}
	if u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return Request{}, true, errors.New("classical recording URL must be plain https on the default port")
	}
	if len(segments) != 3 || segments[2] == "" {
		return Request{}, true, errors.New("classical recording URL must be /{storefront}/recording/{slug}")
	}
	storefront := strings.ToLower(segments[0])
	if len(storefront) != 2 || storefront[0] < 'a' || storefront[0] > 'z' || storefront[1] < 'a' || storefront[1] > 'z' {
		return Request{}, true, fmt.Errorf("invalid storefront %q", segments[0])
	}
	slug, err := url.PathUnescape(segments[2])
	if err != nil || slug == "." || slug == ".." || strings.ContainsAny(slug, `/\`) {
		return Request{}, true, fmt.Errorf("invalid recording id %q", segments[2])
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return Request{}, true, fmt.Errorf("invalid query: %w", err)
	}
	if _, ok := query["i"]; ok {
		return Request{}, true, errors.New("a recording link cannot select a single song with ?i=")
	}
	if len(query["l"]) > 1 {
		return Request{}, true, errors.New("duplicate l parameter")
	}
	// The config language wins, as it does for other links; the link's l
	// applies only when the config sets none.
	language := configLanguage
	if language == "" {
		language = query.Get("l")
	}
	return Request{Storefront: storefront, Language: language, RecordingID: slug, PublicURL: raw}, true, nil
}
