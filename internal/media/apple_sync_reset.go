package media

import (
	"fmt"
	"strings"

	"main/internal/osutil"
	"main/internal/platform"
)

// AppleSyncResetResult reports resetting the Apple sync process stack on Windows.
type AppleSyncResetResult struct {
	OK           bool     `json:"ok"`
	Summary      string   `json:"summary"`
	Message      string   `json:"message"`
	LogPath      string   `json:"log_path"`
	NeedElevated bool     `json:"need_elevated"`
	ManualSteps  []string `json:"manual_steps"`
	StoppedHint  string   `json:"stopped_hint,omitempty"`
}

// AppleSyncUnlockResult is kept for generated bindings compatibility.
type AppleSyncUnlockResult = AppleSyncResetResult

var appleSyncResetManualSteps = []string{
	"Run after a sync finishes or gets stuck — not while files are still copying.",
	"Reopen Apple Music and Apple Devices, then sync one album first on your iPhone.",
	"If art is still wrong on iPhone: delete that album on the phone, then sync that album only from the PC.",
	"Does not delete caches or change your .m4a files.",
}

// ResetAppleSyncStack stops Apple Music and sync agents, then restarts the USB device service.
func ResetAppleSyncStack(elevated bool) AppleSyncResetResult {
	res := AppleSyncResetResult{
		LogPath:     platform.AppleSyncResetLogPath(),
		ManualSteps: appleSyncResetManualSteps,
	}
	if platform.GOOS() != "windows" {
		res.Summary = "Reset Apple sync is available on Windows only."
		res.Message = res.Summary
		return res
	}

	ok, logTail, needElevated, err := osutil.RunAppleSyncReset(elevated)
	res.NeedElevated = needElevated && !elevated
	res.OK = ok && err == nil
	res.StoppedHint = summarizeStopped(logTail)

	switch {
	case err != nil:
		res.Summary = "Sync reset failed."
		res.Message = err.Error()
		if logTail != "" {
			res.Message += "\n" + logTail
		}
	case res.OK && res.NeedElevated:
		res.Summary = "Processes stopped. USB service restart needs administrator approval."
		res.Message = res.Summary
	case res.OK:
		res.Summary = "Apple sync stack reset — reopen Apple Devices and try syncing again."
		res.Message = res.Summary
	default:
		res.Summary = "Sync reset completed with warnings — open the log for details."
		res.Message = res.Summary
	}
	return res
}

// ReleaseAppleSyncLock is deprecated; calls ResetAppleSyncStack.
func ReleaseAppleSyncLock(_, elevated bool) AppleSyncResetResult {
	return ResetAppleSyncStack(elevated)
}

func summarizeStopped(logTail string) string {
	lower := strings.ToLower(logTail)
	if strings.Contains(lower, "already idle") {
		return "No Apple sync processes were running."
	}
	count := strings.Count(lower, "stopped")
	if count > 0 {
		return fmt.Sprintf("Stopped %d process(es) and restarted USB service.", count)
	}
	return ""
}
