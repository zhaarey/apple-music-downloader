// Package classical normalizes Apple Music Classical Recording pages into
// an ordered list of Catalog songs plus the metadata needed for tagging.
package classical

// Request identifies one Classical Recording in one storefront and language.
type Request struct {
	Storefront  string
	Language    string
	RecordingID string
	PublicURL   string
}

// Track is one movement of a Recording, in Recording play order.
type Track struct {
	SongID     string
	Title      string
	Position   int
	Count      int
	Conductors []string
}

// Recording is the normalized result of the API or SSR adapter.
type Recording struct {
	Request   Request
	AlbumID   string
	WorkTitle string
	Composer  string
	Tracks    []Track
	Source    string
	Warnings  []string
}
