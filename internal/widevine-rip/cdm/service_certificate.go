package wv

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/protobuf/proto"
)

// FetchServiceCertificate asks a license server for its service certificate
// by posting a SERVICE_CERTIFICATE_REQUEST message. The returned bytes can be
// passed to CDM.SetServiceCertificate.
func FetchServiceCertificate(client *http.Client, licenseURL string) ([]byte, error) {
	request, err := proto.Marshal(&SignedMessage{Type: SignedMessage_SERVICE_CERTIFICATE_REQUEST.Enum()})
	if err != nil {
		return nil, err
	}
	response, err := client.Post(licenseURL, "application/x-www-form-urlencoded", bytes.NewReader(request))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch service certificate: %s", response.Status)
	}
	return io.ReadAll(response.Body)
}
