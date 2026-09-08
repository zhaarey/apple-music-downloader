package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grafov/m3u8"
	"github.com/olekukonko/tablewriter"
	"github.com/zhaarey/go-mp4tag"
	"amdl/internal/model"
	"amdl/internal/download"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func (r *Runner) writeCover(sanAlbumFolder, name string, url string) (string, error) {
	originalUrl := url
	var ext string
	var covPath string
	if r.Config.CoverFormat == "original" {
		ext = strings.Split(url, "/")[len(strings.Split(url, "/"))-2]
		ext = ext[strings.LastIndex(ext, ".")+1:]
		covPath = filepath.Join(sanAlbumFolder, name+"."+ext)
	} else {
		covPath = filepath.Join(sanAlbumFolder, name+"."+r.Config.CoverFormat)
	}
	if r.Config.CoverFormat == "png" {
		re := regexp.MustCompile(`\{w\}x\{h\}`)
		parts := re.Split(url, 2)
		url = parts[0] + "{w}x{h}" + strings.Replace(parts[1], ".jpg", ".png", 1)
	}
	url = strings.Replace(url, "{w}x{h}", r.Config.CoverSize, 1)
	if r.Config.CoverFormat == "original" {
		url = strings.Replace(url, "is1-ssl.mzstatic.com/image/thumb", "a5.mzstatic.com/us/r1000/0", 1)
		url = url[:strings.LastIndex(url, "/")]
	}

	tmpFile, err := os.CreateTemp(sanAlbumFolder, name+".tmp-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	if err := r.getCover(url, tmpFile); err != nil {
		if r.Config.CoverFormat != "original" {
			return "", err
		}
		splitByDot := strings.Split(originalUrl, ".")
		last := splitByDot[len(splitByDot)-1]
		fallback := originalUrl[:len(originalUrl)-len(last)] + ext
		fallback = strings.Replace(fallback, "{w}x{h}", r.Config.CoverSize, 1)
		if err := r.getCover(fallback, tmpFile); err != nil {
			return "", err
		}
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, covPath); err != nil {
		return "", err
	}
	cleanup = false
	return covPath, nil
}

