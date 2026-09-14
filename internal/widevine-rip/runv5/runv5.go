package runv5

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	widevine "amdl/internal/widevine-rip"
	wv "amdl/internal/widevine-rip/key"
	"amdl/internal/wrapper"

	"github.com/go-resty/resty/v2"
)

// PlaybackLicense is the wrapper-lite /license response envelope.
type PlaybackLicense = wrapper.PlaybackLicense

// BeforeRequest posts the license challenge to wrapper-lite /license. The
// server fills in the extra fields expected by Apple before forwarding.
func BeforeRequest(cl *resty.Client, ctx context.Context, url string, body []byte) (*resty.Response, error) {
	return wrapper.WidevineBeforeRequest(cl, ctx, url, body)
}

// AfterRequest unwraps the lite-server license response.
func AfterRequest(response *resty.Response) ([]byte, error) {
	return wrapper.WidevineAfterRequest(response)
}

// GetWebplayback obtains playback from wrapper-lite instead of Apple's web
// playback endpoint, so it does not require a media user token.
func GetWebplayback(adamId string, liteServer string, mvmode bool) (string, string, string, error) {
	m3u8, err := wrapper.GetWebplayback(liteServer, adamId)
	if err != nil {
		return "", "", "", err
	}
	if mvmode {
		return m3u8, "", "", nil
	}
	kidBase64, fileurl, uriPrefix, err := widevine.ExtractKidBase64(m3u8, false)
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
