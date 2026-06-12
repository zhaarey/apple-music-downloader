package media

import (
	"fmt"
	"strings"

	"main/internal/osutil"
	"main/internal/platform"
)

// ApplePurgeResult reports a Windows deep Apple Music cache purge.
type ApplePurgeResult struct {
	OK           bool             `json:"ok"`
	Summary      string           `json:"summary"`
	Message      string           `json:"message"`
	Steps        []SyncRepairStep `json:"steps"`
	LogPath      string           `json:"log_path"`
	NeedElevated bool             `json:"need_elevated"`
	ManualSteps  []string         `json:"manual_steps"`
}

var applePurgeManualSteps = []string{
	"Re-open Apple Music on this PC.",
	"Re-import your downloaded albums (drag album folders in, or use Add to Library).",
	"On iPhone: if artwork is still wrong, delete that album from Apple Music on the phone, then sync only that album from the PC.",
	"Your downloaded .m4a files on disk are not deleted — only PC artwork caches were cleared.",
}

// RunAppleMusicDeepPurge stops Apple Music, clears PC artwork caches, and restarts the mobile device service.
func RunAppleMusicDeepPurge(elevated bool) ApplePurgeResult {
	res := ApplePurgeResult{
		LogPath:     platform.ApplePurgeLogPath(),
		ManualSteps: applePurgeManualSteps,
	}
	if platform.GOOS() != "windows" {
		res.Summary = "Deep Apple cache purge is available on Windows only."
		res.Message = res.Summary
		return res
	}

	ok, logTail, needElevated, err := osutil.RunAppleMusicDeepPurge(elevated)
	step := SyncRepairStep{
		ID:     "deep_purge",
		Label:  "Deep purge Apple Music PC caches",
		OK:     ok && err == nil,
		Detail: strings.TrimSpace(logTail),
	}
	if err != nil {
		step.OK = false
		step.Detail = err.Error()
		if logTail != "" {
			step.Detail += "\n" + logTail
		}
	}
	res.Steps = []SyncRepairStep{step}
	res.NeedElevated = needElevated && !elevated
	res.OK = step.OK

	switch {
	case res.OK && !res.NeedElevated:
		res.Summary = "PC Apple Music caches cleared. Re-import albums, then re-sync your iPhone if needed."
		res.Message = res.Summary
	case res.OK && res.NeedElevated:
		res.Summary = "PC caches cleared. Restarting Apple Mobile Device Service needs administrator approval."
		res.Message = fmt.Sprintf("%s Use Run as administrator if you sync an iPhone over USB.", res.Summary)
	case err != nil:
		res.Summary = "Deep purge failed or was cancelled."
		res.Message = res.Summary
	default:
		res.Summary = "Deep purge completed with warnings — open the log for details."
		res.Message = res.Summary
	}
	return res
}
