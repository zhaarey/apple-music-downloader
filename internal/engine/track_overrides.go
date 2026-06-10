package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"main/utils/ampapi"
	"main/utils/task"
)

type TrackURLOverride struct {
	Num int    `json:"num"`
	URL string `json:"url"`
}

var guiTrackURLOverrides map[int]string

func resetTrackURLOverrides() {
	guiTrackURLOverrides = nil
}

func applyTrackURLOverrides(overrides []TrackURLOverride) {
	resetTrackURLOverrides()
	if len(overrides) == 0 {
		return
	}
	guiTrackURLOverrides = make(map[int]string, len(overrides))
	for _, o := range overrides {
		url := strings.TrimSpace(o.URL)
		if o.Num > 0 && url != "" {
			guiTrackURLOverrides[o.Num] = url
		}
	}
}

func playlistTrackURLOverride(taskNum int) (string, bool) {
	if guiTrackURLOverrides == nil {
		return "", false
	}
	url, ok := guiTrackURLOverrides[taskNum]
	return url, ok && strings.TrimSpace(url) != ""
}

func songDataToTrackData(s ampapi.SongRespData) (ampapi.TrackRespData, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return ampapi.TrackRespData{}, err
	}
	var t ampapi.TrackRespData
	if err := json.Unmarshal(raw, &t); err != nil {
		return ampapi.TrackRespData{}, err
	}
	return t, nil
}

func applyPlaylistTrackURLOverride(track *task.Track, songURL, token, language string) error {
	songURL = normalizeAppleCatalogURL(strings.TrimSpace(songURL))
	if songURL == "" {
		return nil
	}
	storefront, songID := checkUrlSong(songURL)
	if songID == "" {
		return fmt.Errorf("invalid song URL for track %d", track.TaskNum)
	}
	resp, err := ampapi.GetSongResp(storefront, songID, language, token)
	if err != nil || len(resp.Data) == 0 {
		return fmt.Errorf("could not load song metadata for track %d", track.TaskNum)
	}
	trackResp, err := songDataToTrackData(resp.Data[0])
	if err != nil {
		return err
	}
	track.ID = trackResp.ID
	track.Type = trackResp.Type
	track.Name = trackResp.Attributes.Name
	track.Storefront = storefront
	track.Resp = trackResp
	track.M3u8 = trackResp.Attributes.ExtendedAssetUrls.EnhancedHls
	track.WebM3u8 = track.M3u8
	return nil
}
