package ampapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func GetAlbumResp(storefront string, id string, language string, token string) (*AlbumResp, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/albums/%s", storefront, id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	query := url.Values{}
	query.Set("omit[resource]", "autos")
	query.Set("include", "tracks,artists,record-labels")
	query.Set("include[songs]", "artists")
	//query.Set("fields[artists]", "name,artwork")
	//query.Set("fields[albums:albums]", "artistName,artwork,name,releaseDate,url")
	//query.Set("fields[record-labels]", "name")
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
	obj := new(AlbumResp)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return nil, err
	}
	if len(obj.Data[0].Relationships.Tracks.Next) > 0 {
		next := obj.Data[0].Relationships.Tracks.Next
		for {
			req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com%s", next), nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
			req.Header.Set("Origin", "https://music.apple.com")
			query := req.URL.Query()
			query.Set("omit[resource]", "autos")
			query.Set("include", "artists")
			query.Set("extend", "editorialVideo,extendedAssetUrls")
			req.URL.RawQuery = query.Encode()
			do, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer do.Body.Close()
			if do.StatusCode != http.StatusOK {
				return nil, errors.New(do.Status)
			}
			obj2 := new(TrackResp)
			err = json.NewDecoder(do.Body).Decode(&obj2)
			if err != nil {
				return nil, err
			}
			obj.Data[0].Relationships.Tracks.Data = append(obj.Data[0].Relationships.Tracks.Data, obj2.Data...)
			next = obj2.Next
			if len(next) == 0 {
				break
			}
		}
	}
	return obj, nil
}

func GetAlbumRespByHref(href string, language string, token string) (*AlbumResp, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return nil, err
		}
	}
	href = strings.Split(href, "?")[0]
	req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com%s/albums", href), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	query := url.Values{}
	query.Set("omit[resource]", "autos")
	query.Set("include", "tracks,artists,record-labels")
	query.Set("include[songs]", "artists")
	//query.Set("fields[artists]", "name,artwork")
	//query.Set("fields[albums:albums]", "artistName,artwork,name,releaseDate,url")
	//query.Set("fields[record-labels]", "name")
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
	obj := new(AlbumResp)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return nil, err
	}
	if len(obj.Data[0].Relationships.Tracks.Next) > 0 {
		next := obj.Data[0].Relationships.Tracks.Next
		for {
			req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com%s", next), nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
			req.Header.Set("Origin", "https://music.apple.com")
			query := req.URL.Query()
			query.Set("omit[resource]", "autos")
			query.Set("include", "artists")
			query.Set("extend", "editorialVideo,extendedAssetUrls")
			req.URL.RawQuery = query.Encode()
			do, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer do.Body.Close()
			if do.StatusCode != http.StatusOK {
				return nil, errors.New(do.Status)
			}
			obj2 := new(TrackResp)
			err = json.NewDecoder(do.Body).Decode(&obj2)
			if err != nil {
				return nil, err
			}
			obj.Data[0].Relationships.Tracks.Data = append(obj.Data[0].Relationships.Tracks.Data, obj2.Data...)
			next = obj2.Next
			if len(next) == 0 {
				break
			}
		}
	}
	return obj, nil
}

type AlbumResp struct {
	Href string          `json:"href"`
	Next string          `json:"next"`
	Data []AlbumRespData `json:"data"`
}

type AlbumRespData struct {
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
		ArtistName           string   `json:"artistName"`
		IsSingle             bool     `json:"isSingle"`
		URL                  string   `json:"url"`
		IsComplete           bool     `json:"isComplete"`
		GenreNames           []string `json:"genreNames"`
		TrackCount           int      `json:"trackCount"`
		IsMasteredForItunes  bool     `json:"isMasteredForItunes"`
		IsAppleDigitalMaster bool     `json:"isAppleDigitalMaster"`
		ContentRating        string   `json:"contentRating"`
		ReleaseDate          string   `json:"releaseDate"`
		Name                 string   `json:"name"`
		RecordLabel          string   `json:"recordLabel"`
		Upc                  string   `json:"upc"`
		AudioTraits          []string `json:"audioTraits"`
		Copyright            string   `json:"copyright"`
		PlayParams           struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"playParams"`
		IsCompilation  bool `json:"isCompilation"`
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
	} `json:"attributes"`
	Relationships struct {
		RecordLabels struct {
			Href string        `json:"href"`
			Data []interface{} `json:"data"`
		} `json:"record-labels"`
		Artists struct {
			Href string `json:"href"`
			Data []struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Href       string `json:"href"`
				Attributes struct {
					Name    string `json:"name"`
					Artwork struct {
						Url string `json:"url"`
					} `json:"artwork"`
				} `json:"attributes"`
			} `json:"data"`
		} `json:"artists"`
		Tracks TrackResp `json:"tracks"`
	} `json:"relationships"`
}
