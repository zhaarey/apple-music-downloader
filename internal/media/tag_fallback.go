package media

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func isCorruptMP4TagErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "makeslice") ||
		strings.Contains(msg, "len out of range") ||
		strings.Contains(msg, "capacity out of range") ||
		strings.Contains(msg, "runtime error") ||
		strings.Contains(msg, "slice bounds out of range") ||
		strings.Contains(msg, "index out of range")
}

func isRecoverableMP4TagReadErr(err error) bool {
	return isMissingMetaBoxErr(err) || isCorruptMP4TagErr(err)
}

// readAudioTagsFallback builds best-effort metadata when go-mp4tag cannot parse the file.
func readAudioTagsFallback(ffmpegConfigured, path string, cause error) AudioTagInfo {
	out := AudioTagInfoFromPath(path)
	out.Path = path
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4":
		out.MediaKind = "video"
	default:
		out.MediaKind = "audio"
	}
	if tags, err := probeFormatTags(ffmpegConfigured, path); err == nil {
		applyProbedFormatTags(&out, tags)
	}
	if out.Title == "" {
		_, title := ParseTrackFilename(path)
		out.Title = title
	}
	title := out.Title
	if title == "" {
		title = "Unknown Title"
	}
	artist := out.Artist
	if artist == "" {
		artist = "Unknown Artist"
	}
	out.Summary = title + " · " + artist
	if out.Album != "" {
		out.Summary += " · " + out.Album
	}
	if cause != nil && isCorruptMP4TagErr(cause) {
		out.Summary += " · opened with limited tags (file metadata needs repair)"
		out.TagsPartial = true
		out.TagsWarning = "This M4A has damaged iTunes metadata atoms. You can still edit tags — use Save as new file if saving in place fails."
	} else if cause != nil && isMissingMetaBoxErr(cause) {
		out.Summary += " · no embedded tags yet"
	}
	return out
}

type probedFormatTags struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Year        string
	TrackNumber int16
	TrackTotal  int16
	DiscNumber  int16
	DiscTotal   int16
}

func probeFormatTags(ffmpegConfigured, path string) (probedFormatTags, error) {
	result, err := probeFile(ffmpegConfigured, path)
	if err != nil {
		return probedFormatTags{}, err
	}
	tags := result.Format.Tags
	if len(tags) == 0 {
		return probedFormatTags{}, fmt.Errorf("no format tags")
	}
	out := probedFormatTags{
		Title:       firstTag(tags, "title", "TITLE"),
		Artist:      firstTag(tags, "artist", "ARTIST", "Author"),
		Album:       firstTag(tags, "album", "ALBUM"),
		AlbumArtist: firstTag(tags, "album_artist", "ALBUM_ARTIST", "albumartist"),
		Genre:       firstTag(tags, "genre", "GENRE"),
		Year:        firstTag(tags, "date", "DATE", "year", "YEAR"),
	}
	if out.Year != "" && len(out.Year) >= 4 {
		out.Year = out.Year[:4]
	}
	out.TrackNumber, out.TrackTotal = parseNumberPair(firstTag(tags, "track", "TRACK", "tracknumber"))
	out.DiscNumber, out.DiscTotal = parseNumberPair(firstTag(tags, "disc", "DISC", "discnumber"))
	return out, nil
}

func firstTag(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(tags[key]); v != "" {
			return v
		}
		for k, v := range tags {
			if strings.EqualFold(k, key) {
				if t := strings.TrimSpace(v); t != "" {
					return t
				}
			}
		}
	}
	return ""
}

func parseNumberPair(raw string) (int16, int16) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0
	}
	parts := strings.SplitN(raw, "/", 2)
	n, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	total := 0
	if len(parts) > 1 {
		total, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	if n < 0 {
		n = 0
	}
	if total < 0 {
		total = 0
	}
	if n > 32767 {
		n = 32767
	}
	if total > 32767 {
		total = 32767
	}
	return int16(n), int16(total)
}

func applyProbedFormatTags(out *AudioTagInfo, tags probedFormatTags) {
	if tags.Title != "" {
		out.Title = tags.Title
	}
	if tags.Artist != "" {
		out.Artist = tags.Artist
	}
	if tags.Album != "" {
		out.Album = tags.Album
	}
	if tags.AlbumArtist != "" {
		out.AlbumArtist = tags.AlbumArtist
	}
	if tags.Genre != "" {
		out.Genre = tags.Genre
	}
	if tags.Year != "" {
		out.Year = tags.Year
	}
	if tags.TrackNumber > 0 {
		out.TrackNumber = tags.TrackNumber
	}
	if tags.TrackTotal > 0 {
		out.TrackTotal = tags.TrackTotal
	}
	if tags.DiscNumber > 0 {
		out.DiscNumber = tags.DiscNumber
	}
	if tags.DiscTotal > 0 {
		out.DiscTotal = tags.DiscTotal
	}
}
