package app

import (
	"fmt"
	"amdl/internal/amp-api"
	fairplayrip "amdl/internal/fairplay-rip"
	"amdl/internal/model"
	"amdl/internal/media/alacfix"
	"amdl/internal/media/lyrics"
	"amdl/internal/widevine-rip"
	"amdl/internal/widevine-rip/runv5"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (r *Runner) ripTrack(track *model.Track, token string, mediaUserToken string) {
	var err error
	r.State.Counter.Total++
	fmt.Printf("Track %d of %d: %s\n", track.TaskNum, track.TaskTotal, track.Type)

	//mv dl dev
	if track.Type == "music-videos" {
		if r.Config.LiteServer == "" {
			fmt.Println("lite-server is not set, skip MV dl")
			r.State.Counter.Success++
			return
		}
		err := r.mvDownloader(track.ID, track.SaveDir, token, track.Storefront, track)
		if err != nil {
			fmt.Println("\u26A0 Failed to dl MV:", err)
			r.State.Counter.Error++
			return
		}
		r.State.Counter.Success++
		return
	}

	needDlAacLc := false
	if r.Flags.AAC && r.Config.AacType == "aac-lc" {
		needDlAacLc = true
	}
	if track.WebM3u8 == "" && !needDlAacLc {
		if r.Flags.Atmos {
			fmt.Println("Unavailable")
			r.State.Counter.Unavailable++
			return
		}
		fmt.Println("Unavailable, trying to dl aac-lc")
		needDlAacLc = true
	}
	needCheck := false

	if r.Config.GetM3u8Mode == "all" {
		needCheck = true
	} else if r.Config.GetM3u8Mode == "hires" && contains(track.Resp.Attributes.AudioTraits, "hi-res-lossless") {
		needCheck = true
	}
	var EnhancedHls_m3u8 string
	if needCheck && !needDlAacLc {
		EnhancedHls_m3u8, _ = r.checkM3u8(track.ID, "song")
		if strings.HasSuffix(EnhancedHls_m3u8, ".m3u8") {
			track.DeviceM3u8 = EnhancedHls_m3u8
			track.M3u8 = EnhancedHls_m3u8
		}
	}
	var Quality string
	if strings.Contains(r.Config.SongFileFormat, "Quality") {
		if r.Flags.Atmos {
			Quality = fmt.Sprintf("%dKbps", r.Config.AtmosMax-2000)
		} else if needDlAacLc {
			Quality = "256Kbps"
		} else {
			_, Quality, err = r.extractMedia(track.M3u8, true)
			if err != nil {
				fmt.Println("Failed to extract quality from manifest.\n", err)
				r.State.Counter.Error++
				return
			}
		}
	}
	track.Quality = Quality

	stringsToJoin := []string{}
	if track.Resp.Attributes.IsAppleDigitalMaster {
		if r.Config.AppleMasterChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.AppleMasterChoice)
		}
	}
	if track.Resp.Attributes.ContentRating == "explicit" {
		if r.Config.ExplicitChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.ExplicitChoice)
		}
	}
	if track.Resp.Attributes.ContentRating == "clean" {
		if r.Config.CleanChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.CleanChoice)
		}
	}
	Tag_string := strings.Join(stringsToJoin, " ")

	songName := strings.NewReplacer(
		"{SongId}", track.ID,
		"{SongNumer}", fmt.Sprintf("%02d", track.TaskNum),
		"{ArtistName}", r.LimitString(track.Resp.Attributes.ArtistName),
		"{SongName}", r.LimitString(track.Resp.Attributes.Name),
		"{DiscNumber}", fmt.Sprintf("%0d", track.Resp.Attributes.DiscNumber),
		"{TrackNumber}", fmt.Sprintf("%0d", track.Resp.Attributes.TrackNumber),
		"{Quality}", Quality,
		"{Tag}", Tag_string,
		"{Codec}", track.Codec,
	).Replace(r.Config.SongFileFormat)
	fmt.Println(songName)
	filename := fmt.Sprintf("%s.m4a", forbiddenNames.ReplaceAllString(songName, "_"))
	track.SaveName = filename
	trackPath := filepath.Join(track.SaveDir, track.SaveName)
	lrcFilename := fmt.Sprintf("%s.%s", forbiddenNames.ReplaceAllString(songName, "_"), r.Config.LrcFormat)

	// Determine possible post-conversion target file (so we can skip re-download)
	var convertedPath string
	considerConverted := false
	if r.Config.ConvertAfterDownload &&
		r.Config.ConvertFormat != "" &&
		strings.ToLower(r.Config.ConvertFormat) != "copy" &&
		!r.Config.ConvertKeepOriginal {
		convertedPath = strings.TrimSuffix(trackPath, filepath.Ext(trackPath)) + "." + strings.ToLower(r.Config.ConvertFormat)
		considerConverted = true
	}
	// Existence check now considers converted output (if original was deleted)
	existsOriginal, err := fileExists(trackPath)
	if err != nil {
		fmt.Println("Failed to check if track exists.")
	}
	if existsOriginal {
		fmt.Println("Track already exists locally.")
		r.State.Counter.Success++
		r.State.OKDict[track.PreID] = append(r.State.OKDict[track.PreID], track.TaskNum)

		tArtistId := ""
		if len(track.Resp.Relationships.Artists.Data) > 0 {
			tArtistId = track.Resp.Relationships.Artists.Data[0].ID
		}
		r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
			Path:     trackPath,
			Artist:   track.Resp.Attributes.ArtistName,
			ArtistID: tArtistId,
			Album:    track.Resp.Attributes.AlbumName,
			Song:     track.Resp.Attributes.Name,
		})
		return
	}
	if considerConverted {
		existsConverted, err2 := fileExists(convertedPath)
		if err2 == nil && existsConverted {
			fmt.Println("Converted track already exists locally.")
			r.State.Counter.Success++
			r.State.OKDict[track.PreID] = append(r.State.OKDict[track.PreID], track.TaskNum)

			tArtistId := ""
			if len(track.Resp.Relationships.Artists.Data) > 0 {
				tArtistId = track.Resp.Relationships.Artists.Data[0].ID
			}
			r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
				Path:     convertedPath,
				Artist:   track.Resp.Attributes.ArtistName,
				ArtistID: tArtistId,
				Album:    track.Resp.Attributes.AlbumName,
				Song:     track.Resp.Attributes.Name,
			})
			return
		}
	}

	//提前获取到的播放列表下track所在的专辑信息
	if track.PreType == "playlists" && r.Config.UseSongInfoForPlaylist {
		if err := track.GetAlbumData(token); err != nil {
			fmt.Println("Failed to get album data for playlist track:", err)
			r.State.Counter.Error++
			return
		}
	}

	//get lrc
	var lrc string = ""
	if r.Config.EmbedLrc || r.Config.SaveLrcFile {
		lrcStr, err := lyrics.Get(track.ID, r.Config.LrcType, r.Config.Language, r.Config.LrcFormat, r.Config.LiteServer, r.Config.LrcExtra)
		if err != nil {
			fmt.Println(err)
		} else {
			if r.Config.SaveLrcFile {
				err := r.writeLyrics(track.SaveDir, lrcFilename, lrcStr)
				if err != nil {
					fmt.Printf("Failed to write lyrics")
				}
			}
			if r.Config.EmbedLrc {
				lrc = lrcStr
			}
		}
	}

	if needDlAacLc {
		if r.Config.LiteServer == "" {
			fmt.Println("aac-lc download requires lite-server, but it is not configured")
			r.State.Counter.Error++
			return
		}
		_, err := runv5.Run(track.ID, trackPath, token, false, r.Config.LiteServer)
		if err != nil {
			fmt.Println("Failed to dl aac-lc via lite-server:", err)
			if err.Error() == "Unavailable" {
				r.State.Counter.Unavailable++
				return
			}
			r.State.Counter.Error++
			return
		}
	} else {
		trackM3u8Url, _, err := r.extractMedia(track.M3u8, false)
		if err != nil {
			fmt.Println("\u26A0 Failed to extract info from manifest:", err)
			r.State.Counter.Unavailable++
			return
		}
		//边下载边解密
		//wrapper-lite 模板解密
		err = fairplayrip.Run(track.ID, trackM3u8Url, trackPath, r.Config)
		if err != nil {
			fmt.Println("Failed to run v4:", err)
			r.State.Counter.Error++
			return
		}

	}
	//这里利用MP4box将fmp4转化为mp4，并添加ilst box与cover，方便后面的mp4tag添加更多自定义标签
	tags := []string{
		"tool=",
		"artist=AppleMusic",
	}
	if r.Config.EmbedCover {
		if (strings.Contains(track.PreID, "pl.") || strings.Contains(track.PreID, "ra.")) && r.Config.DlAlbumcoverForPlaylist {
			track.CoverPath, err = r.writeCover(track.SaveDir, track.ID, track.Resp.Attributes.Artwork.URL)
			if err != nil {
				fmt.Println("Failed to write cover.")
			}
		}
		tags = append(tags, fmt.Sprintf("cover=%s", track.CoverPath))
	}
	tagsString := strings.Join(tags, ":")
	cmd := exec.Command("MP4Box", "-itags", tagsString, trackPath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Embed failed: %v\n", err)
		r.State.Counter.Error++
		return
	}
	if (strings.Contains(track.PreID, "pl.") || strings.Contains(track.PreID, "ra.")) && r.Config.DlAlbumcoverForPlaylist {
		if err := os.Remove(track.CoverPath); err != nil {
			fmt.Printf("Error deleting file: %s\n", track.CoverPath)
			r.State.Counter.Error++
			return
		}
	}
	track.SavePath = trackPath

	if r.Config.ALACFix {
		err = alacfix.Run(track.SavePath, false)
		if err != nil {
			fmt.Println("\u26A0 Failed to fix ALAC:", err)
			r.State.Counter.Unavailable++
			return
		}
	}

	err = r.writeMP4Tags(track, lrc)
	if err != nil {
		fmt.Println("\u26A0 Failed to write tags in media:", err)
		r.State.Counter.Unavailable++
		return
	}

	// CONVERSION FEATURE hook
	r.convertIfNeeded(track)

	tArtistId := ""
	if len(track.Resp.Relationships.Artists.Data) > 0 {
		tArtistId = track.Resp.Relationships.Artists.Data[0].ID
	}
	r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
		Path:     track.SavePath,
		Artist:   track.Resp.Attributes.ArtistName,
		ArtistID: tArtistId,
		Album:    track.Resp.Attributes.AlbumName,
		Song:     track.Resp.Attributes.Name,
	})

	r.State.Counter.Success++
	r.State.OKDict[track.PreID] = append(r.State.OKDict[track.PreID], track.TaskNum)
}

