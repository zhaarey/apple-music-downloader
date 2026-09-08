package wv

import (
	"bytes"
	"io"
	"net/http"
)

func GetCertData(client *http.Client, licenseURL string) ([]byte, error) {
	response, err := client.Post(licenseURL, "application/x-www-form-urlencoded", bytes.NewReader([]byte{0x08, 0x04}))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}
