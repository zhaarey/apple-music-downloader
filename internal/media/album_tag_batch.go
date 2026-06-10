package media

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TagAlbumTrackInput is per-track metadata for batch album tagging.
type TagAlbumTrackInput struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	TrackNumber int16  `json:"track_number"`
	DiscNumber  int16  `json:"disc_number"`
}

// TagAlbumBatchInput applies shared album metadata to many tracks.
type TagAlbumBatchInput struct {
	Folder       string               `json:"folder"`
	Album        string               `json:"album"`
	AlbumArtist  string               `json:"album_artist"`
	Genre        string               `json:"genre"`
	Year         string               `json:"year"`
	TrackTotal   int16                `json:"track_total"`
	DiscTotal    int16                `json:"disc_total"`
	CoverPath         string               `json:"cover_path"`
	ClearArtwork      bool                 `json:"clear_artwork"`
	SortTags          bool                 `json:"sort_tags"`
	OptimizeArtwork   *bool                `json:"optimize_artwork"`
	WriteCoverSidecar *bool                `json:"write_cover_sidecar"`
	Mp4boxReembed     bool                 `json:"mp4box_reembed"`
	Tracks            []TagAlbumTrackInput `json:"tracks"`
}

// TagAlbumBatchResult reports batch write outcome.
type TagAlbumBatchResult struct {
	Saved   int      `json:"saved"`
	Errors  []string `json:"errors"`
	Summary string   `json:"summary"`
}

// ListAlbumTagFiles returns .m4a/.m4b paths in a folder (direct files first, else recursive).
func ListAlbumTagFiles(folder string) ([]string, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return nil, fmt.Errorf("no folder path")
	}
	stat, err := os.Stat(folder)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a folder")
	}

	direct := []string{}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(ent.Name()))
		if ext == ".m4a" || ext == ".m4b" {
			direct = append(direct, filepath.Join(folder, ent.Name()))
		}
	}
	if len(direct) > 0 {
		sort.Strings(direct)
		return direct, nil
	}
	return CollectAlbumTracks(folder)
}

