package playreadyrip

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	widevine "amdl/internal/widevine-rip"
	"amdl/internal/wrapper"

	puppyready "git.gay/itouakirai/puppyready"
)

const playReadyKeyFormat = "com.microsoft.playready"

// Run obtains a PlayReady content key from wrapper-lite and returns the same
// key-and-URLs format used by the existing MV downloader.
func Run(adamID string, playlistURL string, liteServerURL string) (string, error) {
	if liteServerURL == "" {
		return "", errors.New("lite-server is not configured")
	}

	keyPayload, fileURLs, uriPrefix, err := widevine.ExtractKeyAndURLs(playlistURL, playReadyKeyFormat, true)
	if err != nil {
		return "", err
	}

	key, err := getContentKey(adamID, keyPayload, uriPrefix, liteServerURL)
	if err != nil {
		return "", err
	}

	return "1:" + hex.EncodeToString(key) + ";" + fileURLs, nil
}

func getContentKey(adamID string, keyPayload string, uriPrefix string, liteServerURL string) ([]byte, error) {
	device, err := puppyready.DefaultDevice()
	if err != nil {
		return nil, fmt.Errorf("initialize PlayReady device: %w", err)
	}

	cdm := puppyready.NewCDM(device)
	sessionID, err := cdm.Open()
	if err != nil {
		return nil, fmt.Errorf("open PlayReady session: %w", err)
	}
	defer cdm.Close(sessionID)

	wrmHeader, licenseURI, err := buildWRMHeader(keyPayload, uriPrefix)
	if err != nil {
		return nil, err
	}

	challenge, err := cdm.GetLicenseChallenge(sessionID, wrmHeader)
	if err != nil {
		return nil, fmt.Errorf("create PlayReady challenge: %w", err)
	}

	licenseXML, err := requestLicense(adamID, base64.StdEncoding.EncodeToString([]byte(challenge)), licenseURI, liteServerURL)
	if err != nil {
		return nil, err
	}
	if err := cdm.ParseLicense(sessionID, licenseXML); err != nil {
		return nil, fmt.Errorf("parse PlayReady license: %w", err)
	}

	keys, err := cdm.GetKeys(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get PlayReady content key: %w", err)
	}
	if len(keys) == 0 {
		return nil, errors.New("PlayReady license returned no content keys")
	}
	if len(keys[0].Key) == 0 {
		return nil, errors.New("PlayReady license returned an empty content key")
	}
	return append([]byte(nil), keys[0].Key...), nil
}

func buildWRMHeader(keyPayload string, uriPrefix string) (string, string, error) {
	licenseURI := uriPrefix + "," + keyPayload
	if pssh, err := puppyready.ParsePSSH(keyPayload); err == nil {
		if len(pssh.WRMHeaders) == 0 {
			return "", "", errors.New("PlayReady PSSH contains no WRM header")
		}
		return pssh.WRMHeaders[0], licenseURI, nil
	}

	decodedInput, err := base64.StdEncoding.DecodeString(keyPayload)
	if err != nil {
		return "", "", fmt.Errorf("decode PlayReady PSSH/KID: %w", err)
	}
	if len(decodedInput) != 16 {
		return "", "", errors.New("PlayReady key data is neither PSSH nor a 16-byte KID")
	}

	wrmHeader := `<WRMHEADER xmlns="http://schemas.microsoft.com/DRM/2007/03/PlayReadyHeader" version="4.0.0.0"><DATA><PROTECTINFO><KEYLEN>16</KEYLEN><ALGID>AESCTR</ALGID></PROTECTINFO><KID>` + keyPayload + `</KID></DATA></WRMHEADER>`
	return wrmHeader, "data:;base64," + keyPayload, nil
}

func requestLicense(adamID string, challenge string, uri string, liteServerURL string) ([]byte, error) {
	return wrapper.GetPlayReadyLicense(liteServerURL, adamID, challenge, uri)
}
