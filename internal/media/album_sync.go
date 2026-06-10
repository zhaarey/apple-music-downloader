package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CollectAlbumTracks returns every .m4a/.m4b under root (recursive).
func CollectAlbumTracks(root string) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("no folder path")
	}
	out := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if IsTagAudioExt(filepath.Ext(path)) {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

type albumGroup struct {
	dir          string
	album        string
	albumArtist  string
	tracks       []string
}

func groupTracksByAlbum(paths []string) ([]albumGroup, error) {
	byDir := map[string][]string{}
	for _, p := range paths {
		dir := filepath.Dir(p)
		byDir[dir] = append(byDir[dir], p)
	}
	groups := []albumGroup{}
	for dir, tracks := range byDir {
		if len(tracks) == 0 {
			continue
		}
		info, err := ReadAudioTags(tracks[0])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tracks[0], err)
		}
		groups = append(groups, albumGroup{
			dir:         dir,
			album:       info.Album,
			albumArtist: info.AlbumArtist,
			tracks:      tracks,
		})
	}
	return groups, nil
}

// ValidateAlbumSync validates tracks and album-level consistency under root.
// When recursive is false, only .m4a/.m4b files directly in root are checked (matches Prepare scope).
func ValidateAlbumSync(ffmpegConfigured, root string, recursive bool) (FolderSyncValidationResult, error) {
	out := FolderSyncValidationResult{Folder: root, Files: []SyncValidationResult{}}
	paths, err := albumTagPaths(root, recursive)
	if err != nil {
		return out, err
	}
	if len(paths) == 0 {
		out.Summary = "No .m4a files found under this folder."
		return out, nil
	}
	for _, p := range paths {
		res, err := ValidateIPhoneSync(ffmpegConfigured, p)
		if err != nil {
			res = SyncValidationResult{
				Path:    p,
				Ready:   false,
				Summary: err.Error(),
				Checks:  []SyncCheck{check("read", "Read file", false, err.Error(), "fail")},
			}
		}
		out.Files = append(out.Files, res)
	}
	groups, _ := groupTracksByAlbum(paths)
	for _, g := range groups {
		hashes := map[string]int{}
		for _, p := range g.tracks {
			h, err := EmbeddedCoverHash(p)
			if err != nil {
				continue
			}
			hashes[h]++
		}
		if len(hashes) > 1 {
			for i := range out.Files {
				for _, tp := range g.tracks {
					if out.Files[i].Path == tp {
						out.Files[i].Checks = append(out.Files[i].Checks, check("album_art_match", "Album artwork match", false,
							"Tracks in this folder have different embedded covers — iPhone may show one art for the album; fix each track in Tag Editor if needed", "warn"))
					}
				}
			}
		}
	}
	out.FolderChecks = folderArtworkDiagnostics(paths)
	out.Total = len(out.Files)
	for _, f := range out.Files {
		if f.Ready {
			out.ReadyCount++
		}
	}
	out.Ready = out.Total > 0 && out.ReadyCount == out.Total
	if out.Total == 0 {
		out.Summary = "No .m4a files found under this folder."
	} else if out.Ready {
		out.Summary = fmt.Sprintf("All %d track(s) in this tree look ready for Apple Music sync.", out.Total)
	} else {
		out.Summary = fmt.Sprintf("%d of %d track(s) ready — fix issues or run Prepare album for sync.", out.ReadyCount, out.Total)
	}
	return out, nil
}

func folderArtworkDiagnostics(paths []string) []SyncCheck {
	if len(paths) == 0 {
		return nil
	}
	checks := []SyncCheck{}
	hashSet := map[string]struct{}{}
	albums := map[string]struct{}{}
	albumArtists := map[string]struct{}{}
	for _, p := range paths {
		if h, err := EmbeddedCoverHash(p); err == nil && h != "" {
			hashSet[h] = struct{}{}
		}
		if info, err := ReadAudioTags(p); err == nil {
			if a := strings.TrimSpace(info.Album); a != "" {
				albums[a] = struct{}{}
			}
			if aa := strings.TrimSpace(info.AlbumArtist); aa != "" {
				albumArtists[aa] = struct{}{}
			}
		}
	}
	if len(albums) > 1 {
		checks = append(checks, check("folder_multi_album", "One album per folder", false,
			fmt.Sprintf("%d different album titles in this folder — bulk artwork update is only for a single album", len(albums)), "warn"))
	}
	switch len(hashSet) {
	case 0:
		checks = append(checks, check("folder_art_none", "Embedded artwork", false,
			"No embedded covers found — iPhone sync needs art inside each file", "fail"))
	case 1:
		if len(paths) > 1 {
			checks = append(checks, check("folder_art_same", "Embedded artwork", true,
				"All tracks share one identical embedded cover (normal for a single album). iPhone shows one artwork per album — if the phone shows the wrong image, delete the album on the iPhone and re-sync; PC tools cannot reset device cache.", "pass"))
		}
	default:
		checks = append(checks, check("folder_art_mixed", "Embedded artwork", false,
			fmt.Sprintf("%d different embedded covers in this folder — Apple Music on iPhone may pick one for the whole album", len(hashSet)), "warn"))
	}
	if len(albumArtists) > 1 {
		checks = append(checks, check("folder_multi_album_artist", "Album artist", false,
			"Multiple album artists in one folder — iPhone may group or display artwork unexpectedly", "warn"))
	}
	return checks
}

