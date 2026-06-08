package runv3

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/proto"

	cdm "main/utils/runv3/cdm"
	key "main/utils/runv3/key"
	"main/internal/logging"
	"os"

	"bytes"
	"errors"
	"io"

	"github.com/itouakirai/mp4ff/mp4"

	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/grafov/m3u8"
	"github.com/schollz/progressbar/v3"
)

const (
	minSongBytes   = 50 * 1024
	maxDownloadTry = 3
	maxConcurrency = 10
)

var cdnHeaders = map[string]string{
	"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Referer":    "https://music.apple.com/",
	"Origin":     "https://music.apple.com",
}

type PlaybackLicense struct {
	ErrorCode  int    `json:"errorCode"`
	License    string `json:"license"`
	RenewAfter int    `json:"renew-after"`
	Status     int    `json:"status"`
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 120 * time.Second}
}

func getPSSH(contentId string, kidBase64 string) (string, error) {
	kidBytes, err := base64.StdEncoding.DecodeString(kidBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 KID: %v", err)
	}
	contentIdEncoded := base64.StdEncoding.EncodeToString([]byte(contentId))
	algo := cdm.WidevineCencHeader_AESCTR
	widevineCencHeader := &cdm.WidevineCencHeader{
		KeyId:     [][]byte{kidBytes},
		Algorithm: &algo,
		Provider:  new(string),
		ContentId: []byte(contentIdEncoded),
		Policy:    new(string),
	}
	widevineCenc, err := proto.Marshal(widevineCencHeader)
	if err != nil {
		return "", fmt.Errorf("failed to marshal WidevineCencHeader: %v", err)
	}
	widevineCenc = append([]byte("0123456789abcdef0123456789abcdef"), widevineCenc...)
	pssh := base64.StdEncoding.EncodeToString(widevineCenc)
	return pssh, nil
}

func BeforeRequest(cl *resty.Client, ctx context.Context, url string, body []byte) (*resty.Response, error) {
	jsondata := map[string]interface{}{
		"challenge":      base64.StdEncoding.EncodeToString(body),
		"key-system":     "com.widevine.alpha",
		"uri":            ctx.Value("uriPrefix").(string) + "," + ctx.Value("pssh").(string),
		"adamId":         ctx.Value("adamId").(string),
		"isLibrary":      false,
		"user-initiated": true,
	}

	resp, err := cl.R().
		SetContext(ctx).
		SetBody(jsondata).
		Post(url)

	if err != nil {
		fmt.Println(err)
	}

	return resp, err
}

func AfterRequest(response *resty.Response) ([]byte, error) {
	var responseData PlaybackLicense

	err := json.Unmarshal(response.Body(), &responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %v", err)
	}

	if responseData.ErrorCode != 0 || responseData.Status != 0 {
		return nil, fmt.Errorf("error in license response, code: %d, status: %d", responseData.ErrorCode, responseData.Status)
	}

	license, err := base64.StdEncoding.DecodeString(responseData.License)
	if err != nil {
		return nil, fmt.Errorf("failed to decode license: %v", err)
	}

	return license, nil
}

func GetWebplayback(adamId string, authtoken string, mutoken string, mvmode bool) (string, string, string, error) {
	url := "https://play.music.apple.com/WebObjects/MZPlay.woa/wa/webPlayback"
	postData := map[string]string{
		"salableAdamId": adamId,
	}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		return "", "", "", wrapStage(StagePlayback, err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		return "", "", "", wrapStage(StagePlayback, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("User-Agent", cdnHeaders["User-Agent"])
	req.Header.Set("Referer", "https://music.apple.com/")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authtoken))
	req.Header.Set("x-apple-music-user-token", mutoken)
	resp, err := httpClient().Do(req)
	if err != nil {
		return "", "", "", wrapStage(StagePlayback, err)
	}
	defer resp.Body.Close()
	obj := new(Songlist)
	err = json.NewDecoder(resp.Body).Decode(&obj)
	if err != nil {
		return "", "", "", wrapStage(StagePlayback, err)
	}
	if len(obj.List) > 0 {
		if mvmode {
			return obj.List[0].HlsPlaylistUrl, "", "", nil
		}
		for i := range obj.List[0].Assets {
			if obj.List[0].Assets[i].Flavor == "28:ctrp256" {
				kidBase64, parts, uriPrefix, err := parsePlaybackPlaylist(obj.List[0].Assets[i].URL)
				if err != nil {
					return "", "", "", err
				}
				return encodePartsForMV(parts), kidBase64, uriPrefix, nil
			}
		}
	}
	return "", "", "", errors.New("Unavailable")
}

