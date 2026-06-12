package spotify

// Track is minimal metadata used to search Apple Music.
type Track struct {
	ID     string `json:"id,omitempty"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album,omitempty"`
	ISRC   string `json:"isrc,omitempty"`
	URL    string `json:"url,omitempty"`
}

// ResolveResult is the Spotify side of a catalog lookup.
type ResolveResult struct {
	Kind  LinkKind `json:"kind"`
	Title string   `json:"title"`
	URL   string   `json:"url"`
	Tracks []Track `json:"tracks"`
}
