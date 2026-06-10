package engine

import (
	"fmt"
	"strconv"
	"strings"

	"main/internal/media"
	"main/utils/ampapi"
	"main/utils/task"
)

func trackToMediaTags(track *task.Track, lrc string) media.TrackTags {
	tags := media.TrackTags{
		Title:       track.Resp.Attributes.Name,
		Artist:      track.Resp.Attributes.ArtistName,
		Album:       track.Resp.Attributes.AlbumName,
		Genre:       firstGenre(track.Resp.Attributes.GenreNames),
		Year:        releaseYear(track.AlbumData.Attributes.ReleaseDate),
		TrackNumber: int16(track.Resp.Attributes.TrackNumber),
		DiscNumber:  int16(track.Resp.Attributes.DiscNumber),
		Lyrics:      lrc,
		Composer:    track.Resp.Attributes.ComposerName,
		SortTags:    Config.TagSortOrder,
		CustomMeta: map[string]string{
			"PERFORMER":   track.Resp.Attributes.ArtistName,
			"RELEASETIME": track.Resp.Attributes.ReleaseDate,
			"ISRC":        track.Resp.Attributes.Isrc,
		},
		RequireCover: Config.EmbedCover,
	}
	switch track.Resp.Attributes.ContentRating {
	case "explicit":
		tags.ContentRating = "explicit"
	case "clean":
		tags.ContentRating = "clean"
	}

	if (track.PreType == "playlists" || track.PreType == "stations") && !Config.UseSongInfoForPlaylist {
		tags.DiscNumber = 1
		tags.DiscTotal = 1
		tags.TrackNumber = int16(track.TaskNum)
		tags.TrackTotal = int16(track.TaskTotal)
		tags.Album = track.PlaylistData.Attributes.Name
		tags.AlbumArtist = track.PlaylistData.Attributes.ArtistName
	} else if (track.PreType == "playlists" || track.PreType == "stations") && Config.UseSongInfoForPlaylist {
		tags.DiscTotal = int16(track.DiscTotal)
		tags.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
		tags.AlbumArtist = track.AlbumData.Attributes.ArtistName
		tags.Year = releaseYear(track.AlbumData.Attributes.ReleaseDate)
		tags.Copyright = track.AlbumData.Attributes.Copyright
		tags.Publisher = track.AlbumData.Attributes.RecordLabel
		tags.CustomMeta["UPC"] = track.AlbumData.Attributes.Upc
		tags.CustomMeta["LABEL"] = track.AlbumData.Attributes.RecordLabel
	} else {
		tags.DiscTotal = int16(track.DiscTotal)
		tags.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
		tags.AlbumArtist = track.AlbumData.Attributes.ArtistName
		tags.Year = releaseYear(track.AlbumData.Attributes.ReleaseDate)
		tags.Copyright = track.AlbumData.Attributes.Copyright
		tags.Publisher = track.AlbumData.Attributes.RecordLabel
		tags.CustomMeta["UPC"] = track.AlbumData.Attributes.Upc
		tags.CustomMeta["LABEL"] = track.AlbumData.Attributes.RecordLabel
		if track.AlbumData.Attributes.IsCompilation {
			tags.IsCompilation = true
		}
	}

	if Config.TagItunesID {
		if track.PreType == "albums" {
			if albumID, err := strconv.ParseUint(track.PreID, 10, 64); err == nil {
				tags.ItunesAlbumID = int32(albumID)
			}
		}
		if len(track.Resp.Relationships.Artists.Data) > 0 {
			if artistID, err := strconv.ParseUint(track.Resp.Relationships.Artists.Data[0].ID, 10, 64); err == nil {
				tags.ItunesArtistID = int32(artistID)
			}
		}
	}

	coverPath := track.CoverPath
	if coverPath == "" {
		if p, err := resolveCoverPath(track); err == nil {
			coverPath = p
		}
	}
	tags.CoverPath = coverPath
	return tags
}

func writeMP4Tags(track *task.Track, lrc string) error {
	return writeMP4TagsForDownload(track, lrc)
}

// writeMP4TagsForDownload embeds Apple Music–ready metadata and normalized JPEG artwork for iPhone sync.
func writeMP4TagsForDownload(track *task.Track, lrc string) error {
	tags := trackToMediaTags(track, lrc)
	if Config.EmbedCover {
		tags.RequireCover = true
		if strings.TrimSpace(tags.CoverPath) == "" {
			if err := prepareDownloadArtwork(track); err != nil {
				return fmt.Errorf("embed cover enabled but no artwork available for %q: %w", track.Resp.Attributes.Name, err)
			}
		}
		tags.CoverPath = track.CoverPath
	}
	if strings.TrimSpace(tags.AlbumArtist) == "" {
		tags.AlbumArtist = tags.Artist
	}
	if tags.TrackTotal <= 0 {
		tags.TrackTotal = tags.TrackNumber
		if tags.TrackTotal <= 0 {
			tags.TrackTotal = 1
		}
	}
	if tags.TrackNumber <= 0 {
		tags.TrackNumber = 1
	}
	if tags.DiscTotal <= 0 {
		tags.DiscTotal = 1
	}
	if tags.DiscNumber <= 0 {
		tags.DiscNumber = 1
	}
	return media.WriteTrackTags(track.SavePath, tags)
}

