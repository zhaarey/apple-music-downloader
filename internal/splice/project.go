package splice

import "strings"
type Track struct {
	Title        string `json:"title"`
	DurationMs   int    `json:"duration_ms"`
	StartMs      *int   `json:"start_ms,omitempty"`
	Artist       string `json:"artist,omitempty"`
	TrackNumber  *int   `json:"track_number,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Album        string `json:"album,omitempty"`
	AlbumArtist  string `json:"album_artist,omitempty"`
	Genre        string `json:"genre,omitempty"`
	Year         string `json:"year,omitempty"`
	DiscNumber   *int   `json:"disc_number,omitempty"`
	DiscTotal    *int   `json:"disc_total,omitempty"`
}

func (t Track) EffectiveTrackNumber(rowIndex int) int {
	if t.TrackNumber != nil && *t.TrackNumber > 0 {
		return *t.TrackNumber
	}
	return rowIndex + 1
}

func (t Track) EffectiveArtist(albumArtistDefault string) string {
	if a := strings.TrimSpace(t.Artist); a != "" {
		return a
	}
	return strings.TrimSpace(albumArtistDefault)
}

func (t Track) TagTitle() string {
	return strings.TrimSpace(t.Title)
}

func (t Track) EffectiveAlbum(defaultAlbum string) string {
	if a := strings.TrimSpace(t.Album); a != "" {
		return a
	}
	return strings.TrimSpace(defaultAlbum)
}

func (t Track) EffectiveAlbumArtist(defaultAlbumArtist string) string {
	if a := strings.TrimSpace(t.AlbumArtist); a != "" {
		return a
	}
	return strings.TrimSpace(defaultAlbumArtist)
}

func (t Track) EffectiveGenre(defaultGenre string) string {
	if g := strings.TrimSpace(t.Genre); g != "" {
		return g
	}
	return strings.TrimSpace(defaultGenre)
}

func (t Track) EffectiveYear(defaultYear string) string {
	if y := strings.TrimSpace(t.Year); y != "" {
		return y
	}
	return strings.TrimSpace(defaultYear)
}

func (t Track) EffectiveDiscNumber(defaultDisc int) int {
	if t.DiscNumber != nil && *t.DiscNumber > 0 {
		return *t.DiscNumber
	}
	if defaultDisc > 0 {
		return defaultDisc
	}
	return 1
}

func (t Track) EffectiveDiscTotal(defaultTotal int) int {
	if t.DiscTotal != nil && *t.DiscTotal > 0 {
		return *t.DiscTotal
	}
	if defaultTotal > 0 {
		return defaultTotal
	}
	return 1
}

// AlbumMetadata describes album-level tags for exported tracks.
type AlbumMetadata struct {
	Album        string  `json:"album"`
	AlbumArtist  string  `json:"album_artist"`
	Artist       string  `json:"artist"`
	Year         string  `json:"year"`
	Genre        string  `json:"genre"`
	ArtworkPath  *string `json:"artwork_path,omitempty"`
	TotalTracks  *int    `json:"total_tracks,omitempty"`
}

func (a AlbumMetadata) EffectiveTotalTracks(trackCount int) int {
	if a.TotalTracks != nil && *a.TotalTracks > 0 {
		return *a.TotalTracks
	}
	if trackCount > 0 {
		return trackCount
	}
	return 0
}

// Project is the splice workspace (compatible with audio-splicer JSON).
type Project struct {
	MasterPath       string        `json:"master_path"`
	OutputDir        string        `json:"output_dir"`
	Album            AlbumMetadata `json:"album"`
	Tracks           []Track       `json:"tracks"`
	MasterDurationMs int           `json:"master_duration_ms"`
}

func (p *Project) ComputeStartEnd() [][3]int {
	return ComputeTrackTimings(p.Tracks, p.MasterDurationMs)
}
