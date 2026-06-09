package media

// SyncCheck is one iPhone sync readiness check.
type SyncCheck struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Pass     bool   `json:"pass"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

// SyncValidationResult summarizes sync readiness for one audio file.
type SyncValidationResult struct {
	Path    string      `json:"path"`
	Ready   bool        `json:"ready"`
	Summary string      `json:"summary"`
	Checks  []SyncCheck `json:"checks"`
}

// FolderSyncValidationResult summarizes sync readiness for a folder of tracks.
type FolderSyncValidationResult struct {
	Folder      string                 `json:"folder"`
	Ready       bool                   `json:"ready"`
	Total       int                    `json:"total"`
	ReadyCount  int                    `json:"ready_count"`
	Summary     string                 `json:"summary"`
	Files       []SyncValidationResult `json:"files"`
}

// CacheClearResult reports cache clearing outcome.
type CacheClearResult struct {
	OK       bool     `json:"ok"`
	Message  string   `json:"message"`
	Cleared  []string `json:"cleared"`
	Errors   []string `json:"errors"`
	Platform string   `json:"platform"`
}

// AppleMusicCacheInfo describes known Apple Music artwork cache locations.
type AppleMusicCacheInfo struct {
	Paths    []string `json:"paths"`
	Platform string   `json:"platform"`
	Note     string   `json:"note"`
}
