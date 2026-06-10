package engine

import "strings"

// BulkQueueEntry is one album/playlist URL in a bulk GUI download with optional track filtering.
type BulkQueueEntry struct {
	URL               string `json:"url"`
	SelectedTrackNums []int  `json:"selected_track_nums"`
	ForceTrackNums    []int  `json:"force_track_nums"`
}

var forceRedownloadNums map[int]bool

func resetBulkEntryState() {
	guiSelectedTracks = nil
	dl_select = false
	forceRedownloadNums = nil
}

func applyBulkEntryForURL(urlRaw string, entries []BulkQueueEntry) {
	resetBulkEntryState()
	if len(entries) == 0 {
		return
	}
	norm := strings.TrimSpace(strings.ToLower(normalizeAppleCatalogURL(urlRaw)))
	for _, ent := range entries {
		if strings.TrimSpace(strings.ToLower(normalizeAppleCatalogURL(ent.URL))) != norm {
			continue
		}
		if len(ent.SelectedTrackNums) > 0 {
			guiSelectedTracks = append([]int(nil), ent.SelectedTrackNums...)
			dl_select = true
		}
		if len(ent.ForceTrackNums) > 0 {
			forceRedownloadNums = make(map[int]bool, len(ent.ForceTrackNums))
			for _, n := range ent.ForceTrackNums {
				forceRedownloadNums[n] = true
			}
		}
		return
	}
}

func shouldForceRedownloadTrack(taskNum int) bool {
	if forceRedownloadNums == nil {
		return false
	}
	return forceRedownloadNums[taskNum]
}
