package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"amdl/internal/amp-api"
	mvmedia "amdl/internal/media/mv"
	"amdl/internal/model"
	"amdl/internal/widevine-rip/runv5"

	"github.com/itouakirai/go-mp4tag"
)

func (r *Runner) mvDownloader(adamID string, saveDir string, token string, storefront string, track *model.Track) error {
	MVInfo, err := ampapi.GetMusicVideoResp(storefront, adamID, r.Config.Language, token)
	if err != nil {
		fmt.Println("\u26A0 Failed to get MV manifest:", err)
		return nil
	}
	if len(MVInfo.Data) == 0 {
		return errors.New("music video response contains no data")
	}

	if strings.HasSuffix(saveDir, ".") {
		saveDir = strings.ReplaceAll(saveDir, ".", "")
	}
	saveDir = strings.TrimSpace(saveDir)

	vidPath := filepath.Join(saveDir, fmt.Sprintf("%s_vid.mp4", adamID))
	audPath := filepath.Join(saveDir, fmt.Sprintf("%s_aud.mp4", adamID))
	mvSaveName := fmt.Sprintf("%s (%s)", MVInfo.Data[0].Attributes.Name, adamID)
	if track != nil {
		mvSaveName = fmt.Sprintf("%02d. %s", track.TaskNum, MVInfo.Data[0].Attributes.Name)
	}

	mvOutPath := filepath.Join(saveDir, fmt.Sprintf("%s.mp4", forbiddenNames.ReplaceAllString(mvSaveName, "_")))

	fmt.Println(MVInfo.Data[0].Attributes.Name)

	exists, _ := fileExists(mvOutPath)
	if exists {
		fmt.Println("MV already exists locally.")

		mvArtistName := MVInfo.Data[0].Attributes.ArtistName
		mvAlbumName := MVInfo.Data[0].Attributes.AlbumName
		mvName := MVInfo.Data[0].Attributes.Name
		mvArtistId := ""
		if len(MVInfo.Data[0].Relationships.Artists.Data) > 0 {
			mvArtistId = MVInfo.Data[0].Relationships.Artists.Data[0].ID
		}

		r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
			Path:     mvOutPath,
			Artist:   mvArtistName,
			ArtistID: mvArtistId,
			Album:    mvAlbumName,
			Song:     mvName,
		})
		return nil
	}

	mvm3u8url, _, _, err := runv5.GetWebplayback(adamID, r.Config.LiteServer, true)
	if err != nil {
		return err
	}
	if mvm3u8url == "" {
		return errors.New("lite-server returned no web playback URL")
	}

	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		return err
	}

	videom3u8url, err := r.extractVideo(mvm3u8url)
	if err != nil {
		return fmt.Errorf("extract video manifest: %w", err)
	}
	videokeyAndUrls, err := runv5.Run(adamID, videom3u8url, token, true, r.Config.LiteServer)
	if err != nil {
		return fmt.Errorf("download video stream: %w", err)
	}
	if err := runv5.ExtMvData(videokeyAndUrls, vidPath); err != nil {
		return fmt.Errorf("write video stream: %w", err)
	}
	defer os.Remove(vidPath)

	audiom3u8url, err := r.extractMvAudio(mvm3u8url)
	if err != nil {
		return fmt.Errorf("extract audio manifest: %w", err)
	}
	audiokeyAndUrls, err := runv5.Run(adamID, audiom3u8url, token, true, r.Config.LiteServer)
	if err != nil {
		return fmt.Errorf("download audio stream: %w", err)
	}
	if err := runv5.ExtMvData(audiokeyAndUrls, audPath); err != nil {
		return fmt.Errorf("write audio stream: %w", err)
	}
	defer os.Remove(audPath)

	var covPath string
	if r.Config.EmbedCover {
		thumbURL := MVInfo.Data[0].Attributes.Artwork.URL
		baseThumbName := forbiddenNames.ReplaceAllString(mvSaveName, "_") + "_thumbnail"
		covPath, err = r.writeCover(saveDir, baseThumbName, thumbURL)
		if err != nil {
			fmt.Println("Failed to save MV thumbnail:", err)
			covPath = ""
		} else {
			defer os.Remove(covPath)
		}
	}

	fmt.Printf("MV Remuxing...")
	if err := mvmedia.Mux(vidPath, audPath, mvOutPath); err != nil {
		fmt.Printf("MV mux failed: %v\n", err)
		return err
	}
	fmt.Printf("\rMV Remuxed.   \n")

	if err := r.writeMVMP4Tags(mvOutPath, MVInfo, track, covPath); err != nil {
		_ = os.Remove(mvOutPath)
		fmt.Printf("MV tag writing failed: %v\n", err)
		return err
	}

	mvArtistName := MVInfo.Data[0].Attributes.ArtistName
	mvAlbumName := MVInfo.Data[0].Attributes.AlbumName
	mvName := MVInfo.Data[0].Attributes.Name
	mvArtistId := ""
	if len(MVInfo.Data[0].Relationships.Artists.Data) > 0 {
		mvArtistId = MVInfo.Data[0].Relationships.Artists.Data[0].ID
	}

	r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
		Path:     mvOutPath,
		Artist:   mvArtistName,
		ArtistID: mvArtistId,
		Album:    mvAlbumName,
		Song:     mvName,
	})

	return nil
}

