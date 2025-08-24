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
			TtmlLocalizations       string `json:"ttmlLocalizations"`
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
		fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/songs/%s/%s?l=%s&extend=ttmlLocalizations", storefront, songId, lrcType, language), nil)
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
		if len(obj.Data[0].Attributes.Ttml) > 0 {
			return obj.Data[0].Attributes.Ttml, nil
		}
		return obj.Data[0].Attributes.TtmlLocalizations, nil
	} else {
		return "", errors.New("failed to get lyrics")
	}
}

// Use for detect if lyrics have CJK, will be replaced by transliteration if exist.
func containsCJK(s string) bool {
	for _, r := range s {
		if (r >= 0x1100 && r <= 0x11FF)    || // Hangul Jamo
			(r >= 0x2E80 && r <= 0x2EFF)   || // CJK Radicals Supplement
			(r >= 0x2F00 && r <= 0x2FDF)   || // Kangxi Radicals
			(r >= 0x2FF0 && r <= 0x2FFF)   || // Ideographic Description Characters
			(r >= 0x3000 && r <= 0x303F)   || // CJK Symbols and Punctuation
			(r >= 0x3040 && r <= 0x309F)   || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF)   || // Katakana
			(r >= 0x3130 && r <= 0x318F)   || // Hangul Compatibility Jamo
			(r >= 0x31C0 && r <= 0x31EF)   || // CJK Strokes
			(r >= 0x31F0 && r <= 0x31FF)   || // Katakana Phonetic Extensions
			(r >= 0x3200 && r <= 0x32FF)   || // Enclosed CJK Letters and Months
			(r >= 0x3300 && r <= 0x33FF)   || // CJK Compatibility
			(r >= 0x3400 && r <= 0x4DBF)   || // CJK Unified Ideographs Extension A
			(r >= 0x4E00 && r <= 0x9FFF)   || // CJK Unified Ideographs
			(r >= 0xA960 && r <= 0xA97F)   || // Hangul Jamo Extended-A
			(r >= 0xAC00 && r <= 0xD7AF)   || // Hangul Syllables
			(r >= 0xD7B0 && r <= 0xD7FF)   || // Hangul Jamo Extended-B
			(r >= 0xF900 && r <= 0xFAFF)   || // CJK Compatibility Ideographs
			(r >= 0xFE30 && r <= 0xFE4F)   || // CJK Compatibility Forms
			(r >= 0xFF65 && r <= 0xFF9F)   || // Halfwidth Katakana
			(r >= 0xFFA0 && r <= 0xFFDC)   || // Halfwidth Jamo
			(r >= 0x1AFF0 && r <= 0x1AFFF) || // Kana Extended-B
			(r >= 0x1B000 && r <= 0x1B0FF) || // Kana Supplement
			(r >= 0x1B100 && r <= 0x1B12F) || // Kana Extended-A
			(r >= 0x1B130 && r <= 0x1B16F) || // Small Kana Extension
			(r >= 0x1F200 && r <= 0x1F2FF) || // Enclosed Ideographic Supplement
			(r >= 0x20000 && r <= 0x2A6DF) || // CJK Unified Ideographs Extension B
			(r >= 0x2A700 && r <= 0x2B73F) || // CJK Unified Ideographs Extension C
			(r >= 0x2B740 && r <= 0x2B81F) || // CJK Unified Ideographs Extension D
			(r >= 0x2B820 && r <= 0x2CEAF) || // CJK Unified Ideographs Extension E
			(r >= 0x2CEB0 && r <= 0x2EBEF) || // CJK Unified Ideographs Extension F
			(r >= 0x2EBF0 && r <= 0x2EE5F) || // CJK Unified Ideographs Extension I
			(r >= 0x2F800 && r <= 0x2FA1F) || // CJK Compatibility Ideographs Supplement
			(r >= 0x30000 && r <= 0x3134F) || // CJK Unified Ideographs Extension G
			(r >= 0x31350 && r <= 0x323AF) {  // CJK Unified Ideographs Extension H
			return true
		}
	}
	return false
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
			beginAttr := lyric.SelectAttr("begin")
			if beginAttr == nil {
				return "", errors.New("no synchronised lyrics")
			}
			beginValue := beginAttr.Value
			if strings.Contains(beginValue, ":") {
				_, err = fmt.Sscanf(beginValue, "%d:%d:%d.%d", &h, &m, &s, &ms)
				if err != nil {
					_, err = fmt.Sscanf(beginValue, "%d:%d.%d", &m, &s, &ms)
					if err != nil {
						_, err = fmt.Sscanf(beginValue, "%d:%d", &m, &s)
					}
					h = 0
				}
			} else {
				_, err = fmt.Sscanf(beginValue, "%d.%d", &s, &ms)
				h, m = 0, 0
			}
			if err != nil {
				return "", err
			}
			m += h * 60
			ms = ms / 10
			var text, transText, translitText string
			//GET trans and translit
			if len(parsedTTML.FindElement("tt").FindElements("head")) > 0 {
				if len(parsedTTML.FindElement("tt").FindElement("head").FindElements("metadata")) > 0 {
					Metadata := parsedTTML.FindElement("tt").FindElement("head").FindElement("metadata")
					if len(Metadata.FindElements("iTunesMetadata")) > 0 {
						iTunesMetadata := Metadata.FindElement("iTunesMetadata")
						if len(iTunesMetadata.FindElements("transliterations")) > 0 {
							if len(iTunesMetadata.FindElement("transliterations").FindElements("transliteration")) > 0 {
								xpath := fmt.Sprintf("text[@for='%s']", lyric.SelectAttr("itunes:key").Value)
								translit := iTunesMetadata.FindElement("transliterations").FindElement("transliteration").FindElement(xpath)
								if translit != nil {
									if translit.SelectAttr("text") != nil {
										translitText = translit.SelectAttr("text").Value
									} else {
										var translitTmp []string
										for _, span := range translit.Child {
											if c, ok := span.(*etree.CharData); ok {
												translitTmp = append(translitTmp, c.Data)
											} else if e, ok := span.(*etree.Element); ok {
												translitTmp = append(translitTmp, e.Text())
											}
										}
										translitText = strings.Join(translitTmp, "")
									}
								}
							}
						}
						if len(iTunesMetadata.FindElements("translations")) > 0 {
							if len(iTunesMetadata.FindElement("translations").FindElements("translation")) > 0 {
								xpath := fmt.Sprintf("//text[@for='%s']", lyric.SelectAttr("itunes:key").Value)
								trans := iTunesMetadata.FindElement("translations").FindElement("translation").FindElement(xpath)
								if trans != nil {
									if trans.SelectAttr("text") != nil {
										transText = trans.SelectAttr("text").Value
									} else {
										var transTmp []string
										for _, span := range trans.Child {
											if c, ok := span.(*etree.CharData); ok {
												transTmp = append(transTmp, c.Data)
											} else if e, ok := span.(*etree.Element); ok {
												transTmp = append(transTmp, e.Text())
											}
										}
										transText = strings.Join(transTmp, "")
									}
								}
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
			if len(transText) > 0 {
				lrcLines = append(lrcLines, fmt.Sprintf("[%02d:%02d.%02d]%s", m, s, ms, transText))
			}
			if len(translitText) > 0 && containsCJK(text) {
				lrcLines = append(lrcLines, fmt.Sprintf("[%02d:%02d.%02d]%s", m, s, ms, translitText))
			} else {
				lrcLines = append(lrcLines, fmt.Sprintf("[%02d:%02d.%02d]%s", m, s, ms, text))
			}
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
	parseTime := func(timeValue string, newLine int) (string, error) {
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
		if newLine == 0 {
			return fmt.Sprintf("[%02d:%02d.%02d]<%02d:%02d.%02d>", m, s, ms, m, s, ms), nil
		} else if newLine == -1 {
			return fmt.Sprintf("[%02d:%02d.%02d]", m, s, ms), nil
		} else {
			return fmt.Sprintf("<%02d:%02d.%02d>", m, s, ms), nil
		}
	}
	divs := parsedTTML.FindElement("tt").FindElement("body").FindElements("div")
	for _, div := range divs {
		for _, item := range div.ChildElements() {     //LINES
			var lrcSyllables []string
			var i int = 0
			var endTime, translitLine, transLine string
			for _, lyrics := range item.Child {    //WORDS
				if _, ok := lyrics.(*etree.CharData); ok {   //是否为span之间的空格
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
				beginTime, err := parseTime(lyric.SelectAttr("begin").Value, i)
				if err != nil {
						return "", err
				}

				endTime, err = parseTime(lyric.SelectAttr("end").Value, 1)
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
				if i == 0 {
					transBeginTime, _ := parseTime(lyric.SelectAttr("begin").Value, -1)
					sharedTimestamp := ""
					if len(parsedTTML.FindElement("tt").FindElements("head")) > 0 {
						if len(parsedTTML.FindElement("tt").FindElement("head").FindElements("metadata")) > 0 {
							Metadata := parsedTTML.FindElement("tt").FindElement("head").FindElement("metadata")
							if len(Metadata.FindElements("iTunesMetadata")) > 0 {
								iTunesMetadata := Metadata.FindElement("iTunesMetadata")
								if len(iTunesMetadata.FindElements("transliterations")) > 0 {
									if len(iTunesMetadata.FindElement("transliterations").FindElements("transliteration")) > 0 {
										xpath := fmt.Sprintf("text[@for='%s']", item.SelectAttr("itunes:key").Value)
										trans := iTunesMetadata.FindElement("transliterations").FindElement("transliteration").FindElement(xpath)
										// Get text content
										var transTxtParts []string
										var transStartTime string
										for i, span := range trans.ChildElements() {
											if span.Tag == "span" {
												spanBegin := span.SelectAttrValue("begin", "")
												spanText := span.Text()
												if spanBegin == "" {
													continue
												}
												// Get timestamp
												timestamp, err := parseTime(spanBegin, 2)
												if err != nil {
													return "", err
												}
												if i == 0 {
													// For [mm:ss.xx] prefix
													transStartTime, _ = parseTime(spanBegin, -1)
													sharedTimestamp = transStartTime
												}
												transTxtParts = append(transTxtParts, fmt.Sprintf("%s%s", timestamp, spanText))
											}
										}
										translitLine = fmt.Sprintf("%s%s", transStartTime, strings.Join(transTxtParts, " "))
									}
								}
								if len(iTunesMetadata.FindElements("translations")) > 0 {
									if len(iTunesMetadata.FindElement("translations").FindElements("translation")) > 0 {
										xpath := fmt.Sprintf("//text[@for='%s']", item.SelectAttr("itunes:key").Value)
										trans := iTunesMetadata.FindElement("translations").FindElement("translation").FindElement(xpath)
										var transTxt string
										if trans.SelectAttr("text") == nil {
											var textTmp []string
											for _, span := range trans.Child {
												if _, ok := span.(*etree.CharData); ok {
													textTmp = append(textTmp, span.(*etree.CharData).Data)
												} /*else {
													textTmp = append(textTmp, span.(*etree.Element).Text())
												}*/
											}
											transTxt = strings.Join(textTmp, "")
										} else {
											transTxt = trans.SelectAttr("text").Value
										}
										//fmt.Println(transTxt)
										if sharedTimestamp != "" {
											transLine = sharedTimestamp + transTxt
										} else {
											transLine = transBeginTime + transTxt
										}
									}
								}
							}
						}
					}
				}
				i += 1
			}
			//endTime, err := parseTime(item.SelectAttr("end").Value)
			//if err != nil {
			//	return "", err
			//}
			if len(transLine) > 0 {
				lrcLines = append(lrcLines, transLine)
			}
			if len(translitLine) > 0 && containsCJK(strings.Join(lrcSyllables, "")) {
				lrcLines = append(lrcLines, translitLine)
			} else {
				lrcLines = append(lrcLines, strings.Join(lrcSyllables, "")+endTime)
			}
		}
	}
	return strings.Join(lrcLines, "\n"), nil
}
