package wv

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"testing"

	"google.golang.org/protobuf/proto"
)

// TestLicenseRoundTrip plays the license server: it verifies the request
// signature, wraps a content key the way Widevine servers do, and checks
// that the CDM recovers it.
func TestLicenseRoundTrip(t *testing.T) {
	device, err := DefaultDevice()
	if err != nil {
		t.Fatal(err)
	}
	cencData, err := proto.Marshal(&WidevineCencHeader{KeyId: [][]byte{bytes.Repeat([]byte{1}, 16)}})
	if err != nil {
		t.Fatal(err)
	}
	cdm, err := NewCDM(device, append(make([]byte, PSSHBoxHeaderSize), cencData...))
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := cdm.BuildLicenseRequest()
	if err != nil {
		t.Fatal(err)
	}
	var request SignedLicenseRequest
	if err := proto.Unmarshal(challenge, &request); err != nil {
		t.Fatal(err)
	}
	requestMsg, err := proto.Marshal(request.Msg)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha1.Sum(requestMsg)
	if err := rsa.VerifyPSS(&device.PrivateKey.PublicKey, crypto.SHA1, hash[:], request.Signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}); err != nil {
		t.Fatalf("license request signature does not verify: %v", err)
	}
	if request.Msg.ClientId == nil {
		t.Fatal("license request has no client ID")
	}

	sessionKey := bytes.Repeat([]byte{2}, 16)
	contentKey := bytes.Repeat([]byte{3}, 16)
	iv := bytes.Repeat([]byte{4}, 16)
	encryptedSessionKey, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, &device.PrivateKey.PublicKey, sessionKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	wrappingKey, err := deriveEncryptionKey(sessionKey, requestMsg)
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(wrappingKey)
	if err != nil {
		t.Fatal(err)
	}
	padded := pkcs7Pad(contentKey, aes.BlockSize)
	wrappedKey := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(wrappedKey, padded)

	response, err := proto.Marshal(&SignedLicense{
		SessionKey: encryptedSessionKey,
		Msg: &License{Key: []*License_KeyContainer{{
			Id:   []byte("kid"),
			Iv:   iv,
			Key:  wrappedKey,
			Type: License_KeyContainer_CONTENT.Enum(),
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	keys, err := cdm.DecryptLicenseKeys(challenge, response)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Type != License_KeyContainer_CONTENT || !bytes.Equal(keys[0].Value, contentKey) {
		t.Fatalf("unexpected keys: %+v", keys)
	}
}

func TestPKCS7Unpad(t *testing.T) {
	if got, err := pkcs7Unpad(pkcs7Pad([]byte("abc"), 16), 16); err != nil || string(got) != "abc" {
		t.Fatalf("pkcs7Unpad(pad(abc)) = %q, %v", got, err)
	}
	for _, bad := range [][]byte{nil, {0}, {17}, {1, 2, 2, 3}} {
		if _, err := pkcs7Unpad(bad, 16); err == nil {
			t.Fatalf("pkcs7Unpad(%v) error = nil, want failure", bad)
		}
	}
}