func releaseYear(date string) string {
	if len(date) < 4 {
		return ""
	}
	return date[:4]
}

func (r *Runner) ripStation(albumId string, token string, storefront string, mediaUserToken string) error {
	station := model.NewStation(storefront, albumId)
	err := station.GetResp(mediaUserToken, token, r.Config.Language)
	if err != nil {
		return err
	}
	fmt.Println(" -", station.Type)
	meta := station.Resp

	var Codec string
	if r.Flags.Atmos {
		Codec = "ATMOS"
	} else if r.Flags.AAC {
		Codec = "AAC"
	} else {
		Codec = "ALAC"
	}
	station.Codec = Codec
	var singerFoldername string
	if r.Config.ArtistFolderFormat != "" {
		singerFoldername = strings.NewReplacer(
			"{ArtistName}", "Apple Music Station",
			"{ArtistId}", "",
			"{UrlArtistName}", "Apple Music Station",
		).Replace(r.Config.ArtistFolderFormat)
		if strings.HasSuffix(singerFoldername, ".") {
			singerFoldername = strings.ReplaceAll(singerFoldername, ".", "")
		}
		singerFoldername = strings.TrimSpace(singerFoldername)
		fmt.Println(singerFoldername)
	}
	singerFolder := filepath.Join(r.Config.AlacSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	if r.Flags.Atmos {
		singerFolder = filepath.Join(r.Config.AtmosSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	}
	if r.Flags.AAC {
		singerFolder = filepath.Join(r.Config.AacSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	}
	if err := createDirectory(singerFolder); err != nil {
		return err
	}
	station.SaveDir = singerFolder

	playlistFolder := strings.NewReplacer(
		"{ArtistName}", "Apple Music Station",
		"{PlaylistName}", r.LimitString(station.Name),
		"{PlaylistId}", station.ID,
		"{Quality}", "",
		"{Codec}", Codec,
		"{Tag}", "",
	).Replace(r.Config.PlaylistFolderFormat)
	playlistFolderPath, err := r.prepareCollectionFolder(singerFolder, playlistFolder)
	if err != nil {
		return err
	}
	station.SaveName = playlistFolder
	fmt.Println(playlistFolder)

	covPath, err := r.writeCover(playlistFolderPath, "cover", meta.Data[0].Attributes.Artwork.URL)
	if err != nil {
		fmt.Println("Failed to write cover.")
	}
	station.CoverPath = covPath

	if r.Config.SaveAnimatedArtwork {
		r.saveAnimatedArtwork(playlistFolderPath, meta.Data[0].Attributes.EditorialVideo.MotionSquare.Video, "")
	}
	if station.Type == "stream" {
		r.State.Counter.Total++
		if contains(r.State.OKDict[station.ID], 1) {
			r.State.Counter.Success++
			return nil
		}
		songName := strings.NewReplacer(
			"{SongId}", station.ID,
			"{SongNumer}", "01",
			"{SongName}", r.LimitString(station.Name),
			"{DiscNumber}", "1",
			"{TrackNumber}", "1",
			"{Quality}", "256Kbps",
			"{Tag}", "",
			"{Codec}", "AAC",
		).Replace(r.Config.SongFileFormat)
		fmt.Println(songName)
		trackPath := filepath.Join(playlistFolderPath, fmt.Sprintf("%s.m4a", forbiddenNames.ReplaceAllString(songName, "_")))
		exists, _ := fileExists(trackPath)
		if exists {
			r.State.Counter.Success++
			r.State.OKDict[station.ID] = append(r.State.OKDict[station.ID], 1)

			fmt.Println("Radio already exists locally.")
			r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
				Path:     trackPath,
				Artist:   "Apple Music Station",
				ArtistID: "",
				Album:    station.Name,
				Song:     station.Name,
			})
			return nil
		}
		assetsUrl, serverUrl, err := ampapi.GetStationAssetsUrlAndServerUrl(station.ID, mediaUserToken, token)
		if err != nil {
			fmt.Println("Failed to get station assets url.", err)
			r.State.Counter.Error++
			return err
		}
		trackM3U8, err := widevinerip.ResolveStationVariantPlaylist(assetsUrl, token, mediaUserToken)
		if err != nil {
			fmt.Println("Failed to resolve station variant playlist.", err)
			r.State.Counter.Error++
			return err
		}
		keyAndUrls, err := widevinerip.Run(station.ID, trackM3U8, token, mediaUserToken, true, serverUrl)
		if err != nil {
			fmt.Println("Failed to get station stream decryption key.", err)
			r.State.Counter.Error++
			return err
		}
		err = widevinerip.ExtMvData(keyAndUrls, trackPath)
		if err != nil {
			fmt.Println("Failed to download station stream.", err)
			r.State.Counter.Error++
			return err
		}
		tags := []string{
			"tool=",
			"disk=1/1",
			"track=1",
			"tracknum=1/1",
			fmt.Sprintf("artist=%s", "Apple Music Station"),
			fmt.Sprintf("performer=%s", "Apple Music Station"),
			fmt.Sprintf("album_artist=%s", "Apple Music Station"),
			fmt.Sprintf("album=%s", station.Name),
			fmt.Sprintf("title=%s", station.Name),
		}
		if r.Config.EmbedCover {
			tags = append(tags, fmt.Sprintf("cover=%s", station.CoverPath))
		}
		tagsString := strings.Join(tags, ":")
		cmd := exec.Command("MP4Box", "-itags", tagsString, trackPath)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Embed failed: %v\n", err)
		}
		r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
			Path:     trackPath,
			Artist:   "Apple Music Station",
			ArtistID: "",
			Album:    station.Name,
			Song:     station.Name,
		})
		r.State.Counter.Success++
		r.State.OKDict[station.ID] = append(r.State.OKDict[station.ID], 1)
		return nil
	}

	for i := range station.Tracks {
		station.Tracks[i].CoverPath = covPath
		station.Tracks[i].SaveDir = playlistFolderPath
		station.Tracks[i].Codec = Codec
	}

	trackTotal := len(station.Tracks)
	arr := make([]int, trackTotal)
	for i := 0; i < trackTotal; i++ {
		arr[i] = i + 1
	}
	var selected []int

	if !r.Flags.Select {
		selected = arr
	} else {
		selected = station.ShowSelect()
	}
	for i := range station.Tracks {
		i++
		if contains(selected, i) {
			r.ripTrack(&station.Tracks[i-1], token, mediaUserToken)
		}
	}
	r.saveM3UPlaylist(playlistFolderPath, playlistFolder)
	return nil
}

