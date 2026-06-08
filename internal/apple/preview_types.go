package apple

// PreviewTrack is one row in a download preview.
type PreviewTrack struct {
	Num         int    `json:"num"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	Type        string `json:"type"`
	Duration    string `json:"duration"`
	DurationMs  int    `json:"duration_ms"`
	Explicit    bool   `json:"explicit"`
	IsMV        bool   `json:"is_mv"`
	URL         string `json:"url,omitempty"`
	ArtURL      string `json:"art_url,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumArtist string `json:"album_artist,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Year        string `json:"year,omitempty"`
	TrackNumber int    `json:"track_number,omitempty"`
	DiscNumber  int    `json:"disc_number,omitempty"`
}

// PreviewResult is returned by PreviewURL for Apple Music links.
type PreviewResult struct {
	URL             string         `json:"url"`
	Type            string         `json:"type"`
	Title           string         `json:"title"`
	Subtitle        string         `json:"subtitle"`
	ArtURL          string         `json:"art_url"`
	TrackCount      int            `json:"track_count"`
	TotalDuration   string         `json:"total_duration"`
	Tracks          []PreviewTrack `json:"tracks"`
	CanSelectTracks bool           `json:"can_select_tracks"`
	OutputFolder    string         `json:"output_folder"`
	Error           string         `json:"error,omitempty"`
}