func prepareDownloadArtwork(track *task.Track) error {
	if track.CoverPath != "" {
		if ok, err := fileExists(track.CoverPath); err == nil && ok {
			// fall through to sidecar
		} else {
			track.CoverPath = ""
		}
	}
	if track.CoverPath == "" {
		p, err := resolveCoverPath(track)
		if err != nil {
			return err
		}
		track.CoverPath = p
	}
	if track.SaveDir == "" {
		return nil
	}
	sidecar, err := media.WriteNormalizedCoverSidecar(track.SaveDir, track.CoverPath)
	if err != nil {
		return err
	}
	track.CoverPath = sidecar
	return nil
}

func writeStationTrackTags(path, name, coverPath string) error {
	tags := media.TrackTags{
		Title:        name,
		Artist:       "Apple Music Station",
		Album:        name,
		AlbumArtist:  "Apple Music Station",
		TrackNumber:  1,
		TrackTotal:   1,
		DiscNumber:   1,
		DiscTotal:    1,
		SortTags:     Config.TagSortOrder,
		CoverPath:    coverPath,
		RequireCover: Config.EmbedCover,
		CustomMeta: map[string]string{
			"PERFORMER": "Apple Music Station",
		},
	}
	return media.WriteTrackTags(path, tags)
}

func writeMVTrackTags(mvPath string, track *task.Track, mvInfo *ampapi.MusicVideoResp, coverPath string) error {
	if !Config.EmbedCover || strings.TrimSpace(coverPath) == "" {
		return nil
	}
	attrs := mvInfo.Data[0].Attributes
	tags := media.TrackTags{
		Title:       attrs.Name,
		Artist:      attrs.ArtistName,
		Album:       attrs.AlbumName,
		TrackNumber: int16(attrs.TrackNumber),
		DiscNumber:  int16(attrs.DiscNumber),
		SortTags:    Config.TagSortOrder,
		CoverPath:   coverPath,
		RequireCover: true,
		CustomMeta: map[string]string{
			"PERFORMER": attrs.ArtistName,
		},
	}
	switch attrs.ContentRating {
	case "explicit":
		tags.ContentRating = "explicit"
	case "clean":
		tags.ContentRating = "clean"
	}
	if track != nil {
		if track.PreType == "playlists" && !Config.UseSongInfoForPlaylist {
			tags.Album = track.PlaylistData.Attributes.Name
			tags.AlbumArtist = track.PlaylistData.Attributes.ArtistName
			tags.TrackNumber = int16(track.TaskNum)
			tags.TrackTotal = int16(track.TaskTotal)
			tags.DiscNumber = 1
			tags.DiscTotal = 1
		} else if track.PreType == "playlists" && Config.UseSongInfoForPlaylist {
			tags.Album = track.AlbumData.Attributes.Name
			tags.AlbumArtist = track.AlbumData.Attributes.ArtistName
			tags.TrackNumber = int16(track.Resp.Attributes.TrackNumber)
			tags.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
			tags.DiscNumber = int16(track.Resp.Attributes.DiscNumber)
			tags.DiscTotal = int16(track.DiscTotal)
			tags.Copyright = track.AlbumData.Attributes.Copyright
			tags.CustomMeta["UPC"] = track.AlbumData.Attributes.Upc
		} else if track != nil {
			tags.Album = track.AlbumData.Attributes.Name
			tags.AlbumArtist = track.AlbumData.Attributes.ArtistName
			tags.TrackNumber = int16(track.Resp.Attributes.TrackNumber)
			tags.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
			tags.DiscNumber = int16(track.Resp.Attributes.DiscNumber)
			tags.DiscTotal = int16(track.DiscTotal)
			tags.Copyright = track.AlbumData.Attributes.Copyright
			tags.CustomMeta["UPC"] = track.AlbumData.Attributes.Upc
			if track.AlbumData.Attributes.IsCompilation {
				tags.IsCompilation = true
			}
		}
	} else {
		tags.AlbumArtist = attrs.ArtistName
	}
	return media.WriteTrackTags(mvPath, tags)
}
