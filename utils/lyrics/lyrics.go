package lyrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/beevik/etree"
)

type SongLyrics struct {
	Data []struct {
		Id         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Ttml       string `json:"ttml"`
			PlayParams struct {
				Id          string `json:"id"`
				Kind        string `json:"kind"`
				CatalogId   string `json:"catalogId"`
				DisplayType int    `json:"displayType"`
			} `json:"playParams"`
		} `json:"attributes"`
	} `json:"data"`
}

func Get(storefront, songId, lrcType, language, lrcFormat, token, mediaUserToken string) (string, error) {
	if len(mediaUserToken) < 50 {
		return "", errors.New("MediaUserToken not set")
	}

	ttml, err := getSongLyrics(songId, storefront, token, mediaUserToken, lrcType, language)
	if err != nil {
		return "", err
	}

	if lrcFormat == "ttml" {
		return ttml, nil
	}

	lrc, err := TtmlToLrc(ttml)
	if err != nil {
		return "", err
	}

	return lrc, nil
}
func getSongLyrics(songId string, storefront string, token string, userToken string, lrcType string, language string) (string, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/songs/%s/%s?l=%s", storefront, songId, lrcType, language), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("Referer", "https://music.apple.com/")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	cookie := http.Cookie{Name: "media-user-token", Value: userToken}
	req.AddCookie(&cookie)
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer do.Body.Close()
	obj := new(SongLyrics)
	_ = json.NewDecoder(do.Body).Decode(&obj)
	if obj.Data != nil {
		return obj.Data[0].Attributes.Ttml, nil
	} else {
		return "", errors.New("failed to get lyrics")
	}
}

func TtmlToLrc(ttml string) (string, error) {
	parsedTTML := etree.NewDocument()
	err := parsedTTML.ReadFromString(ttml)
	if err != nil {
		return "", err
	}

	var lrcLines []string
	timingAttr := parsedTTML.FindElement("tt").SelectAttr("itunes:timing")
	if timingAttr != nil {
		if timingAttr.Value == "Word" {
			lrc, err := conventSyllableTTMLToLRC(ttml)
			return lrc, err
		}
		if timingAttr.Value == "None" {
			for _, p := range parsedTTML.FindElements("//p") {
				line := p.Text()
				line = strings.TrimSpace(line)
				if line != "" {
					lrcLines = append(lrcLines, line)
				}
			}
			return strings.Join(lrcLines, "\n"), nil
		}
	}

	for _, item := range parsedTTML.FindElement("tt").FindElement("body").ChildElements() {
		for _, lyric := range item.ChildElements() {
			var h, m, s, ms int
			if lyric.SelectAttr("begin") == nil {
				return "", errors.New("no synchronised lyrics")
			}
			if strings.Contains(lyric.SelectAttr("begin").Value, ":") {
				_, err = fmt.Sscanf(lyric.SelectAttr("begin").Value, "%d:%d:%d.%d", &h, &m, &s, &ms)
				if err != nil {
					_, err = fmt.Sscanf(lyric.SelectAttr("begin").Value, "%d:%d.%d", &m, &s, &ms)
					if err != nil {
						_, err = fmt.Sscanf(lyric.SelectAttr("begin").Value, "%d:%d", &m, &s)
					}
					h = 0
				}
			} else {
				_, err = fmt.Sscanf(lyric.SelectAttr("begin").Value, "%d.%d", &s, &ms)
				h, m = 0, 0
			}
			if err != nil {
				return "", err
			}
			var text string
			//GET trans
			if len(parsedTTML.FindElement("tt").FindElements("head")) > 0 {
				if len(parsedTTML.FindElement("tt").FindElement("head").FindElements("metadata")) > 0 {
					Metadata := parsedTTML.FindElement("tt").FindElement("head").FindElement("metadata")
					if len(Metadata.FindElements("iTunesMetadata")) > 0 {
						iTunesMetadata := Metadata.FindElement("iTunesMetadata")
						if len(iTunesMetadata.FindElements("translations")) > 0 {
							if len(iTunesMetadata.FindElement("translations").FindElements("translation")) > 0 {
								xpath := fmt.Sprintf("//text[@for='%s']", lyric.SelectAttr("itunes:key").Value)
								trans := iTunesMetadata.FindElement("translations").FindElement("translation").FindElement(xpath)
								lyric = trans
							}
						}
					}
				}
			}
			if lyric.SelectAttr("text") == nil {
				var textTmp []string
				for _, span := range lyric.Child {
					if _, ok := span.(*etree.CharData); ok {
						textTmp = append(textTmp, span.(*etree.CharData).Data)
					} else {
						textTmp = append(textTmp, span.(*etree.Element).Text())
					}
				}
				text = strings.Join(textTmp, "")
			} else {
				text = lyric.SelectAttr("text").Value
			}
			m += h * 60
			ms = ms / 10
			lrcLines = append(lrcLines, fmt.Sprintf("[%02d:%02d.%02d]%s", m, s, ms, text))
		}
	}
	return strings.Join(lrcLines, "\n"), nil
}

