package runv3

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gospider007/requests"
	"google.golang.org/protobuf/proto"
	//"log/slog"
	"os"
	cdm "main/utils/runv3/cdm"
	key "main/utils/runv3/key"

	"github.com/Eyevinn/mp4ff/mp4"
	"bytes"
	"io"
	"errors"

	//"io/ioutil"
	"net/http"
	"encoding/json"
	"github.com/grafov/m3u8"
	"strings"
	"github.com/schollz/progressbar/v3"
)

type PlaybackLicense struct {
	ErrorCode  int    `json:"errorCode"`
	License    string `json:"license"`
	RenewAfter int    `json:"renew-after"`
	Status     int    `json:"status"`
}

// func log() {
// 	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
// 	if err != nil {
// 		slog.Error("error opening file: %s", err)
// 	}
// 	defer func(f *os.File) {
// 		err := f.Close()
// 		if err != nil {
// 			slog.Error("error closing file: %s", err)
// 		}
// 	}(f)
// 	opts := slog.HandlerOptions{
// 		AddSource: true,
// 		Level:     slog.LevelDebug,
// 	}
// 	logger := slog.New(slog.NewJSONHandler(os.Stdout, &opts))
// 	slog.SetDefault(logger)
//}
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
	//最前面添加32字节
	widevineCenc = append([]byte("0123456789abcdef0123456789abcdef"), widevineCenc...)
	pssh := base64.StdEncoding.EncodeToString(widevineCenc)
	return pssh, nil
}
func BeforeRequest(cl *requests.Client, preCtx context.Context, method string, href string, options ...requests.RequestOption) (resp *requests.Response, err error) {
	data := options[0].Data
	jsondata := map[string]interface{}{
		"challenge":      base64.StdEncoding.EncodeToString(data.([]byte)),
		"key-system":     "com.widevine.alpha",
		"uri":            "data:;base64," + preCtx.Value("pssh").(string),
		"adamId":         preCtx.Value("adamId").(string),
		"isLibrary":      false,
		"user-initiated": true,
	}
	options[0].Data = nil
	options[0].Json = jsondata
	resp, err = cl.Request(preCtx, method, href, options...)
	if err != nil {
		fmt.Println(err)
	}

	return
}
func AfterRequest(Response *requests.Response) ([]byte, error) {
	var ResponseData PlaybackLicense
	_, err := Response.Json(&ResponseData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	if ResponseData.ErrorCode != 0 || ResponseData.Status != 0 {
		return nil, fmt.Errorf("error code: %d", ResponseData.ErrorCode)
	}
	License, err := base64.StdEncoding.DecodeString(ResponseData.License)
	if err != nil {
		return nil, fmt.Errorf("failed to decode license: %v", err)
	}
	return License, nil
}
func getWebplayback(adamId string, authtoken string, mutoken string) (string, string, error) {
	url := "https://play.music.apple.com/WebObjects/MZPlay.woa/wa/webPlayback"
	postData := map[string]string{
		"salableAdamId": adamId,
	}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return "", "", err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", "https://music.apple.com/")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authtoken))
	req.Header.Set("x-apple-music-user-token", mutoken)
	// 创建 HTTP 客户端
	//client := &http.Client{}
	resp, err := http.DefaultClient.Do(req)
	// 发送请求
	//resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", "", err
	}
	defer resp.Body.Close()
	//fmt.Println("Response Status:", resp.Status)
	obj := new(Songlist)
	err = json.NewDecoder(resp.Body).Decode(&obj)
	if err != nil {
		fmt.Println("json err:", err)
		return "", "", err
	}
	if len(obj.List) > 0 {
		// 遍历 Assets
		for i, _ := range obj.List[0].Assets {
			if obj.List[0].Assets[i].Flavor == "28:ctrp256" {
				kidBase64, fileurl, err := extractKidBase64(obj.List[0].Assets[i].URL)
				if err != nil {
					return "", "", err
				}
				return fileurl, kidBase64, nil
			}
			continue
		}
	}
	return "", "", nil
}
type Songlist struct {
	List []struct {
		Hlsurl string `json:"hls-key-cert-url"`
		Assets []struct {
			Flavor string `json:"flavor"`
			URL string `json:"URL"`
		}`json:"assets"`
	}`json:"songList"`
	Status int `json:"status"`
}

