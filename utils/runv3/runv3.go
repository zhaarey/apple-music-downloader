package runv3

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/itouakirai/mp4ff/mp4"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/protobuf/proto"

	"main/internal/logging"
	cdm "main/utils/runv3/cdm"
	key "main/utils/runv3/key"
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

type Segment struct {
	Index int
	Data  []byte
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
	return base64.StdEncoding.EncodeToString(widevineCenc), nil
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
	return cl.R().SetContext(ctx).SetBody(jsondata).Post(url)
}

func AfterRequest(response *resty.Response) ([]byte, error) {
	var responseData PlaybackLicense
	if err := json.Unmarshal(response.Body(), &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %v", err)
	}
	if responseData.ErrorCode != 0 || responseData.Status != 0 {
		return nil, fmt.Errorf("error in license response, code: %d, status: %d", responseData.ErrorCode, responseData.Status)
	}
	return base64.StdEncoding.DecodeString(responseData.License)
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

func downloadFmp4Parts(parts []FMP4Part) (bytes.Buffer, error) {
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
	logging.Info("runv3 downloaded %d parts, total %d bytes", len(parts), buf.Len())
	if err := validateFmp4Buffer(buf.Bytes()); err != nil {
		return buf, err
	}
	fmt.Print("Downloaded\n")
	return buf, nil
}

func Run(adamId string, trackpath string, authtoken string, mutoken string, mvmode bool, serverUrl string) (string, error) {
	var playback WebPlayback
	var err error
	if mvmode {
		playback, err = parsePlaybackPlaylist(trackpath)
	} else {
		playback, err = GetSongWebPlayback(adamId, authtoken, mutoken)
	}
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "pssh", playback.KidBase64)
	ctx = context.WithValue(ctx, "adamId", adamId)
	ctx = context.WithValue(ctx, "uriPrefix", playback.UriPrefix)

	pssh, err := getPSSH("", playback.KidBase64)
	if err != nil {
		return "", wrapStage(StageLicense, err)
	}

	client := resty.New()
	client.SetHeaders(map[string]string{
		"authorization":            "Bearer " + authtoken,
		"x-apple-music-user-token": mutoken,
	})
	key := key.Key{
		ReqCli:        client,
		BeforeRequest: BeforeRequest,
		AfterRequest:  AfterRequest,
	}
	key.CdmInit()

	licenseURL := "https://play.itunes.apple.com/WebObjects/MZPlay.woa/wa/acquireWebPlaybackLicense"
	if serverUrl != "" {
		licenseURL = serverUrl
	}
	keystr, keybt, err := key.GetKey(ctx, licenseURL, pssh, nil)
	if err != nil {
		return "", wrapStage(StageLicense, err)
	}
	if mvmode {
		return "1:" + keystr + ";" + EncodeParts(playback.Parts), nil
	}

	body, err := downloadFmp4Parts(playback.Parts)
	if err != nil {
		return "", wrapStage(StageDownload, err)
	}

	var buffer bytes.Buffer
	if err := DecryptMP4(&body, keybt, &buffer); err != nil {
		fmt.Print("Decryption failed\n")
		return "", wrapStage(StageDecrypt, err)
	}
	fmt.Print("Decrypted\n")

	ofh, err := os.Create(trackpath)
	if err != nil {
		return "", err
	}
	defer ofh.Close()
	if _, err = ofh.Write(buffer.Bytes()); err != nil {
		return "", err
	}
	return "", nil
}

func downloadPart(part FMP4Part, index int, wg *sync.WaitGroup, segmentsChan chan<- Segment, client *http.Client, limiter chan struct{}, errChan chan<- error) {
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

func fileWriter(wg *sync.WaitGroup, segmentsChan <-chan Segment, outputFile io.Writer, totalSegments int) {
	defer wg.Done()
	segmentBuffer := make(map[int][]byte)
	nextIndex := 0
	for segment := range segmentsChan {
		if segment.Index == nextIndex {
			_, _ = outputFile.Write(segment.Data)
			nextIndex++
			for {
				data, ok := segmentBuffer[nextIndex]
				if !ok {
					break
				}
				_, _ = outputFile.Write(data)
				delete(segmentBuffer, nextIndex)
				nextIndex++
			}
		} else {
			segmentBuffer[segment.Index] = segment.Data
		}
	}
	if nextIndex != totalSegments {
		fmt.Printf("warning: expected %d segments, wrote %d\n", totalSegments, nextIndex)
	}
}

func ExtMvData(keyAndUrls string, savePath string) error {
	segments := strings.Split(keyAndUrls, ";")
	keyHex := segments[0]
	parts := make([]FMP4Part, 0, len(segments)-1)
	for _, raw := range segments[1:] {
		if raw == "" {
			continue
		}
		parts = append(parts, decodePartToken(raw))
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
			if err.Error() != "no senc box in traf" {
				return fmt.Errorf("failed to decrypt segment: %w", err)
			}
			err = nil
		}
		if err = seg.Encode(w); err != nil {
			return fmt.Errorf("failed to encode segment: %w", err)
		}
	}
	return nil
}
