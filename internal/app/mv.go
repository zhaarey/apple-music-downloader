package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"amdl/internal/amp-api"
	mvmedia "amdl/internal/media/mv"
	"amdl/internal/model"
	playreadyrip "amdl/internal/playready-rip"
	widevinerip "amdl/internal/widevine-rip"
	"amdl/internal/widevine-rip/runv5"
	"amdl/internal/wrapper"

	"github.com/itouakirai/go-mp4tag"
)

func (r *Runner) mvDownloader(adamID string, saveDir string, token string, storefront string, track *model.Track) error {
	MVInfo, err := ampapi.GetMusicVideoResp(storefront, adamID, r.Config.General.Language, token)
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
		r.recordAddedMV(mvOutPath, MVInfo)
		return nil
	}

	mvm3u8url, err := wrapper.GetWebplayback(r.Config.General.LiteServer, adamID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		return err
	}

	defer os.Remove(vidPath)
	defer os.Remove(audPath)
	if err := r.downloadMVTracks(context.Background(), adamID, mvm3u8url, vidPath, audPath); err != nil {
		return err
	}

	var covPath string
	if r.Config.Metadata.Artwork.Embed {
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

	r.recordAddedMV(mvOutPath, MVInfo)
	return nil
}

// recordAddedMV registers the finished (or already present) MV at path so
// post-processing, such as the artist and playlist steps, sees it.
func (r *Runner) recordAddedMV(path string, mvInfo *ampapi.MusicVideoResp) {
	data := mvInfo.Data[0]
	artistID := ""
	if len(data.Relationships.Artists.Data) > 0 {
		artistID = data.Relationships.Artists.Data[0].ID
	}
	r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
		Path:     path,
		Artist:   data.Attributes.ArtistName,
		ArtistID: artistID,
		Album:    data.Attributes.AlbumName,
		Song:     data.Attributes.Name,
	})
}

// downloadMVTracks picks the best video and audio variants from the master
// playlist at masterURL and writes each as a clear fragmented MP4 to
// videoPath and audioPath.
func (r *Runner) downloadMVTracks(ctx context.Context, adamID, masterURL, videoPath, audioPath string) error {
	videoURL, usePlayReady, err := r.extractVideoVariant(masterURL)
	if err != nil {
		return fmt.Errorf("extract video manifest: %w", err)
	}
	if usePlayReady {
		fmt.Println("Video DRM: PlayReady")
	}
	if err := r.downloadMVTrack(ctx, adamID, videoURL, usePlayReady, videoPath); err != nil {
		return fmt.Errorf("video stream: %w", err)
	}

	audioURL, err := r.extractMvAudio(masterURL)
	if err != nil {
		return fmt.Errorf("extract audio manifest: %w", err)
	}
	if err := r.downloadMVTrack(ctx, adamID, audioURL, false, audioPath); err != nil {
		return fmt.Errorf("audio stream: %w", err)
	}
	return nil
}

// downloadMVTrack downloads one MV track playlist to outputPath. The
// playlist may be Widevine- or PlayReady-encrypted, or clear (some music
// videos are served unprotected), in which case no license is requested.
func (r *Runner) downloadMVTrack(ctx context.Context, adamID, playlistURL string, usePlayReady bool, outputPath string) error {
	var (
		stream widevinerip.Stream
		err    error
	)
	if usePlayReady {
		stream, err = playreadyrip.FetchStream(ctx, adamID, playlistURL, r.Config.General.LiteServer)
	} else {
		stream, err = runv5.FetchStream(ctx, adamID, playlistURL, r.Config.General.LiteServer)
	}
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	if err := widevinerip.DownloadStream(ctx, stream, outputPath); err != nil {
		return fmt.Errorf("write: %w", err)
	}
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
	case track != nil && (track.PreType == "playlists" || track.PreType == "stations") && !r.Config.Metadata.Tags.UseSongInfoForPlaylist:
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

	if r.Config.Metadata.Tags.SortOrder {
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

	if r.Config.Metadata.Artwork.Embed && coverPath != "" {
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