func extractKidBase64(b string) (string, string, error) {
	resp, err := http.Get(b)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	masterString := string(body)
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(masterString), true)
	if err != nil {
		return "", "", err
	}
	var kidbase64 string
	var fileurl string
	if listType == m3u8.MEDIA {
		mediaPlaylist := from.(*m3u8.MediaPlaylist)
		if mediaPlaylist.Key != nil {
			split := strings.Split(mediaPlaylist.Key.URI, ",")
			kidbase64 = split[1]
			lastSlashIndex := strings.LastIndex(b, "/")
			// 截取最后一个斜杠之前的部分
			fileurl = b[:lastSlashIndex] + "/" + mediaPlaylist.Map.URI
			//fmt.Println("Extracted URI:", mediaPlaylist.Map.URI)
		} else {
			fmt.Println("No key information found")
		}
	} else {
		fmt.Println("Not a media playlist")
	}
	return kidbase64, fileurl, nil
}
func extsong(b string)(bytes.Buffer){
	resp, err := http.Get(b)
	if err != nil {
		fmt.Printf("下载文件失败: %v\n", err)
	}
	defer resp.Body.Close()
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
		//progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "",
			SaucerHead:    "",
			SaucerPadding: "",
			BarStart:      "",
			BarEnd:        "",
		}),
	)
	io.Copy(io.MultiWriter(&buffer, bar), resp.Body)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
	}
	return buffer
}
func Run(adamId string, trackpath string, authtoken string, mutoken string)(error) {

	fileurl, kidBase64, err := getWebplayback(adamId, authtoken, mutoken)
	if err != nil {
		return err
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, "pssh", kidBase64)
	ctx = context.WithValue(ctx, "adamId", adamId)
	pssh, err := getPSSH("", kidBase64)
	//fmt.Println(pssh)
	if err != nil {
		fmt.Println(err)
		return err
	}
	headers := map[string]interface{}{
		"authorization": "Bearer " + authtoken,
		"x-apple-music-user-token": mutoken,
	}
	client, _ := requests.NewClient(nil, requests.ClientOption{
		Headers: headers,
	})
	key := key.Key{
		ReqCli:        client,
		BeforeRequest: BeforeRequest,
		AfterRequest:  AfterRequest,
	}
	key.CdmInit()
	_, keybt, err := key.GetKey(ctx, "https://play.itunes.apple.com/WebObjects/MZPlay.woa/wa/acquireWebPlaybackLicense", pssh, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Print("Downloading...\n")
	body := extsong(fileurl)
	//bodyReader := bytes.NewReader(body)
	var buffer bytes.Buffer

	err = DecryptMP4(&body, keybt, &buffer)
	if err != nil {
		fmt.Print("Decryption failed\n")
		return err
	} else {
		fmt.Print("Decrypted\n")
	}
	// create output file
	ofh, err := os.Create(trackpath)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return err
	}
	defer ofh.Close()

	_, err = ofh.Write(buffer.Bytes())
	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		return err
	}
	return nil
}
// DecryptMP4Auto decrypts a fragmented MP4 file with the set of keys retreived from the widevice license
// by automatically selecting the appropriate key. Supports CENC and CBCS schemes.
// func DecryptMP4Auto(r io.Reader, keys []*Key, w io.Writer) error {
// 	// Extract content key
// 	var key []byte
// 	for _, k := range keys {
// 		if k.Type == wvpb.License_KeyContainer_CONTENT {
// 			key = k.Key
// 			break
// 		}
// 	}
// 	if key == nil {
// 		return fmt.Errorf("no %s key type found in the provided key set", wvpb.License_KeyContainer_CONTENT)
// 	}
// 	// Execute decryption
// 	return DecryptMP4(r, key, w)
// }

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
