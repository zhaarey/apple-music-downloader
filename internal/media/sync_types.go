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
	Folder       string                 `json:"folder"`
	Ready        bool                   `json:"ready"`
	Total        int                    `json:"total"`
	ReadyCount   int                    `json:"ready_count"`
	Summary      string                 `json:"summary"`
	FolderChecks []SyncCheck            `json:"folder_checks,omitempty"`
	Files        []SyncValidationResult `json:"files"`
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

// AlbumPrepareResult reports batch tag normalization for a folder tree.
type AlbumPrepareResult struct {
	Folder   string   `json:"folder"`
	Prepared int      `json:"prepared"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
	Summary  string   `json:"summary"`
}

// AlbumPreparePreview describes what PrepareAlbumForSync would change (no writes).
type AlbumPreparePreview struct {
	Folder      string `json:"folder"`
	TrackCount  int    `json:"track_count"`
	CoverSource string `json:"cover_source"`
	Recursive   bool   `json:"recursive"`
	Warning     string `json:"warning"`
}

// SyncRepairPreparePreview summarizes library-wide prepare during repair.
type SyncRepairPreparePreview struct {
	Folders     []string `json:"folders"`
	TrackCount  int      `json:"track_count"`
	Warning     string   `json:"warning"`
}

// SyncRepairStep is one step in the repair workflow.
type SyncRepairStep struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail"`
	Skipped bool   `json:"skipped"`
}

// SyncRepairResult summarizes a full sync repair run.
type SyncRepairResult struct {
	OK           bool             `json:"ok"`
	Summary      string           `json:"summary"`
	Steps        []SyncRepairStep `json:"steps"`
	NeedElevated bool             `json:"need_elevated"`
	LogPath      string           `json:"log_path"`
	ManualSteps  []string         `json:"manual_steps"`
}

// SyncRepairOptions configures RunSyncRepair.
type SyncRepairOptions struct {
	PrepareFolders       []string `json:"prepare_folders"`
	SkipPrepare          bool     `json:"skip_prepare"`
	ForceIfMusicRunning  bool     `json:"force_if_music_running"`
	CacheOnly            bool     `json:"cache_only"`
}
