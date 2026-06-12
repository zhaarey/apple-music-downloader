package media

import (
	"strings"

	"github.com/zhaarey/go-mp4tag"
)

// BuildAppleMusicTags maps TrackTags to mp4tag fields for Apple Music / iPhone sync.
func BuildAppleMusicTags(tags TrackTags) (*mp4tag.MP4Tags, error) {
	title := strings.TrimSpace(tags.Title)
	artist := strings.TrimSpace(tags.Artist)
	album := strings.TrimSpace(tags.Album)
	if title == "" {
		title = "Unknown Title"
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = title
	}
	albumArtist := strings.TrimSpace(tags.AlbumArtist)
	if albumArtist == "" {
		albumArtist = artist
	}
	trackNum := tags.TrackNumber
	if trackNum <= 0 {
		trackNum = 1
	}
	discNum := tags.DiscNumber
	if discNum <= 0 {
		discNum = 1
	}
	trackTotal := tags.TrackTotal
	if trackTotal <= 0 {
		trackTotal = trackNum
	}
	discTotal := tags.DiscTotal
	if discTotal <= 0 {
		discTotal = 1
	}

	custom := map[string]string{"PERFORMER": artist}
	for k, v := range tags.CustomMeta {
		if strings.TrimSpace(v) != "" {
			custom[k] = v
		}
	}

	t := &mp4tag.MP4Tags{
		Title:       title,
		Artist:      artist,
		Album:       album,
		AlbumArtist: albumArtist,
		CustomGenre: strings.TrimSpace(tags.Genre),
		TrackNumber: trackNum,
		TrackTotal:  trackTotal,
		DiscNumber:  discNum,
		DiscTotal:   discTotal,
		Date:        strings.TrimSpace(tags.Year),
		Lyrics:      tags.Lyrics,
		Custom:      custom,
	}
	if tags.IsCompilation {
		if t.Custom == nil {
			t.Custom = map[string]string{}
		}
		t.Custom["cpil"] = "1"
	}
	if tags.SortTags {
		t.TitleSort = title
		t.ArtistSort = artist
		t.AlbumSort = album
		t.AlbumArtistSort = albumArtist
	}
	switch strings.ToLower(strings.TrimSpace(tags.ContentRating)) {
	case "explicit":
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryExplicit
	case "clean":
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryClean
	default:
		if tags.ContentRating != "" {
			t.ItunesAdvisory = mp4tag.ItunesAdvisoryNone
		}
	}
	if tags.ItunesAlbumID > 0 {
		t.ItunesAlbumID = tags.ItunesAlbumID
	}
	if tags.ItunesArtistID > 0 {
		t.ItunesArtistID = tags.ItunesArtistID
	}
	if tags.Copyright != "" {
		t.Copyright = tags.Copyright
	}
	if tags.Publisher != "" {
		t.Publisher = tags.Publisher
	}
	if tags.Composer != "" {
		t.Composer = tags.Composer
		if tags.SortTags {
			t.ComposerSort = tags.Composer
		}
	}
	return t, nil
}
