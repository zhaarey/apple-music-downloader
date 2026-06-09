package media

import (
	"fmt"
	"strings"

	"main/internal/platform"
)

var manualSyncSteps = []string{
	"Quit Apple Music / Music.app on your PC before clearing caches or updating artwork.",
	"Re-import only the affected albums (File → Import) — avoid duplicating entries already in your library.",
	"On your iPhone: delete those albums from the Music app (stale artwork lives on the device, not the PC).",
	"In Apple Devices / Finder: sync selected albums first and confirm track artwork before syncing your full library.",
	"If Sync Library is on: turn it off briefly, delete the bad albums on iPhone, then re-enable and sync.",
}

// RunSyncRepair prepares library folders and clears PC artwork caches.
func RunSyncRepair(opts SyncRepairOptions) SyncRepairResult {
	res := SyncRepairResult{
		OK:          true,
		ManualSteps: manualSyncSteps,
		LogPath:     platform.SyncRepairLogPath(),
	}
	addStep := func(id, label string, ok bool, detail string, skipped bool) {
		res.Steps = append(res.Steps, SyncRepairStep{ID: id, Label: label, OK: ok, Detail: detail, Skipped: skipped})
		if !ok && !skipped {
			res.OK = false
		}
	}

	if platform.IsAppleMusicRunning() && !opts.ForceIfMusicRunning {
		addStep("music_running", "Apple Music closed", false,
			"Quit Apple Music / Music.app before repair, or enable force to continue anyway", false)
		res.Summary = "Quit Apple Music first, then run repair again."
		res.NeedElevated = false
		return res
	}
	addStep("music_running", "Apple Music closed", true, "Apple Music is not running", false)

	if !opts.SkipPrepare && !opts.CacheOnly {
		preparedTotal := 0
		errCount := 0
		for _, folder := range opts.PrepareFolders {
			folder = strings.TrimSpace(folder)
			if folder == "" {
				continue
			}
			prep, err := PrepareAlbumForSync("", folder, true)
			if err != nil {
				errCount++
				addStep("prepare_"+folder, "Prepare "+folder, false, err.Error(), false)
				continue
			}
			preparedTotal += prep.Prepared
			if len(prep.Errors) > 0 {
				errCount += len(prep.Errors)
				addStep("prepare_"+folder, "Prepare "+folder, false, prep.Summary, false)
			} else {
				addStep("prepare_"+folder, "Prepare "+folder, true, prep.Summary, false)
			}
		}
		if len(opts.PrepareFolders) == 0 {
			addStep("prepare", "Prepare library folders", true, "No folders configured — skipped", true)
		} else if errCount == 0 {
			addStep("prepare_summary", "Embed artwork in tracks", true,
				fmt.Sprintf("Prepared %d track(s) across library folders", preparedTotal), false)
		}
	} else {
		addStep("prepare", "Prepare library folders", true, "Skipped by request", true)
	}

	appClear := ClearAppTempCache()
	addStep("app_cache", "Clear app temp files", appClear.OK, appClear.Message, false)

	appleClear := ClearAppleMusicArtworkCache()
	addStep("apple_cache", "Clear Apple Music artwork cache", appleClear.OK, appleClear.Message, false)
	if !appleClear.OK && hasAccessDenied(appleClear.Errors) {
		res.NeedElevated = true
	}

	if res.OK && res.NeedElevated {
		res.Summary = "Tags prepared; some caches need administrator access — use Run as administrator."
	} else if res.OK {
		res.Summary = "Repair complete — re-import albums in Apple Music, then re-sync your iPhone."
	} else {
		res.Summary = "Repair finished with issues — review steps below."
	}
	return res
}

func hasAccessDenied(errs []string) bool {
	for _, e := range errs {
		lower := strings.ToLower(e)
		if strings.Contains(lower, "access is denied") || strings.Contains(lower, "permission denied") {
			return true
		}
	}
	return false
}
