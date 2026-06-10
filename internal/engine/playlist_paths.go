package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"main/utils/ampapi"
	"main/utils/task"
)

type albumLocationCache map[string]struct {
	dir   string
	cover string
}

func saveRootForCurrentCodec() string {
	if dl_atmos {
		return Config.AtmosSaveFolder
	}
	if dl_aac {
		return Config.AacSaveFolder
	}
	return Config.AlacSaveFolder
}

func buildAlbumTagString(album ampapi.AlbumRespData) string {
	var parts []string
	if album.Attributes.IsAppleDigitalMaster || album.Attributes.IsMasteredForItunes {
		if Config.AppleMasterChoice != "" {
			parts = append(parts, Config.AppleMasterChoice)
		}
	}
	switch album.Attributes.ContentRating {
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

func buildArtistFolderNameFromAlbum(album ampapi.AlbumRespData) string {
	if Config.ArtistFolderFormat == "" {
		return ""
	}
	artistID := ""
	if len(album.Relationships.Artists.Data) > 0 {
		artistID = album.Relationships.Artists.Data[0].ID
	}
	name := strings.NewReplacer(
		"{UrlArtistName}", LimitString(album.Attributes.ArtistName),
		"{ArtistName}", LimitString(album.Attributes.ArtistName),
		"{ArtistId}", artistID,
	).Replace(Config.ArtistFolderFormat)
	name = strings.TrimSpace(strings.TrimSuffix(name, "."))
	return forbiddenNames.ReplaceAllString(name, "_")
}

func resolveQualityLabelForTrack(storefront, trackID, language, token, codec string) (string, string) {
	if dl_atmos {
		return codec, fmt.Sprintf("%dKbps", Config.AtmosMax-2000)
	}
	if useAacLCDownload() {
		return "AAC", "256Kbps"
	}
	if !strings.Contains(Config.AlbumFolderFormat, "Quality") {
		return codec, ""
	}
	manifest, err := ampapi.GetSongResp(storefront, trackID, language, token)
	if err != nil {
		return codec, ""
	}
	if manifest.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls == "" {
		return "AAC", "256Kbps"
	}
	needCheck := Config.GetM3u8Mode == "all" ||
		(Config.GetM3u8Mode == "hires" && contains(manifest.Data[0].Attributes.AudioTraits, "hi-res-lossless"))
	m3u8 := manifest.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls
	if needCheck {
		if enhanced, err := checkM3u8(trackID, "song"); err == nil && strings.HasSuffix(enhanced, ".m3u8") {
			m3u8 = enhanced
		}
	}
	_, quality, err := extractMedia(m3u8, true)
	if err != nil {
		return codec, ""
	}
	return codec, quality
}

func buildAlbumFolderNameFromAlbum(album ampapi.AlbumRespData, albumID, codec, quality string) string {
	name := strings.NewReplacer(
		"{ReleaseDate}", album.Attributes.ReleaseDate,
		"{ReleaseYear}", releaseYear(album.Attributes.ReleaseDate),
		"{ArtistName}", LimitString(album.Attributes.ArtistName),
		"{AlbumName}", LimitString(album.Attributes.Name),
		"{UPC}", album.Attributes.Upc,
		"{RecordLabel}", album.Attributes.RecordLabel,
		"{Copyright}", album.Attributes.Copyright,
		"{AlbumId}", albumID,
		"{Quality}", quality,
		"{Codec}", codec,
		"{Tag}", buildAlbumTagString(album),
	).Replace(Config.AlbumFolderFormat)
	name = strings.TrimSpace(strings.TrimSuffix(name, "."))
	return forbiddenNames.ReplaceAllString(name, "_")
}

func preparePlaylistTrackLocation(track *task.Track, token, storefront, language, codec string, cache albumLocationCache) error {
	if track.Type == "music-videos" {
		track.SaveDir = Config.MVSaveFolder
		track.Codec = codec
		return nil
	}
	if err := track.GetAlbumData(token); err != nil {
		return err
	}
	album := track.AlbumData
	albumID := album.ID
	if albumID == "" {
		return fmt.Errorf("missing album id for track %q", track.Resp.Attributes.Name)
	}
	if loc, ok := cache[albumID]; ok {
		track.SaveDir = loc.dir
		track.CoverPath = loc.cover
		track.Codec = codec
		return nil
	}

	trackCodec, quality := resolveQualityLabelForTrack(storefront, track.ID, language, token, codec)
	artistFolderName := buildArtistFolderNameFromAlbum(album)
	root := saveRootForCurrentCodec()
	singerFolder := root
	if artistFolderName != "" {
		singerFolder = filepath.Join(root, artistFolderName)
	}
	if Config.SaveArtistCover && len(album.Relationships.Artists.Data) > 0 {
		if album.Relationships.Artists.Data[0].Attributes.Artwork.Url != "" {
			if _, err := writeCover(singerFolder, "folder", album.Relationships.Artists.Data[0].Attributes.Artwork.Url); err != nil {
				fmt.Println("Failed to write artist cover.")
			}
		}
	}
	albumFolderName := buildAlbumFolderNameFromAlbum(album, albumID, trackCodec, quality)
	albumFolderPath := singerFolder
	if albumFolderName != "" {
		albumFolderPath = filepath.Join(singerFolder, albumFolderName)
	}
	if err := os.MkdirAll(albumFolderPath, os.ModePerm); err != nil {
		return err
	}
	covPath, err := writeCover(albumFolderPath, "cover", album.Attributes.Artwork.URL)
	if err != nil {
		fmt.Println("Failed to write cover.")
	}
	cache[albumID] = struct {
		dir   string
		cover string
	}{dir: albumFolderPath, cover: covPath}
	track.SaveDir = albumFolderPath
	track.CoverPath = covPath
	track.Codec = codec
	return nil
}

func buildPlaylistCollectFilename(track *task.Track, ext string) string {
	if ext == "" {
		ext = ".m4a"
	}
	name := fmt.Sprintf("%02d. %s - %s",
		track.TaskNum,
		LimitString(track.Resp.Attributes.ArtistName),
		LimitString(track.Resp.Attributes.Name),
	)
	return forbiddenNames.ReplaceAllString(name, "_") + ext
}

func mirrorAddedTrackToPlaylistFolder(playlistDir string, track *task.Track, beforeLen int) error {
	if len(AddedTracks) <= beforeLen {
		return nil
	}
	last := &AddedTracks[len(AddedTracks)-1]
	src := strings.TrimSpace(last.Path)
	if src == "" {
		src = strings.TrimSpace(track.SavePath)
	}
	if src == "" {
		return nil
	}
	ext := filepath.Ext(src)
	if ext == "" {
		ext = ".m4a"
	}
	dst := filepath.Join(playlistDir, buildPlaylistCollectFilename(track, ext))
	if samePath, _ := fileExists(dst); samePath {
		last.Path = dst
		return nil
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	last.Path = dst
	return nil
}