func (r *Runner) ripAlbum(albumId string, token string, storefront string, mediaUserToken string, urlArg_i string) error {
	album := model.NewAlbum(storefront, albumId)
	err := album.GetResp(token, r.Config.Language)
	if err != nil {
		fmt.Println("Failed to get album response.")
		return err
	}
	meta := album.Resp
	if r.Flags.Debug {
		fmt.Println(meta.Data[0].Attributes.ArtistName)
		fmt.Println(meta.Data[0].Attributes.Name)

		for trackNum, track := range meta.Data[0].Relationships.Tracks.Data {
			trackNum++
			fmt.Printf("\nTrack %d of %d:\n", trackNum, len(meta.Data[0].Relationships.Tracks.Data))
			fmt.Printf("%02d. %s\n", trackNum, track.Attributes.Name)

			manifest, err := ampapi.GetSongResp(storefront, track.ID, album.Language, token)
			if err != nil {
				fmt.Printf("Failed to get manifest for track %d: %v\n", trackNum, err)
				continue
			}

			var m3u8Url string
			if manifest.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls != "" {
				m3u8Url = manifest.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls
			}
			needCheck := false
			if r.Config.GetM3u8Mode == "all" {
				needCheck = true
			} else if r.Config.GetM3u8Mode == "hires" && contains(track.Attributes.AudioTraits, "hi-res-lossless") {
				needCheck = true
			}
			if needCheck {
				fullM3u8Url, err := r.checkM3u8(track.ID, "song")
				if err == nil && strings.HasSuffix(fullM3u8Url, ".m3u8") {
					m3u8Url = fullM3u8Url
				} else {
					fmt.Println("Failed to get best quality m3u8 from lite-server, will use m3u8 from Web API")
				}
			}

			_, _, err = r.extractMedia(m3u8Url, true)
			if err != nil {
				fmt.Printf("Failed to extract quality info for track %d: %v\n", trackNum, err)
				continue
			}
		}
		return nil
	}
	var Codec string
	if r.Flags.Atmos {
		Codec = "ATMOS"
	} else if r.Flags.AAC {
		Codec = "AAC"
	} else {
		Codec = "ALAC"
	}
	album.Codec = Codec
	var singerFoldername string
	if r.Config.ArtistFolderFormat != "" {
		if len(meta.Data[0].Relationships.Artists.Data) > 0 {
			singerFoldername = strings.NewReplacer(
				"{UrlArtistName}", r.LimitString(meta.Data[0].Attributes.ArtistName),
				"{ArtistName}", r.LimitString(meta.Data[0].Attributes.ArtistName),
				"{ArtistId}", meta.Data[0].Relationships.Artists.Data[0].ID,
			).Replace(r.Config.ArtistFolderFormat)
		} else {
			singerFoldername = strings.NewReplacer(
				"{UrlArtistName}", r.LimitString(meta.Data[0].Attributes.ArtistName),
				"{ArtistName}", r.LimitString(meta.Data[0].Attributes.ArtistName),
				"{ArtistId}", "",
			).Replace(r.Config.ArtistFolderFormat)
		}
		if strings.HasSuffix(singerFoldername, ".") {
			singerFoldername = strings.ReplaceAll(singerFoldername, ".", "")
		}
		singerFoldername = strings.TrimSpace(singerFoldername)
		fmt.Println(singerFoldername)
	}
	singerFolder := filepath.Join(r.Config.AlacSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	if r.Flags.Atmos {
		singerFolder = filepath.Join(r.Config.AtmosSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	}
	if r.Flags.AAC {
		singerFolder = filepath.Join(r.Config.AacSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	}
	if err := createDirectory(singerFolder); err != nil {
		return err
	}
	album.SaveDir = singerFolder
	var Quality string
	if strings.Contains(r.Config.AlbumFolderFormat, "Quality") {
		if r.Flags.Atmos {
			Quality = fmt.Sprintf("%dKbps", r.Config.AtmosMax-2000)
		} else if r.Flags.AAC && r.Config.AacType == "aac-lc" {
			Quality = "256Kbps"
		} else {
			manifest1, err := ampapi.GetSongResp(storefront, meta.Data[0].Relationships.Tracks.Data[0].ID, album.Language, token)
			if err != nil {
				fmt.Println("Failed to get manifest.\n", err)
			} else {
				if manifest1.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls == "" {
					Codec = "AAC"
					Quality = "256Kbps"
				} else {
					needCheck := false

					if r.Config.GetM3u8Mode == "all" {
						needCheck = true
					} else if r.Config.GetM3u8Mode == "hires" && contains(meta.Data[0].Relationships.Tracks.Data[0].Attributes.AudioTraits, "hi-res-lossless") {
						needCheck = true
					}
					var EnhancedHls_m3u8 string
					if needCheck {
						EnhancedHls_m3u8, _ = r.checkM3u8(meta.Data[0].Relationships.Tracks.Data[0].ID, "album")
						if strings.HasSuffix(EnhancedHls_m3u8, ".m3u8") {
							manifest1.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls = EnhancedHls_m3u8
						}
					}
					_, Quality, err = r.extractMedia(manifest1.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls, true)
					if err != nil {
						fmt.Println("Failed to extract quality from manifest.\n", err)
					}
				}
			}
		}
	}
	stringsToJoin := []string{}
	if meta.Data[0].Attributes.IsAppleDigitalMaster || meta.Data[0].Attributes.IsMasteredForItunes {
		if r.Config.AppleMasterChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.AppleMasterChoice)
		}
	}
	if meta.Data[0].Attributes.ContentRating == "explicit" {
		if r.Config.ExplicitChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.ExplicitChoice)
		}
	}
	if meta.Data[0].Attributes.ContentRating == "clean" {
		if r.Config.CleanChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.CleanChoice)
		}
	}
	Tag_string := strings.Join(stringsToJoin, " ")
	var albumFolderName string
	albumFolderName = strings.NewReplacer(
		"{ReleaseDate}", meta.Data[0].Attributes.ReleaseDate,
		"{ReleaseYear}", releaseYear(meta.Data[0].Attributes.ReleaseDate),
		"{ArtistName}", r.LimitString(meta.Data[0].Attributes.ArtistName),
		"{AlbumName}", r.LimitString(meta.Data[0].Attributes.Name),
		"{UPC}", meta.Data[0].Attributes.Upc,
		"{RecordLabel}", meta.Data[0].Attributes.RecordLabel,
		"{Copyright}", meta.Data[0].Attributes.Copyright,
		"{AlbumId}", albumId,
		"{Quality}", Quality,
		"{Codec}", Codec,
		"{Tag}", Tag_string,
	).Replace(r.Config.AlbumFolderFormat)

	albumFolderPath, err := r.prepareCollectionFolder(singerFolder, albumFolderName)
	if err != nil {
		return err
	}
	album.SaveName = albumFolderName
	fmt.Println(albumFolderName)
	if r.Config.SaveArtistCover && len(meta.Data[0].Relationships.Artists.Data) > 0 {
		if meta.Data[0].Relationships.Artists.Data[0].Attributes.Artwork.Url != "" {
			_, err = r.writeCover(singerFolder, "folder", meta.Data[0].Relationships.Artists.Data[0].Attributes.Artwork.Url)
			if err != nil {
				fmt.Println("Failed to write artist cover.")
			}
		}
	}
	covPath, err := r.writeCover(albumFolderPath, "cover", meta.Data[0].Attributes.Artwork.URL)
	if err != nil {
		fmt.Println("Failed to write cover.")
	}
	if r.Config.SaveAnimatedArtwork {
		r.saveAnimatedArtwork(
			albumFolderPath,
			meta.Data[0].Attributes.EditorialVideo.MotionDetailSquare.Video,
			meta.Data[0].Attributes.EditorialVideo.MotionDetailTall.Video,
		)
	}
	for i := range album.Tracks {
		album.Tracks[i].CoverPath = covPath
		album.Tracks[i].SaveDir = albumFolderPath
		album.Tracks[i].Codec = Codec
	}
	trackTotal := len(meta.Data[0].Relationships.Tracks.Data)
	arr := make([]int, trackTotal)
	for i := 0; i < trackTotal; i++ {
		arr[i] = i + 1
	}

	if urlArg_i != "" {
		for i := range album.Tracks {
			if urlArg_i == album.Tracks[i].ID {
				r.ripTrack(&album.Tracks[i], token, mediaUserToken)
				return nil
			}
		}
		return nil
	}
	var selected []int
	if !r.Flags.Select {
		selected = arr
	} else {
		selected = album.ShowSelect()
	}
	for i := range album.Tracks {
		i++
		if contains(r.State.OKDict[albumId], i) {
			r.State.Counter.Total++
			r.State.Counter.Success++
			continue
		}
		if contains(selected, i) {
			r.ripTrack(&album.Tracks[i-1], token, mediaUserToken)
		}
	}
	r.saveM3UPlaylist(albumFolderPath, albumFolderName)
	return nil

}