type Songlist struct {
	List []struct {
		Hlsurl         string `json:"hls-key-cert-url"`
		HlsPlaylistUrl string `json:"hls-playlist-url"`
		Assets         []struct {
			Flavor string `json:"flavor"`
			URL    string `json:"URL"`
		} `json:"assets"`
	} `json:"songList"`
	Status int `json:"status"`
}

// fmp4Part is one contiguous byte range (or whole file) to concatenate for decryption.
type fmp4Part struct {
	URL    string
	Offset int64
	Limit  int64
}

func resolveMediaURL(base, uri string) string {
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return uri
	}
	return base + uri
}

func parsePlaybackPlaylist(playlistURL string) (string, []fmp4Part, string, error) {
	body, err := fetchPart(playlistURL)
	if err != nil {
		return "", nil, "", wrapStage(StageDownload, fmt.Errorf("playlist fetch: %w", err))
	}
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(string(body)), true)
	if err != nil {
		return "", nil, "", wrapStage(StageDownload, err)
	}
	if listType != m3u8.MEDIA {
		return "", nil, "", wrapStage(StageDownload, errors.New("m3u8 is not a media playlist"))
	}
	mediaPlaylist := from.(*m3u8.MediaPlaylist)
	if mediaPlaylist.Key == nil {
		return "", nil, "", wrapStage(StageDownload, errors.New("no encryption key in playlist"))
	}
	split := strings.Split(mediaPlaylist.Key.URI, ",")
	if len(split) < 2 {
		return "", nil, "", wrapStage(StageDownload, errors.New("invalid KEY URI in playlist"))
	}
	uriPrefix := split[0]
	kidbase64 := split[1]
	lastSlashIndex := strings.LastIndex(playlistURL, "/")
	if lastSlashIndex < 0 {
		return "", nil, "", wrapStage(StageDownload, errors.New("invalid playlist URL"))
	}
	base := playlistURL[:lastSlashIndex+1]

	var parts []fmp4Part
	usesByteRange := false

	if mediaPlaylist.Map != nil && mediaPlaylist.Map.URI != "" {
		part := fmp4Part{
			URL:    resolveMediaURL(base, mediaPlaylist.Map.URI),
			Offset: mediaPlaylist.Map.Offset,
			Limit:  mediaPlaylist.Map.Limit,
		}
		parts = append(parts, part)
		if part.Limit > 0 {
			usesByteRange = true
		}
	}

	segmentCount := 0
	for _, segment := range mediaPlaylist.Segments {
		if segment == nil || segment.URI == "" {
			continue
		}
		segmentCount++
		part := fmp4Part{
			URL:    resolveMediaURL(base, segment.URI),
			Offset: segment.Offset,
			Limit:  segment.Limit,
		}
		if part.Limit > 0 {
			usesByteRange = true
		}
		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return "", nil, "", wrapStage(StageDownload, errors.New("no init or media segments in playlist"))
	}

	logging.Info("runv3 playlist parsed: %d parts, byte-range=%v", len(parts), usesByteRange)
	return kidbase64, parts, uriPrefix, nil
}

func encodePartsForMV(parts []fmp4Part) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString(";")
		}
		b.WriteString(p.URL)
		if p.Limit > 0 {
			b.WriteString(fmt.Sprintf("#%d:%d", p.Offset, p.Limit))
		}
	}
	return b.String()
}

