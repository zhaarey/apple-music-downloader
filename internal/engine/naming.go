package engine

import (
	"fmt"
	"strings"

	"main/utils/task"
)

func buildTagSuffix(track *task.Track) string {
	var parts []string
	if track.Resp.Attributes.IsAppleDigitalMaster && Config.AppleMasterChoice != "" {
		parts = append(parts, Config.AppleMasterChoice)
	}
	switch track.Resp.Attributes.ContentRating {
	case "explicit":
		if Config.ExplicitChoice != "" {
			parts = append(parts, Config.ExplicitChoice)
		}
	case "clean":
		if Config.CleanChoice != "" {
			parts = append(parts, Config.CleanChoice)
		}
	}
	return strings.Join(parts, " ")
}

func buildSongFilename(track *task.Track, quality string) (filename, lrcFilename string) {
	tagString := buildTagSuffix(track)
	fileNum := trackFileNumber(track)
	songName := strings.NewReplacer(
		"{SongId}", track.ID,
		"{SongNumer}", fmt.Sprintf("%02d", fileNum),
		"{ArtistName}", LimitString(track.Resp.Attributes.ArtistName),
		"{SongName}", LimitString(track.Resp.Attributes.Name),
		"{DiscNumber}", fmt.Sprintf("%0d", track.Resp.Attributes.DiscNumber),
		"{TrackNumber}", fmt.Sprintf("%02d", fileNum),
		"{Quality}", quality,
		"{Tag}", tagString,
		"{Codec}", track.Codec,
	).Replace(Config.SongFileFormat)
	songName = strings.TrimSpace(songName)
	if songName == "" || strings.TrimSpace(strings.ReplaceAll(songName, "_", "")) == "" {
		songName = fmt.Sprintf("%02d. %s", fileNum, LimitString(track.Resp.Attributes.Name))
	}
	fmt.Println(songName)
	base := forbiddenNames.ReplaceAllString(songName, "_")
	return base + ".m4a", base + "." + Config.LrcFormat
}
