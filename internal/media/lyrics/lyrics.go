package lyrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"amdl/internal/download"
)

func Get(songId, lrcType, language, lrcFormat, liteServer, lrcExtra string) (string, error) {
	ttml, err := getSongLyrics(songId, liteServer, lrcType, language)
	if err != nil {
		return "", err
	}

	if lrcFormat == "ttml" {
		return ttml, nil
	}

	lrc, err := TtmlToLrc(ttml, lrcExtra)
	if err != nil {
		return "", err
	}

	return lrc, nil
}

func getSongLyrics(songId string, liteServer string, lrcType string, language string) (string, error) {
	if liteServer == "" {
		return "", errors.New("lite-server is not configured")
	}
	isSyllable := "1"
	if lrcType == "lyrics" {
		isSyllable = "0"
	}
	endpoint := strings.TrimRight(liteServer, "/") + "/lyrics?adamId=" + url.QueryEscape(songId) + "&language=" + url.QueryEscape(language) + "&syllable=" + isSyllable
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
			Lyrics string `json:"lyrics"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}
	if envelope.Code != 0 {
		return "", fmt.Errorf("lite-server /lyrics returned code=%d msg=%s", envelope.Code, envelope.Msg)
	}
	return envelope.Data.Lyrics, nil
}

// TtmlToLrc converts TTML lyrics into plain LRC. Word timing is delegated to
// conventSyllableTTMLToLRC so both entry points continue to share one format.
func TtmlToLrc(ttml string, lyricsExtra string) (string, error) {
	document, err := parseTTML(ttml)
	if err != nil {
		return "", err
	}

	root := document.Root()
	metadata := newItunesMetadataIndex(root)

	switch root.SelectAttrValue("itunes:timing", "") {
	case "Word":
		return conventSyllableTTMLToLRC(ttml, lyricsExtra)
	case "None":
		return convertUnTimedLines(root), nil
	}

	body := root.NotNil().SelectElement("body")
	if body == nil {
		return "", errors.New("TTML does not contain a body")
	}

	var lrcLines []string
	for line := range body.ChildElementsSeq() {
		for lyric := range line.ChildElementsSeq() {
			begin, err := parseTTMLTime(lyric.SelectAttrValue("begin", ""))
			if err != nil {
				return "", err
			}

			key := lyric.SelectAttrValue("itunes:key", "")
			primaryText := extractElementText(lyric)
			translation, transliteration := metadata.textsFor(key)

			if metadata.translationType == "replacement" && translation != "" {
				primaryText = translation
			}
			lrcLines = append(lrcLines, begin.lineTag()+primaryText)

			if transliteration != "" && lyricsExtra == "pronunciation" {
				lrcLines = append(lrcLines, begin.lineTag()+transliteration)
			} else if metadata.translationType == "subtitle" && translation != "" && lyricsExtra == "translation" {
				lrcLines = append(lrcLines, begin.lineTag()+translation)
			}
		}
	}

	return strings.Join(lrcLines, "\n"), nil
}

// conventSyllableTTMLToLRC keeps the historical function name. It converts
// word-level Apple Music TTML while adding pronunciation or translation lines.
func conventSyllableTTMLToLRC(ttml string, lyricsExtra string) (string, error) {
	document, err := parseTTML(ttml)
	if err != nil {
		return "", err
	}

	root := document.Root()
	body := root.NotNil().SelectElement("body")
	if body == nil {
		return "", errors.New("TTML does not contain a body")
	}

	metadata := newItunesMetadataIndex(root)
	var lrcLines []string
	for wordContainer := range body.SelectElementsSeq("div") {
		for line := range wordContainer.ChildElementsSeq() {
			converted, err := convertSyllableLine(line, metadata, lyricsExtra)
			if err != nil {
				return "", err
			}
			if converted.line != "" {
				lrcLines = append(lrcLines, converted.line)
			}
			if converted.extraLine != "" {
				lrcLines = append(lrcLines, converted.extraLine)
			}
		}
	}

	return strings.Join(lrcLines, "\n"), nil
}

func convertUnTimedLines(root *etree.Element) string {
	var lines []string
	for paragraph := range root.FindElementsSeq("//p") {
		if line := strings.TrimSpace(paragraph.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// itunesMetadataIndex is built once per document. It replaces repeated whole-
// tree XPath lookups inside lyric loops with direct key-based access.
type itunesMetadataIndex struct {
	translations     map[string]*etree.Element
	transliterations map[string]*etree.Element
	translationType  string
}

func newItunesMetadataIndex(root *etree.Element) *itunesMetadataIndex {
	index := &itunesMetadataIndex{
		translations:     make(map[string]*etree.Element),
		transliterations: make(map[string]*etree.Element),
	}

	iTunesMetadata := root.
		NotNil().
		SelectElement("head").
		NotNil().
		SelectElement("metadata").
		NotNil().
		SelectElement("iTunesMetadata")
	if iTunesMetadata == nil {
		return index
	}

	for translations := range iTunesMetadata.SelectElementsSeq("translations") {
		if index.translationType == "" {
			index.translationType = translations.SelectAttrValue("type", "")
		}
		for translation := range translations.SelectElementsSeq("translation") {
			index.translations[translation.SelectAttrValue("for", "")] = translation
		}
	}

	for transliterations := range iTunesMetadata.SelectElementsSeq("transliterations") {
		for transliteration := range transliterations.SelectElementsSeq("transliteration") {
			index.transliterations[transliteration.SelectAttrValue("for", "")] = transliteration
		}
	}

	return index
}

func (i *itunesMetadataIndex) textsFor(key string) (translation, transliteration string) {
	if element := i.translations[key]; element != nil {
		translation = extractElementText(element)
	}
	if element := i.transliterations[key]; element != nil {
		transliteration = extractElementText(element)
	}
	return translation, transliteration
}

// syllableLine is the intermediate representation for one word-level LRC line.
type syllableLine struct {
	line      string
	extraLine string
}

func convertSyllableLine(line *etree.Element, metadata *itunesMetadataIndex, lyricsExtra string) (syllableLine, error) {
	var syllables []string
	var endTime lrcTime

	var wordCount int
	for _, node := range line.Child {
		_, isText := node.(*etree.CharData)
		if isText {
			if wordCount > 0 {
				syllables = append(syllables, " ")
			}
			continue
		}

		word, isElement := node.(*etree.Element)
		if !isElement {
			continue
		}
		beginValue := word.SelectAttrValue("begin", "")
		endValue := word.SelectAttrValue("end", "")
		if beginValue == "" && endValue == "" {
			continue
		}

		begin, err := parseTTMLTime(beginValue)
		if err != nil {
			return syllableLine{}, err
		}
		endTime, err = parseTTMLTime(endValue)
		if err != nil {
			return syllableLine{}, err
		}

		syllableTag := begin.syllableTag()
		if wordCount == 0 {
			syllableTag = begin.lineTag() + syllableTag
		}
		syllables = append(syllables, syllableTag+extractElementText(word))
		wordCount++
	}

	converted := syllableLine{line: strings.Join(syllables, "") + endTime.lineTag()}
	if wordCount == 0 {
		return converted, nil
	}

	key := line.SelectAttrValue("itunes:key", "")

	if metadata != nil && metadata.translationType == "replacement" {
		translation := metadata.translations[key]
		if translation != nil {
			replacementLine, err := buildTimedMetadataLine(translation)
			if err != nil {
				return syllableLine{}, err
			}
			if replacementLine != "" {
				converted.line = replacementLine
			}
		}
	}

	switch {
	case metadata != nil && lyricsExtra == "pronunciation":
		if transliteration := metadata.transliterations[key]; transliteration != nil {
			pronunciationLine, err := buildTimedMetadataLine(transliteration)
			if err != nil {
				return syllableLine{}, err
			}
			converted.extraLine = pronunciationLine
		}
	case metadata != nil && metadata.translationType == "subtitle" && lyricsExtra == "translation":
		lineBegin, err := parseTTMLTime(firstWordBegin(line))
		if err != nil {
			return syllableLine{}, err
		}
		translation := metadata.translations[key]
		if translation != nil {
			converted.extraLine = lineBegin.lineTag() + inlineTextWithoutElements(translation)
		}
	}

	return converted, nil
}

func buildTimedMetadataLine(metadataText *etree.Element) (string, error) {
	var timedParts []string
	var startTime lrcTime

	var spanCount int
	for span := range metadataText.ChildElementsSeq() {
		if span.Tag != "span" {
			continue
		}

		spanBegin, err := parseTTMLTime(span.SelectAttrValue("begin", ""))
		if err != nil {
			return "", err
		}
		if spanCount == 0 {
			startTime = spanBegin
		}
		timedParts = append(timedParts, spanBegin.syllableTag()+span.Text())
		spanCount++
	}

	return startTime.lineTag() + strings.Join(timedParts, " "), nil
}

func firstWordBegin(line *etree.Element) string {
	for word := range line.ChildElementsSeq() {
		if word.SelectAttrValue("begin", "") != "" {
			return word.SelectAttrValue("begin", "")
		}
	}
	return ""
}

func parseTTML(ttml string) (*etree.Document, error) {
	document := etree.NewDocument()
	if err := document.ReadFromString(ttml); err != nil {
		return nil, fmt.Errorf("read TTML: %w", err)
	}
	return document, nil
}

// lrcTime stores the centisecond form used by the standard LRC tag formats.
type lrcTime struct {
	minutes      int
	seconds      int
	centiseconds int
}

func parseTTMLTime(timeValue string) (lrcTime, error) {
	timeValue = strings.TrimSuffix(strings.TrimSpace(timeValue), "s")
	fields := strings.Split(timeValue, ":")
	if len(fields) > 3 {
		return lrcTime{}, fmt.Errorf("parse TTML time %q: invalid field count", timeValue)
	}

	secondsField, fraction, hasFraction := strings.Cut(fields[len(fields)-1], ".")
	seconds, err := strconv.Atoi(secondsField)
	if err != nil {
		return lrcTime{}, fmt.Errorf("parse TTML time %q: %w", timeValue, err)
	}

	centiseconds := seconds * 100
	if hasFraction {
		if len(fraction) > 2 {
			fraction = fraction[:2]
		}
		fraction += strings.Repeat("0", 2-len(fraction))
		fractionValue, err := strconv.Atoi(fraction)
		if err != nil {
			return lrcTime{}, fmt.Errorf("parse TTML time %q: %w", timeValue, err)
		}
		centiseconds += fractionValue
	}

	// Convert each non-final field to seconds before adding the final field.
	// This also tolerates fields that overflow their conventional unit.
	for _, field := range fields[:len(fields)-1] {
		value, err := strconv.Atoi(field)
		if err != nil {
			return lrcTime{}, fmt.Errorf("parse TTML time %q: %w", timeValue, err)
		}
		centiseconds += value * 6000
	}

	// A component may legitimately overflow its conventional field (for
	// example "101.046s"). Reduce everything through centiseconds so the LRC
	// tag always keeps seconds below 60.
	return lrcTime{
		minutes:      centiseconds / 6000,
		seconds:      centiseconds % 6000 / 100,
		centiseconds: centiseconds % 100,
	}, nil
}

func (t lrcTime) lineTag() string {
	return fmt.Sprintf("[%02d:%02d.%02d]", t.minutes, t.seconds, t.centiseconds)
}

func (t lrcTime) syllableTag() string {
	return fmt.Sprintf("<%02d:%02d.%02d>", t.minutes, t.seconds, t.centiseconds)
}

// extractElementText reads an explicit text attribute or joins inline text and
// child text. It intentionally does not descend below immediate children.
func extractElementText(element *etree.Element) string {
	if text := element.SelectAttrValue("text", ""); text != "" {
		return text
	}

	var parts []string
	for _, node := range element.Child {
		switch child := node.(type) {
		case *etree.CharData:
			parts = append(parts, child.Data)
		case *etree.Element:
			parts = append(parts, child.Text())
		}
	}
	return strings.Join(parts, "")
}

func inlineTextWithoutElements(element *etree.Element) string {
	if text := element.SelectAttrValue("text", ""); text != "" {
		return text
	}

	var parts []string
	for _, node := range element.Child {
		if text, ok := node.(*etree.CharData); ok {
			parts = append(parts, text.Data)
		}
	}
	return strings.Join(parts, "")
}