func (r *Runner) ripPlaylist(playlistId string, token string, storefront string, mediaUserToken string) error {
	playlist := model.NewPlaylist(storefront, playlistId)
	err := playlist.GetResp(token, r.Config.Language)
	if err != nil {
		fmt.Println("Failed to get playlist response.")
		return err
	}
	meta := playlist.Resp
	if r.Flags.Debug {
		fmt.Println(meta.Data[0].Attributes.ArtistName)
		fmt.Println(meta.Data[0].Attributes.Name)

		for trackNum, track := range meta.Data[0].Relationships.Tracks.Data {
			trackNum++
			fmt.Printf("\nTrack %d of %d:\n", trackNum, len(meta.Data[0].Relationships.Tracks.Data))
			fmt.Printf("%02d. %s\n", trackNum, track.Attributes.Name)

			manifest, err := ampapi.GetSongResp(storefront, track.ID, playlist.Language, token)
			if err != nil {
				fmt.Printf("Failed to get manifest for track %d: %v\n", trackNum, err)
				continue
			}

			var m3u8Url string
			if manifest.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls != "" {
				m3u8Url = manifest.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls
			}
			needCheck := false
			if r.Config.GetM3u8Mode == "all" {
				needCheck = true
			} else if r.Config.GetM3u8Mode == "hires" && contains(track.Attributes.AudioTraits, "hi-res-lossless") {
				needCheck = true
			}
			if needCheck {
				fullM3u8Url, err := r.checkM3u8(track.ID, "song")
				if err == nil && strings.HasSuffix(fullM3u8Url, ".m3u8") {
					m3u8Url = fullM3u8Url
				} else {
					fmt.Println("Failed to get best quality m3u8 from lite-server, will use m3u8 from Web API")
				}
			}

			_, _, err = r.extractMedia(m3u8Url, true)
			if err != nil {
				fmt.Printf("Failed to extract quality info for track %d: %v\n", trackNum, err)
				continue
			}
		}
		return nil
	}
	var Codec string
	if r.Flags.Atmos {
		Codec = "ATMOS"
	} else if r.Flags.AAC {
		Codec = "AAC"
	} else {
		Codec = "ALAC"
	}
	playlist.Codec = Codec
	var singerFoldername string
	if r.Config.ArtistFolderFormat != "" {
		singerFoldername = strings.NewReplacer(
			"{ArtistName}", "Apple Music",
			"{ArtistId}", "",
			"{UrlArtistName}", "Apple Music",
		).Replace(r.Config.ArtistFolderFormat)
		if strings.HasSuffix(singerFoldername, ".") {
			singerFoldername = strings.ReplaceAll(singerFoldername, ".", "")
		}
		singerFoldername = strings.TrimSpace(singerFoldername)
		fmt.Println(singerFoldername)
	}
	singerFolder := filepath.Join(r.Config.AlacSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	if r.Flags.Atmos {
		singerFolder = filepath.Join(r.Config.AtmosSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	}
	if r.Flags.AAC {
		singerFolder = filepath.Join(r.Config.AacSaveFolder, forbiddenNames.ReplaceAllString(singerFoldername, "_"))
	}
	if err := createDirectory(singerFolder); err != nil {
		return err
	}
	playlist.SaveDir = singerFolder

	var Quality string
	if strings.Contains(r.Config.AlbumFolderFormat, "Quality") {
		if r.Flags.Atmos {
			Quality = fmt.Sprintf("%dKbps", r.Config.AtmosMax-2000)
		} else if r.Flags.AAC && r.Config.AacType == "aac-lc" {
			Quality = "256Kbps"
		} else {
			manifest1, err := ampapi.GetSongResp(storefront, meta.Data[0].Relationships.Tracks.Data[0].ID, playlist.Language, token)
			if err != nil {
				fmt.Println("Failed to get manifest.\n", err)
			} else {
				if manifest1.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls == "" {
					Codec = "AAC"
					Quality = "256Kbps"
				} else {
					needCheck := false

					if r.Config.GetM3u8Mode == "all" {
						needCheck = true
					} else if r.Config.GetM3u8Mode == "hires" && contains(meta.Data[0].Relationships.Tracks.Data[0].Attributes.AudioTraits, "hi-res-lossless") {
						needCheck = true
					}
					var EnhancedHls_m3u8 string
					if needCheck {
						EnhancedHls_m3u8, _ = r.checkM3u8(meta.Data[0].Relationships.Tracks.Data[0].ID, "album")
						if strings.HasSuffix(EnhancedHls_m3u8, ".m3u8") {
							manifest1.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls = EnhancedHls_m3u8
						}
					}
					_, Quality, err = r.extractMedia(manifest1.Data[0].Attributes.ExtendedAssetUrls.EnhancedHls, true)
					if err != nil {
						fmt.Println("Failed to extract quality from manifest.\n", err)
					}
				}
			}
		}
	}
	stringsToJoin := []string{}
	if meta.Data[0].Attributes.IsAppleDigitalMaster || meta.Data[0].Attributes.IsMasteredForItunes {
		if r.Config.AppleMasterChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.AppleMasterChoice)
		}
	}
	if meta.Data[0].Attributes.ContentRating == "explicit" {
		if r.Config.ExplicitChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.ExplicitChoice)
		}
	}
	if meta.Data[0].Attributes.ContentRating == "clean" {
		if r.Config.CleanChoice != "" {
			stringsToJoin = append(stringsToJoin, r.Config.CleanChoice)
		}
	}
	Tag_string := strings.Join(stringsToJoin, " ")
	playlistFolder := strings.NewReplacer(
		"{ArtistName}", "Apple Music",
		"{PlaylistName}", r.LimitString(meta.Data[0].Attributes.Name),
		"{PlaylistId}", playlistId,
		"{Quality}", Quality,
		"{Codec}", Codec,
		"{Tag}", Tag_string,
	).Replace(r.Config.PlaylistFolderFormat)
	playlistFolderPath, err := r.prepareCollectionFolder(singerFolder, playlistFolder)
	if err != nil {
		return err
	}
	playlist.SaveName = playlistFolder
	fmt.Println(playlistFolder)
	covPath, err := r.writeCover(playlistFolderPath, "cover", meta.Data[0].Attributes.Artwork.URL)
	if err != nil {
		fmt.Println("Failed to write cover.")
	}

	for i := range playlist.Tracks {
		playlist.Tracks[i].CoverPath = covPath
		playlist.Tracks[i].SaveDir = playlistFolderPath
		playlist.Tracks[i].Codec = Codec
	}

	if r.Config.SaveAnimatedArtwork {
		r.saveAnimatedArtwork(
			playlistFolderPath,
			meta.Data[0].Attributes.EditorialVideo.MotionDetailSquare.Video,
			meta.Data[0].Attributes.EditorialVideo.MotionDetailTall.Video,
		)
	}
	trackTotal := len(meta.Data[0].Relationships.Tracks.Data)
	arr := make([]int, trackTotal)
	for i := 0; i < trackTotal; i++ {
		arr[i] = i + 1
	}
	var selected []int

	if !r.Flags.Select {
		selected = arr
	} else {
		selected = playlist.ShowSelect()
	}
	for i := range playlist.Tracks {
		i++
		if contains(r.State.OKDict[playlistId], i) {
			r.State.Counter.Total++
			r.State.Counter.Success++
			continue
		}
		if contains(selected, i) {
			r.ripTrack(&playlist.Tracks[i-1], token, mediaUserToken)
		}
	}
	r.saveM3UPlaylist(playlistFolderPath, playlistFolder)
	return nil
}

func (r *Runner) ripSong(songId string, token string, storefront string, mediaUserToken string) error {
	// Get song info to find album ID
	manifest, err := ampapi.GetSongResp(storefront, songId, r.Config.Language, token)
	if err != nil {
		fmt.Println("Failed to get song response.")
		return err
	}

	songData := manifest.Data[0]
	albumId := songData.Relationships.Albums.Data[0].ID

	// Use album approach but only download the specific song
	err = r.ripAlbum(albumId, token, storefront, mediaUserToken, songId)
	if err != nil {
		fmt.Println("Failed to rip song:", err)
		return err
	}

	return nil
}
