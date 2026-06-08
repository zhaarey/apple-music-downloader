package youtube

// DownloadMeta is editable metadata for YouTube → Apple Music export.
type DownloadMeta struct {
	Num         int    `json:"num"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Genre       string `json:"genre"`
	Year        string `json:"year"`
	TrackNumber int    `json:"track_number"`
	DiscNumber  int    `json:"disc_number"`
	TrackTotal  int    `json:"track_total"`
	ArtURL      string `json:"art_url,omitempty"`
}

// HandoffPayload is sent to the Split mix tab after a DJ set download.
type HandoffPayload struct {
	MasterPath  string `json:"master_path"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Artist      string `json:"artist"`
	Year        string `json:"year"`
	Genre       string `json:"genre"`
	ArtURL      string `json:"art_url,omitempty"`
}
