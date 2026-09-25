package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"amdl/internal/model"

	mp4tag "github.com/itouakirai/go-mp4tag"
)

const (
	defaultClassicalFolderFormat = "{Composer}/{AlbumName} ({ReleaseYear}) [{Artists}] [{AlbumId}]/{WorkTitle} [{RecordingId}]"
	defaultClassicalFileFormat   = "{DiscNumber}-{TrackNumber} - {MovementTitle}"

	// classicalIdentityTag is a freeform tag that marks which Recording and
	// song a file belongs to, so repeated runs can reuse finished files.
	classicalIdentityTag = "AMDL_CLASSICAL_RECORDING"
)

var classicalPlaceholder = regexp.MustCompile(`\{[^{}]*\}`)

// perTrackPlaceholders differ between the tracks of one Recording. The folder
// is shared by all of them, so these belong in the file name only.
var perTrackPlaceholders = []string{"{MovementTitle}", "{SongId}", "{DiscNumber}", "{TrackNumber}", "{MovementIndex}"}

func classicalIdentity(track *model.Track) string {
	return track.Classical.RecordingID + "/" + track.ID
}

func classicalValues(track *model.Track) map[string]string {
	album := track.AlbumData.Attributes
	c := track.Classical
	return map[string]string{
		"{Composer}":      c.Composer,
		"{WorkTitle}":     c.WorkTitle,
		"{MovementTitle}": c.MovementTitle,
		"{RecordingId}":   c.RecordingID,
		"{Conductor}":     c.Conductor,
		"{AlbumName}":     album.Name,
		"{AlbumId}":       track.AlbumData.ID,
		"{Artists}":       album.ArtistName,
		"{ReleaseYear}":   releaseYear(album.ReleaseDate),
		"{SongId}":        track.ID,
		"{DiscNumber}":    strconv.Itoa(track.Resp.Attributes.DiscNumber),
		"{TrackNumber}":   strconv.Itoa(track.Resp.Attributes.TrackNumber),
		"{MovementIndex}": strconv.Itoa(c.Position),
		"{MovementCount}": strconv.Itoa(c.Count),
	}
}

// fillClassicalSegment replaces placeholders in one path component. Values
// are sanitized individually so a "/" inside a title never adds a directory.
func (r *Runner) fillClassicalSegment(segment string, values map[string]string) (string, error) {
	var unknown []string
	out := classicalPlaceholder.ReplaceAllStringFunc(segment, func(p string) string {
		v, ok := values[p]
		if !ok {
			unknown = append(unknown, p)
			return p
		}
		return r.LimitString(v)
	})
	if len(unknown) > 0 {
		return "", fmt.Errorf("unknown classical placeholder %s", strings.Join(unknown, ", "))
	}
	out = strings.TrimSpace(sanitizeFolderName(strings.TrimSpace(out)))
	if out == "" {
		return "", fmt.Errorf("classical template segment %q is empty", segment)
	}
	return out, nil
}

func (r *Runner) classicalFolder(root string, track *model.Track) (string, error) {
	format := r.Config.ClassicalFolderFormat
	if format == "" {
		format = defaultClassicalFolderFormat
	}
	for _, p := range perTrackPlaceholders {
		if strings.Contains(format, p) {
			return "", fmt.Errorf("%s differs per track; use it in classical-file-format", p)
		}
	}
	values := classicalValues(track)
	parts := []string{root}
	for _, segment := range strings.Split(format, "/") {
		part, err := r.fillClassicalSegment(segment, values)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	return filepath.Join(parts...), nil
}

// classicalFileName returns the file name without extension.
func (r *Runner) classicalFileName(track *model.Track) (string, error) {
	format := r.Config.ClassicalFileFormat
	if format == "" {
		format = defaultClassicalFileFormat
	}
	if strings.ContainsAny(format, `/\`) {
		return "", errors.New("classical file format must not contain a directory separator")
	}
	return r.fillClassicalSegment(format, classicalValues(track))
}

// checkClassicalExisting reports nil when the file at path was written for
// the same Recording and song, and an error for any other file.
func checkClassicalExisting(path string, track *model.Track) error {
	m, err := mp4tag.Open(path)
	if err != nil {
		return fmt.Errorf("open existing file: %w", err)
	}
	defer m.Close()
	tags, err := m.Read()
	if err != nil {
		return fmt.Errorf("read existing file tags: %w", err)
	}
	got := tags.Custom[classicalIdentityTag]
	if got == "" {
		return fmt.Errorf("existing file %s has no %s tag", path, classicalIdentityTag)
	}
	if want := classicalIdentity(track); got != want {
		return fmt.Errorf("existing file %s belongs to %s, not %s", path, got, want)
	}
	return nil
}
