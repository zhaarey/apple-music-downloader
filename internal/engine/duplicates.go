package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"main/internal/apple"
	"main/internal/youtube"
)

// TrackDuplicateStatus is duplicate detection for one preview row.
type TrackDuplicateStatus struct {
	Num              int    `json:"num"`
	OnDisk           bool   `json:"on_disk"`
	ExistingPath     string `json:"existing_path,omitempty"`
	ExistingRoot     string `json:"existing_root_label,omitempty"`
	ExpectedPath     string `json:"expected_path,omitempty"`
	ExpectedFilename string `json:"expected_filename,omitempty"`
}

// DuplicateCheckResult summarizes on-disk matches before download.
type DuplicateCheckResult struct {
	Roots         []string               `json:"roots"`
	Tracks        []TrackDuplicateStatus `json:"tracks"`
	ExistingCount int                    `json:"existing_count"`
	SelectedCount int                    `json:"selected_count"`
}

// DuplicateCheckRoots returns output folder plus any extra folders configured for duplicate checks.
func DuplicateCheckRoots(primaryOutput string) []string {
	primaryOutput = strings.TrimSpace(primaryOutput)
	seen := map[string]struct{}{}
	out := []string{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	add(primaryOutput)
	for _, p := range Config.DuplicateCheckFolders {
		add(p)
	}
	return out
}

func rootLabel(root, primaryOutput string) string {
	if strings.EqualFold(filepath.Clean(root), filepath.Clean(primaryOutput)) {
		return "output folder"
	}
	return root
}

// LocateExistingMedia finds a track file under check roots (expected relative path, then filename search).
func LocateExistingMedia(roots []string, relPath, basename string, extraNames ...string) (path, root string, ok bool) {
	names := []string{basename}
	for _, n := range extraNames {
		n = strings.TrimSpace(n)
		if n != "" {
			names = append(names, n)
		}
	}
	relPath = filepath.Clean(strings.TrimPrefix(filepath.Clean(relPath), string(os.PathSeparator)))

	for _, r := range roots {
		if relPath != "" && relPath != "." {
			candidate := filepath.Join(r, relPath)
			if ok, _ := fileExists(candidate); ok {
				return candidate, r, true
			}
		}
		for _, name := range names {
			if name == "" {
				continue
			}
			if found, ok := findFileByName(r, name, 10); ok {
				return found, r, true
			}
		}
	}
	return "", "", false
}

func findFileByName(root, basename string, maxDepth int) (string, bool) {
	if maxDepth < 1 {
		maxDepth = 1
	}
	basename = strings.TrimSpace(basename)
	if basename == "" {
		return "", false
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			rel, relErr := filepath.Rel(root, path)
			if relErr == nil && rel != "." {
				depth := strings.Count(rel, string(os.PathSeparator)) + 1
				if depth > maxDepth {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.EqualFold(d.Name(), basename) {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	return found, found != ""
}

func (e *Engine) CheckTrackDuplicates(opts RunOptions, preview apple.PreviewResult, youtubeMetas []YouTubeDownloadMeta) DuplicateCheckResult {
	res := DuplicateCheckResult{}
	if len(preview.Tracks) == 0 {
		return res
	}
	selected := map[int]bool{}
	for _, n := range opts.SelectedTrackNums {
		selected[n] = true
	}
	if len(selected) == 0 {
		for _, t := range preview.Tracks {
			selected[t.Num] = true
		}
	}

	youtubeMode := useYouTubePipeline(opts)
	primary := preview.OutputFolder
	if primary == "" {
		if youtubeMode {
			primary = youtube.OutputDir(Config)
		} else {
			primary = outputFolderForQuality(opts.Quality)
		}
	}
	res.Roots = DuplicateCheckRoots(primary)

	_ = e.applyOptions(opts)

	metaByNum := map[int]youtube.DownloadMeta{}
	for _, m := range youtubeMetas {
		metaByNum[m.Num] = youtube.DownloadMeta(m)
	}

	for _, t := range preview.Tracks {
		if !selected[t.Num] {
			continue
		}
		res.SelectedCount++
		st := TrackDuplicateStatus{Num: t.Num}
		var relPath, basename string
		var extra []string

		if youtubeMode {
			meta := metaByNum[t.Num]
			if meta.Title == "" {
				meta = youtube.DownloadMeta{
					Num:         t.Num,
					Title:       t.Name,
					Artist:      t.Artist,
					Album:       t.Album,
					AlbumArtist: t.AlbumArtist,
					TrackNumber: t.TrackNumber,
					DiscNumber:  t.DiscNumber,
				}
				if meta.Album == "" {
					meta.Album = preview.Title
				}
				if meta.AlbumArtist == "" {
					meta.AlbumArtist = preview.Subtitle
				}
				if meta.TrackNumber <= 0 {
					meta.TrackNumber = t.Num
				}
			}
			multi := preview.TrackCount > 1
			delivery := NormalizeYouTubeDelivery(opts.YouTubeDeliveryMode)
			if delivery.SaveAudio() {
				expected := youtube.OutputPath(primary, meta, multi, false)
				st.ExpectedPath = expected
				st.ExpectedFilename = filepath.Base(expected)
				relPath, _ = filepath.Rel(primary, expected)
				basename = st.ExpectedFilename
				if delivery.SaveVideo() {
					extra = append(extra, filepath.Base(youtube.OutputPath(primary, meta, multi, true)))
				}
			} else if delivery.SaveVideo() {
				vid := youtube.OutputPath(primary, meta, multi, true)
				st.ExpectedPath = vid
				st.ExpectedFilename = filepath.Base(vid)
				relPath, _ = filepath.Rel(primary, vid)
				basename = st.ExpectedFilename
			}
		} else {
			plan := planAppleTrackPath(preview.Type, preview, t, opts.Quality)
			st.ExpectedPath = filepath.Join(plan.PrimaryRoot, plan.RelPath)
			st.ExpectedFilename = plan.Basename
			relPath = plan.RelPath
			basename = plan.Basename
			extra = plan.AltBasenames
		}

		if path, root, ok := LocateExistingMedia(res.Roots, relPath, basename, extra...); ok {
			st.OnDisk = true
			st.ExistingPath = path
			st.ExistingRoot = rootLabel(root, primary)
			res.ExistingCount++
		}
		res.Tracks = append(res.Tracks, st)
	}
	return res
}

type applePathPlan struct {
	PrimaryRoot  string
	RelPath      string
	Basename     string
	AltBasenames []string
}

func planAppleTrackPath(previewType string, preview apple.PreviewResult, track apple.PreviewTrack, quality string) applePathPlan {
	root := outputFolderForQuality(quality)
	if track.IsMV {
		root = Config.MVSaveFolder
	}
	artist := track.Artist
	if artist == "" {
		artist = preview.Subtitle
	}
	album := track.Album
	if album == "" {
		album = preview.Title
	}

	artistDir := buildArtistDirName(artist)
	var containerDir string
	switch previewType {
	case "Playlist":
		containerDir = buildPlaylistDirName(preview.Title, artist, quality)
		if containerDir == "" {
			containerDir = buildAlbumDirName(album, artist, quality)
		}
	default:
		containerDir = buildAlbumDirName(album, artist, quality)
	}

	basename := buildPreviewSongFilename(track, quality)
	ext := ".m4a"
	if track.IsMV {
		ext = ".mp4"
		basename = strings.TrimSuffix(basename, ".m4a") + ext
		if !strings.HasSuffix(strings.ToLower(basename), ".mp4") {
			basename = forbiddenNames.ReplaceAllString(track.Name, "_") + ext
		}
	}

	rel := filepath.Join(artistDir, containerDir, basename)
	return applePathPlan{
		PrimaryRoot: root,
		RelPath:     rel,
		Basename:    basename,
		AltBasenames: []string{
			forbiddenNames.ReplaceAllString(fmtTrackFallbackName(track), "_") + ext,
		},
	}
}

func buildArtistDirName(artist string) string {
	name := strings.NewReplacer(
		"{ArtistName}", LimitString(artist),
		"{UrlArtistName}", LimitString(artist),
		"{ArtistId}", "",
	).Replace(Config.ArtistFolderFormat)
	name = strings.TrimSpace(strings.TrimSuffix(name, "."))
	if name == "" {
		name = LimitString(artist)
	}
	return forbiddenNames.ReplaceAllString(name, "_")
}

func buildAlbumDirName(album, artist, quality string) string {
	name := strings.NewReplacer(
		"{AlbumName}", LimitString(album),
		"{ArtistName}", LimitString(artist),
		"{Quality}", qualityLabelForCheck(quality),
		"{Codec}", codecForQuality(quality),
	).Replace(Config.AlbumFolderFormat)
	name = strings.TrimSpace(strings.TrimSuffix(name, "."))
	if name == "" {
		name = LimitString(album)
	}
	return forbiddenNames.ReplaceAllString(name, "_")
}

func buildPlaylistDirName(playlist, artist, quality string) string {
	name := strings.NewReplacer(
		"{PlaylistName}", LimitString(playlist),
		"{ArtistName}", LimitString(artist),
		"{Quality}", qualityLabelForCheck(quality),
	).Replace(Config.PlaylistFolderFormat)
	name = strings.TrimSpace(strings.TrimSuffix(name, "."))
	if name == "" {
		name = LimitString(playlist)
	}
	return forbiddenNames.ReplaceAllString(name, "_")
}

func qualityLabelForCheck(quality string) string {
	switch quality {
	case "atmos":
		return "256Kbps"
	case "aac", "youtube":
		return "256Kbps"
	default:
		return "Lossless"
	}
}

func codecForQuality(quality string) string {
	switch quality {
	case "atmos":
		return "ATMOS"
	case "aac", "youtube":
		return "AAC"
	default:
		return "ALAC"
	}
}

func buildPreviewSongFilename(track apple.PreviewTrack, quality string) string {
	fileNum := track.TrackNumber
	if fileNum <= 0 {
		fileNum = track.Num
	}
	tagString := ""
	if track.Explicit && Config.ExplicitChoice != "" {
		tagString = Config.ExplicitChoice
	}
	songName := strings.NewReplacer(
		"{SongId}", track.ID,
		"{SongNumer}", fmt.Sprintf("%02d", fileNum),
		"{ArtistName}", LimitString(track.Artist),
		"{SongName}", LimitString(track.Name),
		"{DiscNumber}", fmt.Sprintf("%d", track.DiscNumber),
		"{TrackNumber}", fmt.Sprintf("%02d", fileNum),
		"{Quality}", qualityLabelForCheck(quality),
		"{Tag}", tagString,
		"{Codec}", codecForQuality(quality),
	).Replace(Config.SongFileFormat)
	songName = strings.TrimSpace(songName)
	if songName == "" {
		songName = fmtTrackFallbackName(track)
	}
	base := forbiddenNames.ReplaceAllString(songName, "_")
	if track.IsMV {
		return base + ".mp4"
	}
	return base + ".m4a"
}

func existingConvertedLocation(convertedPath string) (path, label string, ok bool) {
	primary := Config.AlacSaveFolder
	if dl_atmos {
		primary = Config.AtmosSaveFolder
	} else if dl_aac {
		primary = Config.AacSaveFolder
	}
	roots := DuplicateCheckRoots(primary)
	basename := filepath.Base(convertedPath)
	rel, err := filepath.Rel(primary, convertedPath)
	if err != nil {
		rel = basename
	}
	path, root, ok := LocateExistingMedia(roots, rel, basename)
	if !ok {
		return "", "", false
	}
	return path, rootLabel(root, primary), true
}

func youtubeExistingLocation(primary string, meta youtube.DownloadMeta, multiTrack, video bool) (path, label string, ok bool) {
	expected := youtube.OutputPath(primary, meta, multiTrack, video)
	rel, err := filepath.Rel(primary, expected)
	if err != nil {
		rel = filepath.Base(expected)
	}
	path, root, ok := LocateExistingMedia(DuplicateCheckRoots(primary), rel, filepath.Base(expected))
	if !ok {
		return "", "", false
	}
	return path, rootLabel(root, primary), true
}

func filterMissingYouTubeTracks(saveDir string, selectedNums []int, multiTrack bool, metaMap map[int]youtube.DownloadMeta, delivery YouTubeDeliveryMode) []int {
	if len(selectedNums) == 0 {
		return selectedNums
	}
	checkVideo := delivery.ExistingMediaIsVideo()
	out := make([]int, 0, len(selectedNums))
	for _, num := range selectedNums {
		meta := metaMap[num]
		if meta.Num == 0 {
			meta.Num = num
		}
		if _, _, exists := youtubeExistingLocation(saveDir, meta, multiTrack, checkVideo); exists {
			continue
		}
		out = append(out, num)
	}
	return out
}

func existingTrackLocation(trackPath, basename string) (path, label string, ok bool) {
	primary := Config.AlacSaveFolder
	if dl_atmos {
		primary = Config.AtmosSaveFolder
	} else if dl_aac {
		primary = Config.AacSaveFolder
	}
	roots := DuplicateCheckRoots(primary)
	rel, err := filepath.Rel(primary, trackPath)
	if err != nil {
		rel = basename
	}
	path, root, ok := LocateExistingMedia(roots, rel, basename)
	if !ok {
		return "", "", false
	}
	return path, rootLabel(root, primary), true
}

func fmtTrackFallbackName(track apple.PreviewTrack) string {
	n := track.TrackNumber
	if n <= 0 {
		n = track.Num
	}
	return fmt.Sprintf("%02d. %s", n, LimitString(track.Name))
}