func conventSyllableTTMLToLRC(ttml string) (string, error) {
	parsedTTML := etree.NewDocument()
	err := parsedTTML.ReadFromString(ttml)
	if err != nil {
		return "", err
	}
	var lrcLines []string
	parseTime := func(timeValue string, newLine bool) (string, error) {
		var h, m, s, ms int
		if strings.Contains(timeValue, ":") {
			_, err = fmt.Sscanf(timeValue, "%d:%d:%d.%d", &h, &m, &s, &ms)
			if err != nil {
				_, err = fmt.Sscanf(timeValue, "%d:%d.%d", &m, &s, &ms)
				h = 0
			}
		} else {
			_, err = fmt.Sscanf(timeValue, "%d.%d", &s, &ms)
			h, m = 0, 0
		}
		if err != nil {
			return "", err
		}
		m += h * 60
		ms = ms / 10
		if newLine {
			return fmt.Sprintf("[%02d:%02d.%02d]<%02d:%02d.%02d>", m, s, ms, m, s, ms), nil
		} else {
			return fmt.Sprintf("<%02d:%02d.%02d>", m, s, ms), nil
		}
	}
	divs := parsedTTML.FindElement("tt").FindElement("body").FindElements("div")
	//get trans
	if len(parsedTTML.FindElement("tt").FindElements("head")) > 0 {
		if len(parsedTTML.FindElement("tt").FindElement("head").FindElements("metadata")) > 0 {
			Metadata := parsedTTML.FindElement("tt").FindElement("head").FindElement("metadata")
			if len(Metadata.FindElements("iTunesMetadata")) > 0 {
				iTunesMetadata := Metadata.FindElement("iTunesMetadata")
				if len(iTunesMetadata.FindElements("translations")) > 0 {
					if len(iTunesMetadata.FindElement("translations").FindElements("translation")) > 0 {
						divs = iTunesMetadata.FindElement("translations").FindElements("translation")
					}
				}
			}
		}
	}
	for _, div := range divs {
		for _, item := range div.ChildElements() {
			var lrcSyllables []string
			var i int = 0
			var endTime string
			for _, lyrics := range item.Child {
				if _, ok := lyrics.(*etree.CharData); ok {
					if i > 0 {
						lrcSyllables = append(lrcSyllables, " ")
						continue
					}
					continue
				}
				lyric := lyrics.(*etree.Element)
				if lyric.SelectAttr("begin") == nil {
					continue
				}
				beginTime, err := parseTime(lyric.SelectAttr("begin").Value, i == 0)
				if err != nil {
					return "", err
				}
				endTime, err = parseTime(lyric.SelectAttr("end").Value, false)
				if err != nil {
					return "", err
				}
				var text string
				if lyric.SelectAttr("text") == nil {
					var textTmp []string
					for _, span := range lyric.Child {
						if _, ok := span.(*etree.CharData); ok {
							textTmp = append(textTmp, span.(*etree.CharData).Data)
						} else {
							textTmp = append(textTmp, span.(*etree.Element).Text())
						}
					}
					text = strings.Join(textTmp, "")
				} else {
					text = lyric.SelectAttr("text").Value
				}
				lrcSyllables = append(lrcSyllables, fmt.Sprintf("%s%s", beginTime, text))
				i += 1
			}
			//endTime, err := parseTime(item.SelectAttr("end").Value)
			//if err != nil {
			//	return "", err
			//}
			lrcLines = append(lrcLines, strings.Join(lrcSyllables, "")+endTime)
		}
	}
	return strings.Join(lrcLines, "\n"), nil
}
