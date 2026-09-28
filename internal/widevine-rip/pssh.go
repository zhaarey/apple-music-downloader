package widevinerip

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"google.golang.org/protobuf/proto"

	wv "amdl/internal/widevine-rip/cdm"
)

// widevineSystemID is the DRM system ID of Widevine in 'pssh' boxes.
var widevineSystemID = [16]byte{
	0xed, 0xef, 0x8b, 0xa9, 0x79, 0xd6, 0x4a, 0xce,
	0xa3, 0xc8, 0x27, 0xdc, 0xd5, 0x1d, 0x21, 0xed,
}

// BuildPSSH builds a version-0 Widevine 'pssh' box for a single key ID.
// keyIDBase64 is the key ID as it appears in the HLS EXT-X-KEY URI; contentID
// is optional.
func BuildPSSH(keyIDBase64 string, contentID string) ([]byte, error) {
	keyID, err := base64.StdEncoding.DecodeString(keyIDBase64)
	if err != nil {
		return nil, fmt.Errorf("decode base64 key ID: %w", err)
	}
	cencHeader := &wv.WidevineCencHeader{
		KeyId:     [][]byte{keyID},
		Algorithm: wv.WidevineCencHeader_AESCTR.Enum(),
		Provider:  new(string),
		ContentId: []byte(base64.StdEncoding.EncodeToString([]byte(contentID))),
		Policy:    new(string),
	}
	data, err := proto.Marshal(cencHeader)
	if err != nil {
		return nil, fmt.Errorf("marshal WidevineCencHeader: %w", err)
	}

	box := make([]byte, wv.PSSHBoxHeaderSize, wv.PSSHBoxHeaderSize+len(data))
	binary.BigEndian.PutUint32(box[0:4], uint32(wv.PSSHBoxHeaderSize+len(data)))
	copy(box[4:8], "pssh")
	// box[8:12] stays zero: version 0, no flags.
	copy(box[12:28], widevineSystemID[:])
	binary.BigEndian.PutUint32(box[28:32], uint32(len(data)))
	return append(box, data...), nil
}