func (r *Runner) getCover(url string, out *os.File) error {
	if err := out.Truncate(0); err != nil {
		return err
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		return err
	}
	resp, err := download.GetWithHeaders(url, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

func (r *Runner) writeLyrics(sanAlbumFolder, filename string, lrc string) error {
	lyricspath := filepath.Join(sanAlbumFolder, filename)
	f, err := os.Create(lyricspath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(lrc)
	if err != nil {
		return err
	}
	return nil
}

func (r *Runner) writeM3UPlaylist(folderPath string, name string, tracks []AddedTrack) error {
	if !r.Flags.SaveM3U8 {
		return nil
	}
	m3uPath := filepath.Join(folderPath, sanitizeFolderName(name)+".m3u8")
	f, err := os.Create(m3uPath)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintln(f, "#EXTM3U")
	for _, track := range tracks {
		fmt.Fprintf(f, "#EXTINF:-1,%s - %s\n", track.Artist, track.Song)
		fmt.Fprintln(f, filepath.Base(track.Path))
	}
	return nil
}

func firstGenre(genres []string) string {
	if len(genres) == 0 {
		return ""
	}
	return genres[0]
}

func (r *Runner) writeMP4Tags(track *model.Track, lrc string) error {
	t := &mp4tag.MP4Tags{
		Title:  track.Resp.Attributes.Name,
		Artist: track.Resp.Attributes.ArtistName,
		Custom: map[string]string{
			"PERFORMER":   track.Resp.Attributes.ArtistName,
			"RELEASETIME": track.Resp.Attributes.ReleaseDate,
			"ISRC":        track.Resp.Attributes.Isrc,
			"LABEL":       "",
			"UPC":         "",
		},
		Composer:    track.Resp.Attributes.ComposerName,
		CustomGenre: firstGenre(track.Resp.Attributes.GenreNames),
		Lyrics:      lrc,
		TrackNumber: int16(track.Resp.Attributes.TrackNumber),
		DiscNumber:  int16(track.Resp.Attributes.DiscNumber),
		Album:       track.Resp.Attributes.AlbumName,
	}

	if r.Config.TagSortOrder {
		t.TitleSort = track.Resp.Attributes.Name
		t.ArtistSort = track.Resp.Attributes.ArtistName
		t.ComposerSort = track.Resp.Attributes.ComposerName
		t.AlbumSort = track.Resp.Attributes.AlbumName
	}

	if r.Config.TagItunesID {
		if track.PreType == "albums" {
			albumID, err := strconv.ParseUint(track.PreID, 10, 64)
			if err != nil {
				return err
			}
			t.ItunesAlbumID = int32(albumID)
		}

		if len(track.Resp.Relationships.Artists.Data) > 0 {
			artistID, err := strconv.ParseUint(track.Resp.Relationships.Artists.Data[0].ID, 10, 64)
			if err != nil {
				return err
			}
			t.ItunesArtistID = int32(artistID)
		}
	}

	if (track.PreType == "playlists" || track.PreType == "stations") && !r.Config.UseSongInfoForPlaylist {
		t.DiscNumber = 1
		t.DiscTotal = 1
		t.TrackNumber = int16(track.TaskNum)
		t.TrackTotal = int16(track.TaskTotal)
		t.Album = track.PlaylistData.Attributes.Name
		t.AlbumArtist = track.PlaylistData.Attributes.ArtistName
		if r.Config.TagSortOrder {
			t.AlbumSort = track.PlaylistData.Attributes.Name
			t.AlbumArtistSort = track.PlaylistData.Attributes.ArtistName
		}
	} else if (track.PreType == "playlists" || track.PreType == "stations") && r.Config.UseSongInfoForPlaylist {
		t.DiscTotal = int16(track.DiscTotal)
		t.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
		t.AlbumArtist = track.AlbumData.Attributes.ArtistName
		t.Custom["UPC"] = track.AlbumData.Attributes.Upc
		t.Custom["LABEL"] = track.AlbumData.Attributes.RecordLabel
		t.Date = track.AlbumData.Attributes.ReleaseDate
		t.Copyright = track.AlbumData.Attributes.Copyright
		t.Publisher = track.AlbumData.Attributes.RecordLabel
		if r.Config.TagSortOrder {
			t.AlbumArtistSort = track.AlbumData.Attributes.ArtistName
		}
	} else {
		t.DiscTotal = int16(track.DiscTotal)
		t.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
		t.AlbumArtist = track.AlbumData.Attributes.ArtistName
		t.Custom["UPC"] = track.AlbumData.Attributes.Upc
		t.Date = track.AlbumData.Attributes.ReleaseDate
		t.Copyright = track.AlbumData.Attributes.Copyright
		t.Publisher = track.AlbumData.Attributes.RecordLabel
		if r.Config.TagSortOrder {
			t.AlbumArtistSort = track.AlbumData.Attributes.ArtistName
		}
	}

	if track.Resp.Attributes.ContentRating == "explicit" {
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryExplicit
	} else if track.Resp.Attributes.ContentRating == "clean" {
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryClean
	} else {
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryNone
	}

	mp4, err := mp4tag.Open(track.SavePath)
	if err != nil {
		return err
	}
	defer mp4.Close()
	err = mp4.Write(t, []string{})
	if err != nil {
		return err
	}
	return nil
}

func (r *Runner) extractMvAudio(c string) (string, error) {
	mediaURL, err := url.Parse(c)
	if err != nil {
		return "", err
	}

	resp, err := download.Get(c)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	from, listType, err := m3u8.DecodeFrom(strings.NewReader(string(body)), true)
	if err != nil || listType != m3u8.MASTER {
		return "", errors.New("m3u8 not of media type")
	}

	master := from.(*m3u8.MasterPlaylist)

	type audioStream struct {
		url     string
		groupID string
		score   int
		rank    int
	}
	var streams []audioStream
	rankRe := regexp.MustCompile(`_gr(\d+)_`)

	for _, variant := range master.Variants {
		for _, alt := range variant.Alternatives {
			if alt.URI == "" {
				continue
			}
			score := audioScore(alt.GroupId, r.Config.MVAudioType)
			if score < 0 {
				continue
			}
			var rank int
			if m := rankRe.FindStringSubmatch(alt.URI); len(m) == 2 {
				rank, _ = strconv.Atoi(m[1])
			}
			streamURL, _ := mediaURL.Parse(alt.URI)
			streams = append(streams, audioStream{
				url:     streamURL.String(),
				groupID: alt.GroupId,
				score:   score,
				rank:    rank,
			})
		}
	}

	if len(streams) == 0 {
		return "", errors.New("no suitable audio stream found")
	}

	sort.Slice(streams, func(i, j int) bool {
		if streams[i].score != streams[j].score {
			return streams[i].score > streams[j].score
		}
		return streams[i].rank > streams[j].rank
	})
	fmt.Println("Audio: " + streams[0].groupID)
	return streams[0].url, nil
}

// audioScore rates an audio group ID by quality. Higher is better; -1 means
// the format is not allowed for the selected mv-audio-type.
func audioScore(groupID, preferredType string) int {
	switch {
	case groupID == "audio-atmos":
		if preferredType == "ac3" || preferredType == "aac" {
			return -1
		}
		return 10000
	case groupID == "audio-ac3":
		if preferredType == "aac" {
			return -1
		}
		if preferredType == "ac3" {
			return 10000
		}
		return 9000
	case strings.HasPrefix(groupID, "audio-stereo-"):
		// LC-AAC stereo, e.g. audio-stereo-256.
		return 8000 + parseAudioBitrate(groupID, "audio-stereo-")
	case strings.HasPrefix(groupID, "audio-HE-stereo-"):
		// HE-AAC stereo, e.g. audio-HE-stereo-64.
		return 7000 + parseAudioBitrate(groupID, "audio-HE-stereo-")
	case strings.HasPrefix(groupID, "audio-HE2-stereo-"):
		// HE-AAC v2 stereo, e.g. audio-HE2-stereo-32.
		return 6000 + parseAudioBitrate(groupID, "audio-HE2-stereo-")
	default:
		return -1
	}
}

func parseAudioBitrate(groupID, prefix string) int {
	bitrate, err := strconv.Atoi(strings.TrimPrefix(groupID, prefix))
	if err != nil {
		return 0
	}
	return bitrate
}

func (r *Runner) checkM3u8(b string, f string) (string, error) {
	var EnhancedHls string
	if r.Config.LiteServer == "" {
		return "", errors.New("lite-server is not configured")
	}
	endpoint := strings.TrimRight(r.Config.LiteServer, "/") + "/m3u8?adamId=" + url.QueryEscape(b)
	resp, err := download.Get(endpoint)
	if err != nil {
		fmt.Println("Error connecting to lite-server:", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	var envelope struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			M3u8 string `json:"m3u8"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}
	if envelope.Code != 0 {
		return "", fmt.Errorf("lite-server /m3u8 returned code=%d msg=%s", envelope.Code, envelope.Msg)
	}
	EnhancedHls = envelope.Data.M3u8
	if f == "song" {
		if EnhancedHls != "" {
			fmt.Println("Received URL:", EnhancedHls)
		} else {
			fmt.Println("Received an empty response")
		}
	}
	return EnhancedHls, nil
}

func formatAvailability(available bool, quality string) string {
	if !available {
		return "Not Available"
	}
	return quality
}

func (r *Runner) extractMedia(b string, more_mode bool) (string, string, error) {
	masterUrl, err := url.Parse(b)
	if err != nil {
		return "", "", err
	}
	resp, err := download.Get(b)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	masterString := string(body)
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(masterString), true)
	if err != nil || listType != m3u8.MASTER {
		return "", "", errors.New("m3u8 not of master type")
	}
	master := from.(*m3u8.MasterPlaylist)
	var streamUrl *url.URL
	sort.Slice(master.Variants, func(i, j int) bool {
		return master.Variants[i].AverageBandwidth > master.Variants[j].AverageBandwidth
	})
	if r.Flags.Debug && more_mode {
		fmt.Println("\nDebug: All Available Variants:")
		var data [][]string
		for _, variant := range master.Variants {
			data = append(data, []string{variant.Codecs, variant.Audio, fmt.Sprint(variant.Bandwidth)})
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Codec", "Audio", "Bandwidth"})
		table.SetAutoMergeCells(true)
		table.SetRowLine(true)
		table.AppendBulk(data)
		table.Render()

		var hasAAC, hasLossless, hasHiRes, hasAtmos, hasDolbyAudio bool
		var aacQuality, losslessQuality, hiResQuality, atmosQuality, dolbyAudioQuality string

		for _, variant := range master.Variants {
			if variant.Codecs == "mp4a.40.2" { // AAC
				hasAAC = true
				split := strings.Split(variant.Audio, "-")
				if len(split) >= 3 {
					bitrate, _ := strconv.Atoi(split[2])
					currentBitrate := 0
					if aacQuality != "" {
						current := strings.Split(aacQuality, " | ")[2]
						current = strings.Split(current, " ")[0]
						currentBitrate, _ = strconv.Atoi(current)
					}
					if bitrate > currentBitrate {
						aacQuality = fmt.Sprintf("AAC | 2 Channel | %d Kbps", bitrate)
					}
				}
			} else if variant.Codecs == "ec-3" && strings.Contains(variant.Audio, "atmos") { // Dolby Atmos
				hasAtmos = true
				split := strings.Split(variant.Audio, "-")
				if len(split) > 0 {
					bitrateStr := split[len(split)-1]
					if len(bitrateStr) == 4 && bitrateStr[0] == '2' {
						bitrateStr = bitrateStr[1:]
					}
					bitrate, _ := strconv.Atoi(bitrateStr)
					currentBitrate := 0
					if atmosQuality != "" {
						current := strings.Split(strings.Split(atmosQuality, " | ")[2], " ")[0]
						currentBitrate, _ = strconv.Atoi(current)
					}
					if bitrate > currentBitrate {
						atmosQuality = fmt.Sprintf("E-AC-3 | 16 Channel | %d Kbps", bitrate)
					}
				}
			} else if variant.Codecs == "alac" { // ALAC (Lossless or Hi-Res)
				split := strings.Split(variant.Audio, "-")
				if len(split) >= 3 {
					bitDepth := split[len(split)-1]
					sampleRate := split[len(split)-2]
					sampleRateInt, _ := strconv.Atoi(sampleRate)
					if sampleRateInt > 48000 { // Hi-Res
						hasHiRes = true
						hiResQuality = fmt.Sprintf("ALAC | 2 Channel | %s-bit/%d kHz", bitDepth, sampleRateInt/1000)
					} else { // Standard Lossless
						hasLossless = true
						losslessQuality = fmt.Sprintf("ALAC | 2 Channel | %s-bit/%d kHz", bitDepth, sampleRateInt/1000)
					}
				}
			} else if variant.Codecs == "ac-3" { // Dolby Audio
				hasDolbyAudio = true
				split := strings.Split(variant.Audio, "-")
				if len(split) > 0 {
					bitrate, _ := strconv.Atoi(split[len(split)-1])
					dolbyAudioQuality = fmt.Sprintf("AC-3 |  16 Channel | %d Kbps", bitrate)
				}
			}
		}

		fmt.Println("Available Audio Formats:")
		fmt.Println("------------------------")
		fmt.Printf("AAC             : %s\n", formatAvailability(hasAAC, aacQuality))
		fmt.Printf("Lossless        : %s\n", formatAvailability(hasLossless, losslessQuality))
		fmt.Printf("Hi-Res Lossless : %s\n", formatAvailability(hasHiRes, hiResQuality))
		fmt.Printf("Dolby Atmos     : %s\n", formatAvailability(hasAtmos, atmosQuality))
		fmt.Printf("Dolby Audio     : %s\n", formatAvailability(hasDolbyAudio, dolbyAudioQuality))
		fmt.Println("------------------------")

		fmt.Printf("%+v\n", r.Config)
		fmt.Println("===== SELECTOR =====")
		for _, variant := range master.Variants {
			fmt.Printf("Codec=%q Audio=%q AvgBW=%d BW=%d\n",
				variant.Codecs,
				variant.Audio,
				variant.AverageBandwidth,
				variant.Bandwidth,
			)
		}
		fmt.Printf("r.Flags.Atmos=%v r.Flags.AAC=%v AlacMax=%d\n",
			r.Flags.Atmos,
			r.Flags.AAC,
			r.Config.AlacMax,
		)
		fmt.Println("====================")

		return "", "", nil
	}
	var Quality string
	for _, variant := range master.Variants {
		if r.Flags.Atmos {
			if variant.Codecs == "ec-3" && strings.Contains(variant.Audio, "atmos") {
				if r.Flags.Debug && !more_mode {
					fmt.Printf("Debug: Found Dolby Atmos variant - %s (Bitrate: %d Kbps)\n",
						variant.Audio, variant.Bandwidth/1000)
				}
				split := strings.Split(variant.Audio, "-")
				length := len(split)
				length_int, err := strconv.Atoi(split[length-1])
				if err != nil {
					return "", "", err
				}
				if length_int <= r.Config.AtmosMax {
					if !r.Flags.Debug && !more_mode {
						fmt.Printf("%s\n", variant.Audio)
					}
					streamUrlTemp, err := masterUrl.Parse(variant.URI)
					if err != nil {
						return "", "", err
					}
					streamUrl = streamUrlTemp
					Quality = fmt.Sprintf("%s Kbps", split[len(split)-1])
					break
				}
			} else if variant.Codecs == "ac-3" { // Add Dolby Audio support
				if r.Flags.Debug && !more_mode {
					fmt.Printf("Debug: Found Dolby Audio variant - %s (Bitrate: %d Kbps)\n",
						variant.Audio, variant.Bandwidth/1000)
				}
				streamUrlTemp, err := masterUrl.Parse(variant.URI)
				if err != nil {
					return "", "", err
				}
				streamUrl = streamUrlTemp
				split := strings.Split(variant.Audio, "-")
				Quality = fmt.Sprintf("%s Kbps", split[len(split)-1])
				break
			}
		} else if r.Flags.AAC {
			if variant.Codecs == "mp4a.40.2" {
				if r.Flags.Debug && !more_mode {
					fmt.Printf("Debug: Found AAC variant - %s (Bitrate: %d)\n", variant.Audio, variant.Bandwidth)
				}
				aacregex := regexp.MustCompile(`audio-stereo-\d+`)
				replaced := aacregex.ReplaceAllString(variant.Audio, "aac")
				if replaced == r.Config.AacType {
					if !r.Flags.Debug && !more_mode {
						fmt.Printf("%s\n", variant.Audio)
					}
					streamUrlTemp, err := masterUrl.Parse(variant.URI)
					if err != nil {
						return "", "", err
					}
					streamUrl = streamUrlTemp
					split := strings.Split(variant.Audio, "-")
					Quality = fmt.Sprintf("%s Kbps", split[2])
					break
				}
			}
		} else {
			if variant.Codecs == "alac" {
				split := strings.Split(variant.Audio, "-")
				length := len(split)
				length_int, err := strconv.Atoi(split[length-2])
				if err != nil {
					return "", "", err
				}
				max := r.Config.AlacMax
				if max == 0 {
					max = 192000
				}
				if length_int <= r.Config.AlacMax {
					if !r.Flags.Debug && !more_mode {
						fmt.Printf("%s-bit / %s Hz\n", split[length-1], split[length-2])
					}
					streamUrlTemp, err := masterUrl.Parse(variant.URI)
					if err != nil {
						return "", "", err
					}
					streamUrl = streamUrlTemp
					KHZ := float64(length_int) / 1000.0
					Quality = fmt.Sprintf("%sB-%.1fkHz", split[length-1], KHZ)
					break
				}
			}
		}
	}
	if streamUrl == nil {
		return "", "", errors.New("no codec found")
	}
	return streamUrl.String(), Quality, nil
}

func (r *Runner) extractVideo(c string) (string, error) {
	MediaUrl, err := url.Parse(c)
	if err != nil {
		return "", err
	}

	resp, err := download.Get(c)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	videoString := string(body)

	from, listType, err := m3u8.DecodeFrom(strings.NewReader(videoString), true)
	if err != nil || listType != m3u8.MASTER {
		return "", errors.New("m3u8 not of media type")
	}

	video := from.(*m3u8.MasterPlaylist)

	re := regexp.MustCompile(`_(\d+)x(\d+)`)

	var streamUrl *url.URL
	sort.Slice(video.Variants, func(i, j int) bool {
		return video.Variants[i].AverageBandwidth > video.Variants[j].AverageBandwidth
	})

	maxHeight := r.Config.MVMax

	for _, variant := range video.Variants {
		matches := re.FindStringSubmatch(variant.URI)
		if len(matches) == 3 {
			height := matches[2]
			var h int
			_, err := fmt.Sscanf(height, "%d", &h)
			if err != nil {
				continue
			}
			if h <= maxHeight {
				streamUrl, err = MediaUrl.Parse(variant.URI)
				if err != nil {
					return "", err
				}
				fmt.Println("Video: " + variant.Resolution + "-" + variant.VideoRange)
				break
			}
		}
	}

	if streamUrl == nil {
		return "", errors.New("no suitable video stream found")
	}

	return streamUrl.String(), nil
}
