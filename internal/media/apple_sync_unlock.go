package media

import (
	"fmt"
	"strings"

	"main/internal/osutil"
	"main/internal/platform"
)

// AppleSyncUnlockResult reports stopping stuck Apple Devices sync agents.
type AppleSyncUnlockResult struct {
	OK           bool     `json:"ok"`
	Summary      string   `json:"summary"`
	Message      string   `json:"message"`
	LogPath      string   `json:"log_path"`
	NeedElevated bool     `json:"need_elevated"`
	ManualSteps  []string `json:"manual_steps"`
	KilledHint   string   `json:"killed_hint,omitempty"`
}

var appleSyncUnlockManualSteps = []string{
	"Run this after a sync finishes or gets stuck — not while files are still copying.",
	"Open Apple Devices → your iPhone → sync one album first to verify artwork.",
	"If art is still wrong on iPhone: delete that album on the phone, then sync that album only from the PC.",
	"Your .m4a files on disk are not modified by this action.",
}

// ReleaseAppleSyncLock stops AMPDevicesAgent and related Windows sync processes.
func ReleaseAppleSyncLock(restartService, elevated bool) AppleSyncUnlockResult {
	res := AppleSyncUnlockResult{
		LogPath:     platform.AppleSyncUnlockLogPath(),
		ManualSteps: appleSyncUnlockManualSteps,
	}
	if platform.GOOS() != "windows" {
		res.Summary = "Release sync lock is available on Windows only."
		res.Message = res.Summary
		return res
	}

	ok, logTail, needElevated, err := osutil.RunAppleSyncUnlock(restartService, elevated)
	res.NeedElevated = needElevated && !elevated
	res.OK = ok && err == nil
	res.KilledHint = summarizeKilled(logTail)

	switch {
	case err != nil:
		res.Summary = "Sync unlock failed."
		res.Message = err.Error()
		if logTail != "" {
			res.Message += "\n" + logTail
		}
	case res.OK && res.NeedElevated:
		res.Summary = "Sync agents stopped. Restarting Apple Mobile Device Service needs administrator approval."
		res.Message = res.Summary
	case res.OK:
		res.Summary = "Apple sync agents released — try syncing again in Apple Devices."
		res.Message = res.Summary
	default:
		res.Summary = "Sync unlock completed with warnings — open the log for details."
		res.Message = res.Summary
	}
	return res
}

func summarizeKilled(logTail string) string {
	lower := strings.ToLower(logTail)
	if strings.Contains(lower, "no apple sync agent") || strings.Contains(lower, "already idle") {
		return "No stuck sync agents were running."
	}
	count := strings.Count(lower, "stopped")
	if count > 0 {
		return fmt.Sprintf("Stopped %d sync-related process(es).", count)
	}
	return ""
}