func (r *Runner) writeMVMP4Tags(path string, mvInfo *ampapi.MusicVideoResp, track *model.Track, coverPath string) error {
	if mvInfo == nil || len(mvInfo.Data) == 0 {
		return errors.New("music video response contains no data")
	}
	attrs := mvInfo.Data[0].Attributes

	tags := &mp4tag.MP4Tags{
		Title:       attrs.Name,
		Artist:      attrs.ArtistName,
		Album:       attrs.AlbumName,
		CustomGenre: firstGenre(attrs.GenreNames),
		Date:        attrs.ReleaseDate,
		TrackNumber: int16(attrs.TrackNumber),
		DiscNumber:  int16(attrs.DiscNumber),
		Custom: map[string]string{
			"PERFORMER":   attrs.ArtistName,
			"RELEASETIME": attrs.ReleaseDate,
			"ISRC":        attrs.Isrc,
		},
	}

	switch {
	case track != nil && (track.PreType == "playlists" || track.PreType == "stations") && !r.Config.UseSongInfoForPlaylist:
		tags.Album = track.PlaylistData.Attributes.Name
		tags.DiscNumber = 1
		tags.DiscTotal = 1
		tags.TrackNumber = int16(track.TaskNum)
		tags.TrackTotal = int16(track.TaskTotal)
		tags.AlbumArtist = track.PlaylistData.Attributes.ArtistName
		tags.Custom["PERFORMER"] = track.Resp.Attributes.ArtistName
	case track != nil:
		tags.Album = track.AlbumData.Attributes.Name
		tags.DiscNumber = int16(track.Resp.Attributes.DiscNumber)
		tags.DiscTotal = int16(track.DiscTotal)
		tags.TrackNumber = int16(track.Resp.Attributes.TrackNumber)
		tags.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
		tags.AlbumArtist = track.AlbumData.Attributes.ArtistName
		tags.Custom["PERFORMER"] = track.Resp.Attributes.ArtistName
		tags.Custom["UPC"] = track.AlbumData.Attributes.Upc
		tags.Copyright = track.AlbumData.Attributes.Copyright
		tags.Publisher = track.AlbumData.Attributes.RecordLabel
	}

	if r.Config.TagSortOrder {
		tags.TitleSort = attrs.Name
		tags.ArtistSort = attrs.ArtistName
		tags.AlbumSort = tags.Album
		tags.AlbumArtistSort = tags.AlbumArtist
	}

	switch attrs.ContentRating {
	case "explicit":
		tags.ItunesAdvisory = mp4tag.ItunesAdvisoryExplicit
	case "clean":
		tags.ItunesAdvisory = mp4tag.ItunesAdvisoryClean
	default:
		tags.ItunesAdvisory = mp4tag.ItunesAdvisoryNone
	}

	if r.Config.EmbedCover && coverPath != "" {
		cover, err := os.ReadFile(coverPath)
		if err != nil {
			return fmt.Errorf("read MV cover: %w", err)
		}
		tags.Pictures = []*mp4tag.MP4Picture{{
			Format: mp4tag.ImageTypeAuto,
			Data:   cover,
		}}
	}

	mp4, err := mp4tag.Open(path)
	if err != nil {
		return err
	}
	defer mp4.Close()
	return mp4.Write(tags, []string{})
}
