package media

import (
	"fmt"
	"strings"

	"main/internal/platform"
)

var manualSyncSteps = []string{
	"Re-import only the affected albums (File → Import) — avoid duplicating entries already in your library.",
	"On your iPhone: delete those albums from the Music app (stale artwork lives on the device, not the PC).",
	"In Apple Devices / Finder: sync selected albums first and confirm track artwork before syncing your full library.",
	"After syncing, use Reset Apple sync if artwork still looks wrong until you restart Windows.",
}

// RunSyncRepair embeds artwork across configured library folders (no cache deletion).
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
			"Quit Apple Music before updating artwork, or enable force to continue anyway", false)
		res.Summary = "Quit Apple Music first, then run again."
		return res
	}
	addStep("music_running", "Apple Music closed", true, "Apple Music is not running", false)

	if opts.CacheOnly {
		addStep("prepare", "Embed artwork", true, "Skipped — cache-only mode is no longer used", true)
		res.Summary = "Use Reset Apple sync in Settings for stuck sync agents."
		return res
	}

	if opts.SkipPrepare {
		addStep("prepare", "Embed artwork", true, "Skipped by request", true)
		res.Summary = "Nothing to do."
		return res
	}

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
		addStep("prepare", "Embed artwork", true, "No folders configured — skipped", true)
	} else if errCount == 0 {
		addStep("prepare_summary", "Embed artwork in tracks", true,
			fmt.Sprintf("Prepared %d track(s) across library folders", preparedTotal), false)
	}

	if res.OK {
		res.Summary = "Artwork updated — re-import albums in Apple Music, then re-sync your iPhone."
	} else {
		res.Summary = "Finished with issues — review steps below."
	}
	return res
}
