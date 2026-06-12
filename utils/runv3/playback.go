package runv3

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const aacFlavor = "28:ctrp256"

// WebPlayback holds Widevine metadata and encrypted fMP4 byte ranges.
type WebPlayback struct {
	KidBase64 string
	UriPrefix string
	Parts     []FMP4Part
}

type songlistResponse struct {
	List []struct {
		HlsPlaylistURL string `json:"hls-playlist-url"`
		Assets         []struct {
			Flavor string `json:"flavor"`
			URL    string `json:"URL"`
		} `json:"assets"`
	} `json:"songList"`
}

func postWebPlayback(adamID, authToken, muToken string) (*songlistResponse, error) {
	const url = "https://play.music.apple.com/WebObjects/MZPlay.woa/wa/webPlayback"
	payload, err := json.Marshal(map[string]string{"salableAdamId": adamID})
	if err != nil {
		return nil, wrapStage(StagePlayback, err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, wrapStage(StagePlayback, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("User-Agent", cdnHeaders["User-Agent"])
	req.Header.Set("Referer", "https://music.apple.com/")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	req.Header.Set("x-apple-music-user-token", muToken)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, wrapStage(StagePlayback, err)
	}
	defer resp.Body.Close()

	var obj songlistResponse
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return nil, wrapStage(StagePlayback, err)
	}
	return &obj, nil
}

// GetSongWebPlayback resolves AAC-LC stream parts for a track adam ID.
func GetSongWebPlayback(adamID, authToken, muToken string) (WebPlayback, error) {
	obj, err := postWebPlayback(adamID, authToken, muToken)
	if err != nil {
		return WebPlayback{}, err
	}
	if len(obj.List) == 0 {
		return WebPlayback{}, errors.New("Unavailable")
	}
	for _, asset := range obj.List[0].Assets {
		if asset.Flavor != aacFlavor {
			continue
		}
		return parsePlaybackPlaylist(asset.URL)
	}
	return WebPlayback{}, errors.New("Unavailable")
}

// GetMVPlaylistURL returns the HLS playlist URL for music-video license acquisition.
func GetMVPlaylistURL(adamID, authToken, muToken string) (string, error) {
	obj, err := postWebPlayback(adamID, authToken, muToken)
	if err != nil {
		return "", err
	}
	if len(obj.List) == 0 || obj.List[0].HlsPlaylistURL == "" {
		return "", errors.New("Unavailable")
	}
	return obj.List[0].HlsPlaylistURL, nil
}