// ListDirectAlbumTagFiles returns .m4a/.m4b files immediately inside folder (not subfolders).
func ListDirectAlbumTagFiles(folder string) ([]string, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return nil, fmt.Errorf("no folder path")
	}
	stat, err := os.Stat(folder)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a folder")
	}
	out := []string{}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(ent.Name()))
		if ext == ".m4a" || ext == ".m4b" {
			out = append(out, filepath.Join(folder, ent.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

func albumTagPaths(root string, recursive bool) ([]string, error) {
	if recursive {
		return CollectAlbumTracks(root)
	}
	return ListDirectAlbumTagFiles(root)
}

// PreviewPrepareAlbum reports how many tracks would be modified and which cover would be used.
func PreviewPrepareAlbum(root string, recursive bool) (AlbumPreparePreview, error) {
	out := AlbumPreparePreview{Folder: root, Recursive: recursive}
	paths, err := albumTagPaths(root, recursive)
	if err != nil {
		return out, err
	}
	out.TrackCount = len(paths)
	out.CoverSource = describePrepareCoverSource(root, paths)
	out.Warning = "Only embedded artwork is updated — title, artist, and track numbers stay as they are. Tracks that already match are skipped."
	if recursive {
		out.Warning += " Includes all .m4a files in subfolders."
	}
	return out, nil
}

func describePrepareCoverSource(dir string, tracks []string) string {
	if len(tracks) > 0 {
		hashCounts := map[string]int{}
		for _, p := range tracks {
			if h, err := EmbeddedCoverHash(p); err == nil {
				hashCounts[h]++
			}
		}
		if len(hashCounts) == 1 {
			return "embedded art (all tracks already match)"
		}
	}
	if sidecar := FindAlbumCoverFile(dir); sidecar != "" {
		return filepath.Base(sidecar) + " (folder sidecar — used when tracks differ)"
	}
	for _, p := range tracks {
		info, err := ReadAudioTags(p)
		if err == nil && info.HasArtwork {
			return "embedded art from " + filepath.Base(p)
		}
	}
	return "none found — add cover.jpg or embed art before prepare"
}

// ReadAlbumTags reads metadata for every audio file in an album folder.
func ReadAlbumTags(folder string) ([]AudioTagInfo, error) {
	paths, err := ListAlbumTagFiles(folder)
	if err != nil {
		return nil, err
	}
	out := make([]AudioTagInfo, 0, len(paths))
	for _, p := range paths {
		info, err := ReadAudioTags(p)
		if err != nil {
			return out, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, info)
	}
	sortAlbumTagInfos(out)
	return out, nil
}

func sortAlbumTagInfos(infos []AudioTagInfo) {
	sort.SliceStable(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		if a.DiscNumber != b.DiscNumber && a.DiscNumber > 0 && b.DiscNumber > 0 {
			return a.DiscNumber < b.DiscNumber
		}
		if a.TrackNumber != b.TrackNumber && a.TrackNumber > 0 && b.TrackNumber > 0 {
			return a.TrackNumber < b.TrackNumber
		}
		return strings.ToLower(filepath.Base(a.Path)) < strings.ToLower(filepath.Base(b.Path))
	})
}

// WriteAlbumBatch writes shared album metadata and per-track fields to many files.
func WriteAlbumBatch(input TagAlbumBatchInput) TagAlbumBatchResult {
	out := TagAlbumBatchResult{}
	if len(input.Tracks) == 0 {
		out.Summary = "No tracks to save."
		return out
	}
	trackTotal := input.TrackTotal
	if trackTotal <= 0 {
		trackTotal = int16(len(input.Tracks))
	}
	discTotal := input.DiscTotal
	if discTotal <= 0 {
		discTotal = 1
	}
	sortTags := input.SortTags
	if !sortTags {
		sortTags = true
	}
	coverPath := strings.TrimSpace(input.CoverPath)
	optimize := boolOrDefault(input.OptimizeArtwork, true)
	writeSidecar := boolOrDefault(input.WriteCoverSidecar, true)

	for _, tr := range input.Tracks {
		if strings.TrimSpace(tr.Path) == "" {
			continue
		}
		trackNum := tr.TrackNumber
		if trackNum <= 0 {
			trackNum = 1
		}
		discNum := tr.DiscNumber
		if discNum <= 0 {
			discNum = 1
		}
		artist := strings.TrimSpace(tr.Artist)
		if artist == "" {
			artist = strings.TrimSpace(input.AlbumArtist)
		}
		writeIn := WriteAudioTagsInput{
			Path:              tr.Path,
			Title:             tr.Title,
			Artist:            artist,
			Album:             input.Album,
			AlbumArtist:       input.AlbumArtist,
			Genre:             input.Genre,
			Year:              input.Year,
			TrackNumber:       trackNum,
			TrackTotal:        trackTotal,
			DiscNumber:        discNum,
			DiscTotal:         discTotal,
			CoverPath:         coverPath,
			ClearArtwork:      input.ClearArtwork,
			SortTags:          sortTags,
			OptimizeArtwork:   &optimize,
			WriteCoverSidecar: boolPtr(false),
			Mp4boxReembed:     input.Mp4boxReembed,
		}
		if err := WriteAudioTags(writeIn); err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", tr.Path, err))
			continue
		}
		out.Saved++
	}
	if coverPath != "" && writeSidecar && out.Saved > 0 {
		folder := strings.TrimSpace(input.Folder)
		if folder == "" && len(input.Tracks) > 0 {
			folder = filepath.Dir(input.Tracks[0].Path)
		}
		if folder != "" {
			if data, err := os.ReadFile(coverPath); err == nil {
				opts := DefaultCoverNormalizeOptions()
				if !optimize {
					opts = LegacyCoverNormalizeOptions()
				}
				if norm, err := NormalizeCoverWithOptions(data, opts); err == nil {
					_, _ = WriteCoverSidecarForDir(folder, norm)
				}
			}
		}
	}
	if len(out.Errors) == 0 {
		out.Summary = fmt.Sprintf("Saved tags on %d track(s).", out.Saved)
	} else {
		out.Summary = fmt.Sprintf("Saved %d track(s) with %d error(s).", out.Saved, len(out.Errors))
	}
	return out
}

func boolPtr(v bool) *bool {
	return &v
}
