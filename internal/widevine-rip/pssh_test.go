package widevinerip

import (
	"bytes"
	"encoding/binary"
	"testing"

	"google.golang.org/protobuf/proto"

	wv "amdl/internal/widevine-rip/cdm"
)

func TestBuildPSSH(t *testing.T) {
	keyID := bytes.Repeat([]byte{0xab}, 16)
	box, err := BuildPSSH("q6urq6urq6urq6urq6urqw==", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.BigEndian.Uint32(box[0:4]); int(got) != len(box) {
		t.Fatalf("box size = %d, want %d", got, len(box))
	}
	if string(box[4:8]) != "pssh" {
		t.Fatalf("box type = %q, want pssh", box[4:8])
	}
	if !bytes.Equal(box[12:28], widevineSystemID[:]) {
		t.Fatalf("unexpected system ID %x", box[12:28])
	}
	if got := binary.BigEndian.Uint32(box[28:32]); int(got) != len(box)-wv.PSSHBoxHeaderSize {
		t.Fatalf("data size = %d, want %d", got, len(box)-wv.PSSHBoxHeaderSize)
	}

	var header wv.WidevineCencHeader
	if err := proto.Unmarshal(box[wv.PSSHBoxHeaderSize:], &header); err != nil {
		t.Fatal(err)
	}
	if len(header.KeyId) != 1 || !bytes.Equal(header.KeyId[0], keyID) {
		t.Fatalf("unexpected key IDs %x", header.KeyId)
	}
}

func TestBuildPSSHRejectsInvalidKeyID(t *testing.T) {
	if _, err := BuildPSSH("not base64!", ""); err == nil {
		t.Fatal("expected invalid key ID to fail")
	}
}
