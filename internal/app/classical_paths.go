package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"amdl/internal/model"

	mp4tag "github.com/itouakirai/go-mp4tag"
)

const (
	defaultClassicalFolderFormat = "{Composer}/{AlbumName} ({ReleaseYear}) [{Artists}] [{AlbumId}]/{WorkTitle}"
	defaultClassicalFileFormat   = "{DiscNumber}-{TrackNumber} - {MovementTitle}"

	// classicalIdentityTag is a freeform tag that marks which Recording and
	// song a file belongs to, so repeated runs can reuse finished files.
	classicalIdentityTag = "AMDL_CLASSICAL_RECORDING"

	// maxFolderName is the longest path component common file systems accept.
	maxFolderName = 255
	// maxFileName leaves room for the extension and the longest suffix added
	// to a track's name: go-mp4tag writes tags through a copy named
	// "<name>.m4a_tmp_<13-digit milliseconds>". Downloads use ".m4a.part" and
	// ".m4a.tmp-<up to 10 digits>".
	maxFileName = maxFolderName - 22
)

var classicalPlaceholder = regexp.MustCompile(`\{[^{}]*\}`)

// windowsDeviceName matches names Windows reserves for devices, which it
// refuses as file names even with an extension. It follows Go's own list,
// which counts superscript digits and the console names.
var windowsDeviceName = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[1-9¹²³]|LPT[1-9¹²³]|CONIN\$|CONOUT\$) *$`)

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
		"{Conductor}":     c.RecordingConductor,
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
// The result is at most maxLen long (see nameLength).
func (r *Runner) fillClassicalSegment(segment string, values map[string]string, maxLen int) (string, error) {
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
	out = strings.Map(func(c rune) rune {
		if unicode.IsControl(c) {
			return '_'
		}
		return c
	}, forbiddenNames.ReplaceAllString(out, "_"))
	// Windows drops trailing dots and spaces, so "..." would name the parent.
	out = strings.TrimRightFunc(strings.TrimSpace(out), func(c rune) bool {
		return c == '.' || unicode.IsSpace(c)
	})
	if out == "" {
		return "", fmt.Errorf("classical template segment %q gives an empty name", segment)
	}
	if base, _, _ := strings.Cut(out, "."); windowsDeviceName.MatchString(base) {
		out = "_" + out
	}
	if n := nameLength(out); n > maxLen {
		return "", fmt.Errorf("classical name is %d long, over the limit of %d; shorten the template or lower limit-max: %s", n, maxLen, out)
	}
	return out, nil
}

// nameLength measures a path component the way file systems limit it:
// UTF-16 units on Windows, bytes elsewhere.
func nameLength(name string) int {
	if runtime.GOOS == "windows" {
		return len(utf16.Encode([]rune(name)))
	}
	return len(name)
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
		part, err := r.fillClassicalSegment(segment, values, maxFolderName)
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
	return r.fillClassicalSegment(format, classicalValues(track), maxFileName)
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
