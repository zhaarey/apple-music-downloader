package ampapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

func GetStationResp(storefront string, id string, language string, token string) (*StationResp, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/stations/%s", storefront, id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	query := url.Values{}
	query.Set("omit[resource]", "autos")
	query.Set("extend", "editorialVideo")
	query.Set("l", language)
	req.URL.RawQuery = query.Encode()
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return nil, errors.New(do.Status)
	}
	obj := new(StationResp)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func GetStationAssetsUrlAndServerUrl(id string, mutoken string, token string) (string, string, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return "", "", err
		}
	}

	req, err := http.NewRequest("GET", "https://amp-api.music.apple.com/v1/play/assets", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("Media-User-Token", mutoken)
	query := url.Values{}
	//query.Set("omit[resource]", "autos")
	//query.Set("extend", "editorialVideo")
	query.Set("id", id)
	query.Set("kind", "radioStation")
	query.Set("keyFormat", "web")
	req.URL.RawQuery = query.Encode()
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return "", "", errors.New(do.Status)
	}
	obj := new(StationAssets)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return "", "", err
	}
	return obj.Results.Assets[0].Url, obj.Results.Assets[0].KeyServerUrl, nil
}

func GetStationNextTracks(id, mutoken, language, token string) (*TrackResp, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://amp-api.music.apple.com/v1/me/stations/next-tracks/%s", id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("Media-User-Token", mutoken)
	query := url.Values{}
	query.Set("omit[resource]", "autos")
	//query.Set("include", "tracks,artists,record-labels")
	query.Set("include[songs]", "artists,albums")
	query.Set("limit", "10")
	query.Set("extend", "editorialVideo,extendedAssetUrls")
	query.Set("l", language)
	req.URL.RawQuery = query.Encode()
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return nil, errors.New(do.Status)
	}
	obj := new(TrackResp)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

type StationResp struct {
	Href string            `json:"href"`
	Next string            `json:"next"`
	Data []StationRespData `json:"data"`
}

type StationAssets struct {
	Results struct {
		Assets []struct {
			KeyServerUrl              string `json:"keyServerUrl"`
			Url                       string `json:"url"`
			WidevineKeyCertificateUrl string `json:"widevineKeyCertificateUrl"`
			FairPlayKeyCertificateUrl string `json:"fairPlayKeyCertificateUrl"`
		} `json:"assets"`
	} `json:"results"`
}

type StationRespData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Href       string `json:"href"`
	Attributes struct {
		Artwork struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			URL        string `json:"url"`
			BgColor    string `json:"bgColor"`
			TextColor1 string `json:"textColor1"`
			TextColor2 string `json:"textColor2"`
			TextColor3 string `json:"textColor3"`
			TextColor4 string `json:"textColor4"`
		} `json:"artwork"`
		IsLive         bool   `json:"isLive"`
		URL            string `json:"url"`
		Name           string `json:"name"`
		EditorialVideo struct {
			MotionTall struct {
				Video string `json:"video"`
			} `json:"motionTallVideo3x4"`
			MotionSquare struct {
				Video string `json:"video"`
			} `json:"motionSquareVideo1x1"`
			MotionDetailTall struct {
				Video string `json:"video"`
			} `json:"motionDetailTall"`
			MotionDetailSquare struct {
				Video string `json:"video"`
			} `json:"motionDetailSquare"`
		} `json:"editorialVideo"`
		PlayParams struct {
			ID          string `json:"id"`
			Kind        string `json:"kind"`
			Format      string `json:"format"`
			StationHash string `json:"stationHash"`
		} `json:"playParams"`
	} `json:"attributes"`
}
