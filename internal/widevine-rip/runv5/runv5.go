package runv5

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"amdl/internal/download"
	widevine "amdl/internal/widevine-rip"
	wv "amdl/internal/widevine-rip/key"

	"github.com/go-resty/resty/v2"
)

// PlaybackLicense is the wrapper-lite /license response envelope. The Apple
// endpoint used by the shared backend returns a flatter payload instead.
type PlaybackLicense struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AdamId  string `json:"adamId"`
		License string `json:"license"`
		Renew   int    `json:"renew"`
	} `json:"data"`
}

// BeforeRequest posts the license challenge to wrapper-lite /license. The
// server fills in the extra fields expected by Apple before forwarding.
func BeforeRequest(cl *resty.Client, ctx context.Context, url string, body []byte) (*resty.Response, error) {
	uri := ctx.Value("uriPrefix").(string) + "," + ctx.Value("pssh").(string)
	jsondata := map[string]interface{}{
		"challenge": base64.StdEncoding.EncodeToString(body),
		"uri":       uri,
		"adamId":    ctx.Value("adamId").(string),
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

// AfterRequest unwraps the lite-server license response.
func AfterRequest(response *resty.Response) ([]byte, error) {
	var responseData PlaybackLicense
	if err := json.Unmarshal(response.Body(), &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}
	if responseData.Code != 0 {
		return nil, fmt.Errorf("lite-server /license returned code=%d msg=%s", responseData.Code, responseData.Msg)
	}
	if responseData.Data.License == "" {
		return nil, errors.New("empty license in lite-server response")
	}
	license, err := base64.StdEncoding.DecodeString(responseData.Data.License)
	if err != nil {
		return nil, fmt.Errorf("failed to decode license: %w", err)
	}
	return license, nil
}

// GetWebplayback obtains playback from wrapper-lite instead of Apple's web
// playback endpoint, so it does not require a media user token.
func GetWebplayback(adamId string, liteServer string, mvmode bool) (string, string, string, error) {
	if liteServer == "" {
		return "", "", "", errors.New("lite-server is not configured")
	}
	endpoint := strings.TrimRight(liteServer, "/") + "/webplayback?adamId=" + url.QueryEscape(adamId)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", "", err
	}
	resp, err := download.Client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("lite-server /webplayback returned %s", resp.Status)
	}

	var envelope struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			M3u8 string `json:"m3u8"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", "", "", err
	}
	if envelope.Code != 0 {
		return "", "", "", fmt.Errorf("lite-server /webplayback returned code=%d msg=%s", envelope.Code, envelope.Msg)
	}
	if envelope.Data.M3u8 == "" {
		return "", "", "", errors.New("Unavailable")
	}
	if mvmode {
		return envelope.Data.M3u8, "", "", nil
	}
	kidBase64, fileurl, uriPrefix, err := widevine.ExtractKidBase64(envelope.Data.M3u8, false)
	if err != nil {
		return "", "", "", err
	}
	return fileurl, kidBase64, uriPrefix, nil
}

// Run keeps the signature used by the catalog orchestrators. authtoken and
// mutoken are ignored by the lite-server backend but retained for callers.
func Run(adamId string, trackpath string, authtoken string, mvmode bool, liteServerUrl string) (string, error) {
	if liteServerUrl == "" {
		return "", errors.New("lite-server is not configured")
	}

	var keystr string
	var fileurl, kidBase64, uriPrefix string
	var err error
	if mvmode {
		kidBase64, fileurl, uriPrefix, err = widevine.ExtractKidBase64(trackpath, true)
	} else {
		fileurl, kidBase64, uriPrefix, err = GetWebplayback(adamId, liteServerUrl, false)
	}
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "pssh", kidBase64)
	ctx = context.WithValue(ctx, "adamId", adamId)
	ctx = context.WithValue(ctx, "uriPrefix", uriPrefix)

	pssh, err := widevine.GetPSSH("", kidBase64)
	if err != nil {
		return "", err
	}

	key := wv.NewKey(BeforeRequest, AfterRequest)
	keystr, keybt, err := key.GetKey(ctx, liteServerUrl+"/license", pssh, nil)
	if err != nil {
		return "", err
	}
	if mvmode {
		return "1:" + keystr + ";" + fileurl, nil
	}

	body, err := widevine.Extsong(fileurl)
	if err != nil {
		return "", err
	}
	fmt.Print("Downloaded\n")
	var buffer bytes.Buffer
	if err := widevine.DecryptMP4(body, keybt, &buffer); err != nil {
		fmt.Print("Decryption failed\n")
		return "", err
	}
	fmt.Print("Decrypted\n")
	if err := widevine.WriteDecryptedMP4(bytes.NewReader(buffer.Bytes()), keybt, trackpath); err != nil {
		return "", err
	}
	return "", nil
}

func ExtMvData(keyAndUrls string, savePath string) error {
	return widevine.ExtMvData(keyAndUrls, savePath)
}
