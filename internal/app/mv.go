package app

import (
	"errors"
	"fmt"
	"amdl/internal/amp-api"
	"amdl/internal/model"
	"amdl/internal/widevine-rip/runv5"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	tags := []string{
		"tool=",
		fmt.Sprintf("artist=%s", MVInfo.Data[0].Attributes.ArtistName),
		fmt.Sprintf("title=%s", MVInfo.Data[0].Attributes.Name),
		fmt.Sprintf("genre=%s", firstGenre(MVInfo.Data[0].Attributes.GenreNames)),
		fmt.Sprintf("created=%s", MVInfo.Data[0].Attributes.ReleaseDate),
		fmt.Sprintf("ISRC=%s", MVInfo.Data[0].Attributes.Isrc),
	}

	if MVInfo.Data[0].Attributes.ContentRating == "explicit" {
		tags = append(tags, "rating=1")
	} else if MVInfo.Data[0].Attributes.ContentRating == "clean" {
		tags = append(tags, "rating=2")
	} else {
		tags = append(tags, "rating=0")
	}

	if track != nil {
		if track.PreType == "playlists" && !r.Config.UseSongInfoForPlaylist {
			tags = append(tags, "disk=1/1")
			tags = append(tags, fmt.Sprintf("album=%s", track.PlaylistData.Attributes.Name))
			tags = append(tags, fmt.Sprintf("track=%d", track.TaskNum))
			tags = append(tags, fmt.Sprintf("tracknum=%d/%d", track.TaskNum, track.TaskTotal))
			tags = append(tags, fmt.Sprintf("album_artist=%s", track.PlaylistData.Attributes.ArtistName))
			tags = append(tags, fmt.Sprintf("performer=%s", track.Resp.Attributes.ArtistName))
		} else if track.PreType == "playlists" && r.Config.UseSongInfoForPlaylist {
			tags = append(tags, fmt.Sprintf("album=%s", track.AlbumData.Attributes.Name))
			tags = append(tags, fmt.Sprintf("disk=%d/%d", track.Resp.Attributes.DiscNumber, track.DiscTotal))
			tags = append(tags, fmt.Sprintf("track=%d", track.Resp.Attributes.TrackNumber))
			tags = append(tags, fmt.Sprintf("tracknum=%d/%d", track.Resp.Attributes.TrackNumber, track.AlbumData.Attributes.TrackCount))
			tags = append(tags, fmt.Sprintf("album_artist=%s", track.AlbumData.Attributes.ArtistName))
			tags = append(tags, fmt.Sprintf("performer=%s", track.Resp.Attributes.ArtistName))
			tags = append(tags, fmt.Sprintf("copyright=%s", track.AlbumData.Attributes.Copyright))
			tags = append(tags, fmt.Sprintf("UPC=%s", track.AlbumData.Attributes.Upc))
		} else {
			tags = append(tags, fmt.Sprintf("album=%s", track.AlbumData.Attributes.Name))
			tags = append(tags, fmt.Sprintf("disk=%d/%d", track.Resp.Attributes.DiscNumber, track.DiscTotal))
			tags = append(tags, fmt.Sprintf("track=%d", track.Resp.Attributes.TrackNumber))
			tags = append(tags, fmt.Sprintf("tracknum=%d/%d", track.Resp.Attributes.TrackNumber, track.AlbumData.Attributes.TrackCount))
			tags = append(tags, fmt.Sprintf("album_artist=%s", track.AlbumData.Attributes.ArtistName))
			tags = append(tags, fmt.Sprintf("performer=%s", track.Resp.Attributes.ArtistName))
			tags = append(tags, fmt.Sprintf("copyright=%s", track.AlbumData.Attributes.Copyright))
			tags = append(tags, fmt.Sprintf("UPC=%s", track.AlbumData.Attributes.Upc))
		}
	} else {
		tags = append(tags, fmt.Sprintf("album=%s", MVInfo.Data[0].Attributes.AlbumName))
		tags = append(tags, fmt.Sprintf("disk=%d", MVInfo.Data[0].Attributes.DiscNumber))
		tags = append(tags, fmt.Sprintf("track=%d", MVInfo.Data[0].Attributes.TrackNumber))
		tags = append(tags, fmt.Sprintf("tracknum=%d", MVInfo.Data[0].Attributes.TrackNumber))
		tags = append(tags, fmt.Sprintf("performer=%s", MVInfo.Data[0].Attributes.ArtistName))
	}

	var covPath string
	thumbURL := MVInfo.Data[0].Attributes.Artwork.URL
	baseThumbName := forbiddenNames.ReplaceAllString(mvSaveName, "_") + "_thumbnail"
	covPath, err = r.writeCover(saveDir, baseThumbName, thumbURL)
	if err != nil {
		fmt.Println("Failed to save MV thumbnail:", err)
	} else {
		tags = append(tags, fmt.Sprintf("cover=%s", covPath))
	}
	defer os.Remove(covPath)

	tagsString := strings.Join(tags, ":")
	muxCmd := exec.Command("MP4Box", "-itags", tagsString, "-quiet", "-add", vidPath, "-add", audPath, "-keep-utc", "-new", mvOutPath)
	fmt.Printf("MV Remuxing...")
	if err := muxCmd.Run(); err != nil {
		fmt.Printf("MV mux failed: %v\n", err)
		return err
	}
	fmt.Printf("\rMV Remuxed.   \n")

	// Append to r.State.AddedTracks
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
