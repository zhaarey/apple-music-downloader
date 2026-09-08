package widevinerip

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/proto"

	cdm "amdl/internal/widevine-rip/cdm"
	key "amdl/internal/widevine-rip/key"
	"os"

	"bytes"
	"errors"
	"io"

	"github.com/itouakirai/mp4ff/mp4"

	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/grafov/m3u8"
	"github.com/schollz/progressbar/v3"

	"amdl/internal/download"
)

type PlaybackLicense struct {
	ErrorCode  int    `json:"errorCode"`
	License    string `json:"license"`
	RenewAfter int    `json:"renew-after"`
	Status     int    `json:"status"`
}

func WriteDecryptedMP4(r io.Reader, key []byte, path string) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	tmpPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = tempFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tempFile, r); err != nil {
		return fmt.Errorf("write temporary output: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace output: %w", err)
	}
	cleanup = false
	return nil
}

func GetPSSH(contentId string, kidBase64 string) (string, error) {
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
	//最前面添加32字节
	widevineCenc = append([]byte("0123456789abcdef0123456789abcdef"), widevineCenc...)
	pssh := base64.StdEncoding.EncodeToString(widevineCenc)
	return pssh, nil
}

func BeforeRequest(cl *resty.Client, ctx context.Context, url string, body []byte) (*resty.Response, error) {
	jsondata := map[string]interface{}{
		"challenge":      base64.StdEncoding.EncodeToString(body), // 'body' is passed in directly
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

func GetPlaybackHeaders(authtoken string, mutoken string) map[string]string {
	headers := map[string]string{
		"User-Agent":          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Origin":              "https://music.apple.com",
		"Referer":             "https://music.apple.com/",
		"Accept":              "application/vnd.apple.mpegurl,application/x-mpegURL,text/plain;q=0.8,*/*;q=0.5",
		"X-Apple-Store-Front": "143441-1,25",
	}
	if mutoken != "" {
		headers["x-apple-music-user-token"] = mutoken
		headers["Media-User-Token"] = mutoken
	}
	return headers
}

func GetURLWithHeaders(url string, authtoken string, mutoken string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range GetPlaybackHeaders(authtoken, mutoken) {
		req.Header.Set(key, value)
	}
	resp, err := download.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func GetWebplayback(adamId string, authtoken string, mutoken string, mvmode bool) (string, string, string, error) {
	url := "https://play.music.apple.com/WebObjects/MZPlay.woa/wa/webPlayback"
	postData := map[string]string{
		"salableAdamId": adamId,
	}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return "", "", "", err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", "https://music.apple.com/")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authtoken))
	req.Header.Set("x-apple-music-user-token", mutoken)
	// 创建 HTTP 客户端
	//client := &http.Client{}
	resp, err := download.Client.Do(req)
	// 发送请求
	//resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", "", "", err
	}
	defer resp.Body.Close()
	//fmt.Println("Response Status:", resp.Status)
	obj := new(Songlist)
	err = json.NewDecoder(resp.Body).Decode(&obj)
	if err != nil {
		fmt.Println("json err:", err)
		return "", "", "", err
	}
	if len(obj.List) > 0 {
		if mvmode {
			return obj.List[0].HlsPlaylistUrl, "", "", nil
		}
		// 遍历 Assets
		for i := range obj.List[0].Assets {
			if obj.List[0].Assets[i].Flavor == "28:ctrp256" {
				kidBase64, fileurl, uriPrefix, err := ExtractKidBase64(obj.List[0].Assets[i].URL, false)
				if err != nil {
					return "", "", "", err
				}
				return fileurl, kidBase64, uriPrefix, nil
			}
			continue
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

func ResolveStationVariantPlaylist(masterURL string, authtoken string, mutoken string) (string, error) {
	body, err := GetURLWithHeaders(masterURL, authtoken, mutoken)
	if err != nil {
		return "", err
	}
	masterString := string(body)
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(masterString), true)
	if err != nil {
		return "", err
	}
	if listType != m3u8.MASTER {
		return masterURL, nil
	}
	masterPlaylist := from.(*m3u8.MasterPlaylist)
	var preferred string
	for _, variant := range masterPlaylist.Variants {
		if variant == nil {
			continue
		}
		uri := variant.URI
		if strings.Contains(uri, "256") || strings.Contains(uri, "256_6") {
			preferred = uri
			break
		}
		if preferred == "" {
			preferred = uri
		}
	}
	if preferred == "" {
		return masterURL, nil
	}
	if strings.HasPrefix(preferred, "http") {
		return preferred, nil
	}
	lastSlashIndex := strings.LastIndex(masterURL, "/")
	if lastSlashIndex == -1 {
		return masterURL, nil
	}
	return masterURL[:lastSlashIndex+1] + preferred, nil
}

func ExtractKidBase64(b string, mvmode bool) (string, string, string, error) {
	req, err := http.NewRequest(http.MethodGet, b, nil)
	if err != nil {
		return "", "", "", err
	}
	resp, err := download.Client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", errors.New(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}
	masterString := string(body)
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(masterString), true)
	if err != nil {
		return "", "", "", err
	}
	var kidbase64 string
	var uriPrefix string
	var urlBuilder strings.Builder
	if listType == m3u8.MEDIA {
		mediaPlaylist := from.(*m3u8.MediaPlaylist)
		if mediaPlaylist.Key != nil {
			split := strings.Split(mediaPlaylist.Key.URI, ",")
			uriPrefix = split[0]
			kidbase64 = split[1]
			lastSlashIndex := strings.LastIndex(b, "/")
			// 截取最后一个斜杠之前的部分
			urlBuilder.WriteString(b[:lastSlashIndex])
			urlBuilder.WriteString("/")
			urlBuilder.WriteString(mediaPlaylist.Map.URI)
			//fileurl = b[:lastSlashIndex] + "/" + mediaPlaylist.Map.URI
			//fmt.Println("Extracted URI:", mediaPlaylist.Map.URI)
			if mvmode {
				for _, segment := range mediaPlaylist.Segments {
					if segment != nil {
						//fmt.Println("Extracted URI:", segment.URI)
						urlBuilder.WriteString(";")
						urlBuilder.WriteString(b[:lastSlashIndex])
						urlBuilder.WriteString("/")
						urlBuilder.WriteString(segment.URI)
						//fileurl = fileurl + ";" + b[:lastSlashIndex] + "/" + segment.URI
					}
				}
			}
		} else {
			fmt.Println("No key information found")
		}
	} else {
		fmt.Println("Not a media playlist")
	}
	return kidbase64, urlBuilder.String(), uriPrefix, nil
}
func Extsong(b string) (*bytes.Buffer, error) {
	req, err := http.NewRequest(http.MethodGet, b, nil)
	if err != nil {
		return nil, fmt.Errorf("create media request: %w", err)
	}
	resp, err := download.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download media: %s", resp.Status)
	}
	var buffer bytes.Buffer
	bar := progressbar.NewOptions64(
		resp.ContentLength,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "",
			SaucerHead:    "",
			SaucerPadding: "",
			BarStart:      "",
			BarEnd:        "",
		}),
	)
	if _, err := io.Copy(io.MultiWriter(&buffer, bar), resp.Body); err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	return &buffer, nil
}
func Run(adamId string, trackpath string, authtoken string, mutoken string, mvmode bool, serverUrl string) (string, error) {
	var keystr string //for mv key
	var fileurl string
	var kidBase64 string
	var uriPrefix string
	var err error
	if mvmode {
		kidBase64, fileurl, uriPrefix, err = ExtractKidBase64(trackpath, true)
		if err != nil {
			return "", err
		}
	} else {
		fileurl, kidBase64, uriPrefix, err = GetWebplayback(adamId, authtoken, mutoken, false)
		if err != nil {
			return "", err
		}
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, "pssh", kidBase64)
	ctx = context.WithValue(ctx, "adamId", adamId)
	ctx = context.WithValue(ctx, "uriPrefix", uriPrefix)
	pssh, err := GetPSSH("", kidBase64)
	//fmt.Println(pssh)
	if err != nil {
		fmt.Println(err)
		return "", err
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
	if serverUrl != "" {
		keystr, keybt, err = key.GetKey(ctx, serverUrl, pssh, nil)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	} else {
		keystr, keybt, err = key.GetKey(ctx, "https://play.itunes.apple.com/WebObjects/MZPlay.woa/wa/acquireWebPlaybackLicense", pssh, nil)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	}
	if mvmode {
		keyAndUrls := "1:" + keystr + ";" + fileurl
		return keyAndUrls, nil
	}
	body, err := Extsong(fileurl)
	if err != nil {
		return "", err
	}
	fmt.Print("Downloaded\n")
	//bodyReader := bytes.NewReader(body)
	var buffer bytes.Buffer

	err = DecryptMP4(body, keybt, &buffer)
	if err != nil {
		fmt.Print("Decryption failed\n")
		return "", err
	} else {
		fmt.Print("Decrypted\n")
	}
	if err := WriteDecryptedMP4(bytes.NewReader(buffer.Bytes()), keybt, trackpath); err != nil {
		return "", err
	}
	return "", nil
}

func ExtMvData(keyAndUrls string, savePath string) error {
	segments := strings.Split(keyAndUrls, ";")
	key := segments[0]
	urls := segments[1:]
	if key == "" {
		return errors.New("empty decryption key")
	}
	if len(urls) == 0 {
		return errors.New("no download URLs")
	}
	tempFile, err := os.CreateTemp("", "enc_mv_data-*.mp4")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	bar := progressbar.DefaultBytes(-1, "Downloading...")
	err = download.DownloadSegments(context.Background(), download.Client, urls, tempFile, download.SegmentConfig{
		Concurrency: 10,
		MaxRetries:  3,
		Progress:    func(n int) { _ = bar.Add(n) },
	})
	if err != nil {
		return fmt.Errorf("download stream: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	fmt.Println("\nDownloaded.")

	keyHex := key
	if i := strings.Index(keyHex, ":"); i >= 0 {
		keyHex = keyHex[i+1:]
	}
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("decode decryption key: %w", err)
	}

	inFile, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	if err := DecryptMP4(inFile, keyBytes, outFile); err != nil {
		return fmt.Errorf("decrypt stream: %w", err)
	}

	fmt.Println("Decrypted.")
	return nil
}

// DecryptMP4 decrypts a fragmented MP4 file with keys from widevice license. Supports CENC and CBCS schemes.
func DecryptMP4(r io.Reader, key []byte, w io.Writer) error {
	// Initialization
	inMp4, err := mp4.DecodeFile(r)
	if err != nil {
		return fmt.Errorf("failed to decode file: %w", err)
	}
	if !inMp4.IsFragmented() {
		return errors.New("file is not fragmented")
	}
	// Handle init segment
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
	// Decode segments
	for _, seg := range inMp4.Segments {
		if err = mp4.DecryptSegment(seg, decryptInfo, key); err != nil {
			if err.Error() == "no senc box in traf" {
				// No SENC box, skip decryption for this segment as samples can have
				// unencrypted segments followed by encrypted segments. See:
				// https://github.com/iyear/gowidevine/pull/26#issuecomment-2385960551
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