func resolveGroupCover(dir string, tracks []string) ([]byte, string, error) {
	hashCounts := map[string]int{}
	var sampleByHash = map[string][]byte{}

	for _, p := range tracks {
		h, err := EmbeddedCoverHash(p)
		if err != nil {
			continue
		}
		hashCounts[h]++
		if _, ok := sampleByHash[h]; !ok {
			data, err := ReadNormalizedEmbeddedCover(p)
			if err != nil {
				continue
			}
			sampleByHash[h] = data
		}
	}

	sidecar := FindAlbumCoverFile(dir)

	// All tracks already share one embedded cover — keep it (do not override with a stale sidecar).
	if len(hashCounts) == 1 {
		for h := range hashCounts {
			return sampleByHash[h], "embedded art (all tracks match)", nil
		}
	}

	if sidecar != "" {
		data, err := os.ReadFile(sidecar)
		if err != nil {
			return nil, "", err
		}
		norm, err := NormalizeCoverForApple(data)
		if err != nil {
			return nil, "", err
		}
		return norm, filepath.Base(sidecar) + " (folder sidecar)", nil
	}

	if len(hashCounts) > 0 {
		bestHash := ""
		bestCount := 0
		for h, count := range hashCounts {
			if count > bestCount {
				bestCount = count
				bestHash = h
			}
		}
		if data, ok := sampleByHash[bestHash]; ok {
			return data, "embedded art (majority of tracks)", nil
		}
	}

	return nil, "", fmt.Errorf("no cover.jpg or embedded art in %s", dir)
}

// PrepareAlbumForSync re-embeds normalized JPEG artwork on tracks under root.
// Text metadata (title, artist, track numbers, etc.) is preserved — only the covr atom changes.
// When recursive is false, only files directly in root are modified (safer for single-album folders).
func PrepareAlbumForSync(ffmpegConfigured, root string, recursive bool) (AlbumPrepareResult, error) {
	out := AlbumPrepareResult{Folder: root}
	paths, err := albumTagPaths(root, recursive)
	if err != nil {
		return out, err
	}
	if len(paths) == 0 {
		out.Summary = "No .m4a files found to prepare."
		return out, nil
	}
	groups, err := groupTracksByAlbum(paths)
	if err != nil {
		return out, err
	}
	for _, g := range groups {
		coverData, _, err := resolveGroupCover(g.dir, g.tracks)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", g.dir, err))
			continue
		}
		for _, p := range g.tracks {
			if TrackArtworkAlreadyPrepared(p, coverData) {
				out.Skipped++
				continue
			}
			if err := WriteTrackArtworkOnly(p, coverData); err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", p, err))
				continue
			}
			out.Prepared++
		}
	}
	switch {
	case len(out.Errors) == 0 && out.Prepared == 0 && out.Skipped > 0:
		out.Summary = fmt.Sprintf("All %d track(s) already had matching artwork — nothing changed.", out.Skipped)
	case len(out.Errors) == 0:
		if out.Skipped > 0 {
			out.Summary = fmt.Sprintf("Updated artwork on %d track(s); %d already matched.", out.Prepared, out.Skipped)
		} else {
			out.Summary = fmt.Sprintf("Updated artwork on %d track(s). Title and other tags were not changed.", out.Prepared)
		}
	default:
		out.Summary = fmt.Sprintf("Updated %d track(s) with %d error(s).", out.Prepared, len(out.Errors))
	}
	_ = ffmpegConfigured
	return out, nil
}