func decodeMVPart(raw string) fmp4Part {
	part := fmp4Part{URL: raw}
	if idx := strings.LastIndex(raw, "#"); idx > 0 && strings.Contains(raw[idx:], ":") {
		part.URL = raw[:idx]
		var off, lim int64
		if _, err := fmt.Sscanf(raw[idx+1:], "%d:%d", &off, &lim); err == nil && lim > 0 {
			part.Offset = off
			part.Limit = lim
		}
	}
	return part
}

func fetchPartRange(rawURL string, offset, limit int64) ([]byte, error) {
	if limit <= 0 {
		return fetchPart(rawURL)
	}
	var lastErr error
	client := httpClient()
	for attempt := 1; attempt <= maxDownloadTry; attempt++ {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range cdnHeaders {
			req.Header.Set(k, v)
		}
		end := offset + limit - 1
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d for range %s", resp.StatusCode, rawURL)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if int64(len(data)) != limit {
			lastErr = fmt.Errorf("short range read: got %d bytes, expected %d", len(data), limit)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func (p fmp4Part) fetch(client *http.Client) ([]byte, error) {
	if p.Limit > 0 {
		if client == nil {
			return fetchPartRange(p.URL, p.Offset, p.Limit)
		}
		return fetchPartRangeWithClient(client, p.URL, p.Offset, p.Limit)
	}
	if client == nil {
		return fetchPart(p.URL)
	}
	return fetchPartWithClient(client, p.URL)
}

func fetchPartRangeWithClient(client *http.Client, rawURL string, offset, limit int64) ([]byte, error) {
	if limit <= 0 {
		return fetchPartWithClient(client, rawURL)
	}
	var lastErr error
	for attempt := 1; attempt <= maxDownloadTry; attempt++ {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range cdnHeaders {
			req.Header.Set(k, v)
		}
		end := offset + limit - 1
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if int64(len(data)) != limit {
			lastErr = fmt.Errorf("short range read: got %d bytes, expected %d", len(data), limit)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func fetchPart(rawURL string) ([]byte, error) {
	var lastErr error
	client := httpClient()
	for attempt := 1; attempt <= maxDownloadTry; attempt++ {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range cdnHeaders {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.ContentLength > 0 && int64(len(data)) != resp.ContentLength {
			lastErr = fmt.Errorf("short read: got %d bytes, expected %d", len(data), resp.ContentLength)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func validateFmp4Buffer(buf []byte) error {
	if len(buf) < 8 {
		return fmt.Errorf("download too small (%d bytes)", len(buf))
	}
	if string(buf[4:8]) != "ftyp" {
		return fmt.Errorf("download invalid: missing ftyp box (got %q)", string(buf[4:8]))
	}
	if len(buf) < minSongBytes {
		return fmt.Errorf("download too small (%d bytes, minimum %d)", len(buf), minSongBytes)
	}
	return nil
}

func downloadFmp4Parts(parts []fmp4Part) (bytes.Buffer, error) {
	var buf bytes.Buffer
	if len(parts) == 0 {
		return buf, errors.New("no parts to download")
	}
	if len(parts) == 1 {
		data, err := parts[0].fetch(nil)
		if err != nil {
			return buf, err
		}
		buf.Write(data)
		logging.Info("runv3 downloaded single part: %d bytes", len(data))
		return buf, validateFmp4Buffer(buf.Bytes())
	}

	var downloadWg, writerWg sync.WaitGroup
	segmentsChan := make(chan Segment, len(parts))
	errChan := make(chan error, len(parts))
	limiter := make(chan struct{}, maxConcurrency)
	client := httpClient()

	bar := progressbar.NewOptions64(-1,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionShowBytes(true),
	)
	barWriter := io.MultiWriter(&buf, bar)

	writerWg.Add(1)
	go fileWriter(&writerWg, segmentsChan, barWriter, len(parts))

	for i, part := range parts {
		limiter <- struct{}{}
		downloadWg.Add(1)
		go downloadPart(part, i, &downloadWg, segmentsChan, client, limiter, errChan)
	}
	downloadWg.Wait()
	close(segmentsChan)
	writerWg.Wait()

drainErrors:
	for {
		select {
		case err := <-errChan:
			if err != nil {
				return buf, err
			}
		default:
			break drainErrors
		}
	}
	if buf.Len() < len(parts)*32 {
		return buf, fmt.Errorf("segment download incomplete: got %d bytes for %d parts", buf.Len(), len(parts))
	}

	total := buf.Len()
	logging.Info("runv3 downloaded %d parts, total %d bytes", len(parts), total)
	if err := validateFmp4Buffer(buf.Bytes()); err != nil {
		return buf, err
	}
	fmt.Print("Downloaded\n")
	return buf, nil
}

func decodeParts(encoded string) []fmp4Part {
	raw := strings.Split(encoded, ";")
	parts := make([]fmp4Part, 0, len(raw))
	for _, item := range raw {
		if item == "" {
			continue
		}
		parts = append(parts, decodeMVPart(item))
	}
	return parts
}

func Run(adamId string, trackpath string, authtoken string, mutoken string, mvmode bool, serverUrl string) (string, error) {
	var keystr string
	var parts []fmp4Part
	var kidBase64 string
	var uriPrefix string
	var err error
	if mvmode {
		kidBase64, parts, uriPrefix, err = parsePlaybackPlaylist(trackpath)
		if err != nil {
			return "", err
		}
	} else {
		encoded, kid, prefix, err := GetWebplayback(adamId, authtoken, mutoken, false)
		if err != nil {
			return "", err
		}
		kidBase64 = kid
		uriPrefix = prefix
		parts = decodeParts(encoded)
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, "pssh", kidBase64)
	ctx = context.WithValue(ctx, "adamId", adamId)
	ctx = context.WithValue(ctx, "uriPrefix", uriPrefix)
	pssh, err := getPSSH("", kidBase64)
	if err != nil {
		return "", wrapStage(StageLicense, err)
	}
	headers := map[string]string{
		"authorization":            "Bearer " + authtoken,
		"x-apple-music-user-token": mutoken,
	}
	client := resty.New()
	client.SetHeaders(headers)
	key := key.Key{
		ReqCli:        client,
		BeforeRequest: BeforeRequest,
		AfterRequest:  AfterRequest,
	}
	key.CdmInit()
	var keybt []byte
	licenseURL := "https://play.itunes.apple.com/WebObjects/MZPlay.woa/wa/acquireWebPlaybackLicense"
	if serverUrl != "" {
		licenseURL = serverUrl
	}
	keystr, keybt, err = key.GetKey(ctx, licenseURL, pssh, nil)
	if err != nil {
		return "", wrapStage(StageLicense, err)
	}
	if mvmode {
		keyAndUrls := "1:" + keystr + ";" + encodePartsForMV(parts)
		return keyAndUrls, nil
	}

	body, err := downloadFmp4Parts(parts)
	if err != nil {
		return "", wrapStage(StageDownload, err)
	}

	var buffer bytes.Buffer
	err = DecryptMP4(&body, keybt, &buffer)
	if err != nil {
		fmt.Print("Decryption failed\n")
		return "", wrapStage(StageDecrypt, err)
	}
	fmt.Print("Decrypted\n")

	ofh, err := os.Create(trackpath)
	if err != nil {
		return "", err
	}
	defer ofh.Close()

	_, err = ofh.Write(buffer.Bytes())
	if err != nil {
		return "", err
	}
	return "", nil
}

type Segment struct {
	Index int
	Data  []byte
}

func downloadPart(part fmp4Part, index int, wg *sync.WaitGroup, segmentsChan chan<- Segment, client *http.Client, limiter chan struct{}, errChan chan<- error) {
	defer func() {
		<-limiter
		wg.Done()
	}()

	data, err := part.fetch(client)
	if err != nil {
		select {
		case errChan <- fmt.Errorf("part %d: %w", index, err):
		default:
		}
		return
	}
	segmentsChan <- Segment{Index: index, Data: data}
}

func fetchPartWithClient(client *http.Client, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= maxDownloadTry; attempt++ {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range cdnHeaders {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.ContentLength > 0 && int64(len(data)) != resp.ContentLength {
			lastErr = fmt.Errorf("short read: got %d bytes, expected %d", len(data), resp.ContentLength)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func fileWriter(wg *sync.WaitGroup, segmentsChan <-chan Segment, outputFile io.Writer, totalSegments int) {
	defer wg.Done()

	segmentBuffer := make(map[int][]byte)
	nextIndex := 0

	for segment := range segmentsChan {
		if segment.Index == nextIndex {
			_, err := outputFile.Write(segment.Data)
			if err != nil {
				fmt.Printf("错误(分段 %d): 写入文件失败: %v\n", segment.Index, err)
			}
			nextIndex++
			for {
				data, ok := segmentBuffer[nextIndex]
				if !ok {
					break
				}
				_, err := outputFile.Write(data)
				if err != nil {
					fmt.Printf("错误(分段 %d): 从缓冲区写入文件失败: %v\n", nextIndex, err)
				}
				delete(segmentBuffer, nextIndex)
				nextIndex++
			}
		} else {
			segmentBuffer[segment.Index] = segment.Data
		}
	}

	if nextIndex != totalSegments {
		fmt.Printf("警告: 写入完成，但似乎有分段丢失。期望 %d 个, 实际写入 %d 个。\n", totalSegments, nextIndex)
	}
}

func ExtMvData(keyAndUrls string, savePath string) error {
	segments := strings.Split(keyAndUrls, ";")
	keyHex := segments[0]
	var parts []fmp4Part
	for _, raw := range segments[1:] {
		if raw == "" {
			continue
		}
		parts = append(parts, decodeMVPart(raw))
	}
	tempFile, err := os.CreateTemp("", "enc_mv_data-*.mp4")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	body, err := downloadFmp4Parts(parts)
	if err != nil {
		return err
	}
	if _, err := tempFile.Write(body.Bytes()); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	fmt.Println("\nDownloaded.")

	cmd1 := exec.Command("mp4decrypt", "--key", keyHex, tempFile.Name(), filepath.Base(savePath))
	cmd1.Dir = filepath.Dir(savePath)
	outlog, err := cmd1.CombinedOutput()
	if err != nil {
		fmt.Printf("Decrypt failed: %v\n", err)
		fmt.Printf("Output:\n%s\n", outlog)
		return err
	}
	fmt.Println("Decrypted.")
	return nil
}

func DecryptMP4(r io.Reader, key []byte, w io.Writer) error {
	inMp4, err := mp4.DecodeFile(r)
	if err != nil {
		return fmt.Errorf("failed to decode file: %w", err)
	}
	if !inMp4.IsFragmented() {
		return errors.New("file is not fragmented")
	}
	if inMp4.Init == nil {
		return errors.New("no init part of file")
	}
	decryptInfo, err := mp4.DecryptInit(inMp4.Init)
	if err != nil {
		return fmt.Errorf("failed to decrypt init: %w", err)
	}
	if err = inMp4.Init.Encode(w); err != nil {
		return fmt.Errorf("failed to write init: %w", err)
	}
	for _, seg := range inMp4.Segments {
		if err = mp4.DecryptSegment(seg, decryptInfo, key); err != nil {
			if err.Error() == "no senc box in traf" {
				err = nil
			} else {
				return fmt.Errorf("failed to decrypt segment: %w", err)
			}
		}
		if err = seg.Encode(w); err != nil {
			return fmt.Errorf("failed to encode segment: %w", err)
		}
	}
	return nil
}
