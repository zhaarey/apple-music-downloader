package wv

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/aead/cmac"
	"google.golang.org/protobuf/proto"
	"lukechampine.com/frand"
)

// PSSHBoxHeaderSize is the size of the ISO BMFF 'pssh' box header (size,
// type, version/flags, system ID, data size) that precedes the
// WidevineCencHeader payload.
const PSSHBoxHeaderSize = 32

// Key is a key decrypted from a Widevine license.
type Key struct {
	ID    []byte
	Type  License_KeyContainer_KeyType
	Value []byte
}

// CDM is a single Widevine license session: it builds one signed license
// request and decrypts the keys from the matching license response.
type CDM struct {
	device     *Device
	sessionID  [32]byte
	cencHeader *WidevineCencHeader

	// serviceCertificate enables privacy mode (encrypted client ID) when set.
	serviceCertificate *SignedDeviceCertificate
}

// NewCDM creates a license session for pssh (a complete 'pssh' box) using
// the given device.
func NewCDM(device *Device, pssh []byte) (*CDM, error) {
	if device == nil {
		return nil, errors.New("nil Widevine device")
	}
	if len(pssh) < PSSHBoxHeaderSize {
		return nil, errors.New("PSSH box is too short")
	}
	cencHeader := new(WidevineCencHeader)
	if err := proto.Unmarshal(pssh[PSSHBoxHeaderSize:], cencHeader); err != nil {
		return nil, fmt.Errorf("parse Widevine PSSH data: %w", err)
	}
	return &CDM{
		device:     device,
		sessionID:  newSessionID(),
		cencHeader: cencHeader,
	}, nil
}

// NewDefaultCDM creates a license session using the built-in device.
func NewDefaultCDM(pssh []byte) (*CDM, error) {
	device, err := DefaultDevice()
	if err != nil {
		return nil, err
	}
	return NewCDM(device, pssh)
}

// newSessionID returns a request ID in the format used by Chrome CDMs:
// 16 random upper-case hex characters, "01", then zero padding.
func newSessionID() (id [32]byte) {
	const hexChars = "ABCDEF0123456789"
	for i := 0; i < 16; i++ {
		id[i] = hexChars[frand.Intn(len(hexChars))]
	}
	copy(id[16:], "01")
	for i := 18; i < len(id); i++ {
		id[i] = '0'
	}
	return id
}

// SetServiceCertificate enables privacy mode. certData is a serialized
// SignedMessage carrying the license server's service certificate. Most
// Widevine license servers do not require this.
func (c *CDM) SetServiceCertificate(certData []byte) error {
	var message SignedMessage
	if err := proto.Unmarshal(certData, &message); err != nil {
		return fmt.Errorf("parse service certificate message: %w", err)
	}
	certificate := new(SignedDeviceCertificate)
	if err := proto.Unmarshal(message.Msg, certificate); err != nil {
		return fmt.Errorf("parse service certificate: %w", err)
	}
	c.serviceCertificate = certificate
	return nil
}

// ServiceCertificate returns the certificate set by SetServiceCertificate,
// or nil when privacy mode is off.
func (c *CDM) ServiceCertificate() *SignedDeviceCertificate {
	return c.serviceCertificate
}

// BuildLicenseRequest returns the serialized SignedLicenseRequest (the
// license "challenge") to send to the license server.
func (c *CDM) BuildLicenseRequest() ([]byte, error) {
	msg := &LicenseRequest{
		ContentId: &LicenseRequest_ContentIdentification{
			CencId: &LicenseRequest_ContentIdentification_CENC{
				Pssh:        c.cencHeader,
				LicenseType: LicenseType_DEFAULT.Enum(),
				RequestId:   c.sessionID[:],
			},
		},
		Type:            LicenseRequest_NEW.Enum(),
		RequestTime:     proto.Uint32(uint32(time.Now().Unix())),
		ProtocolVersion: ProtocolVersion_CURRENT.Enum(),
		KeyControlNonce: proto.Uint32(uint32(frand.Uint64n(math.MaxUint32))),
	}

	if c.serviceCertificate != nil {
		encryptedClientID, err := c.encryptClientID()
		if err != nil {
			return nil, err
		}
		msg.EncryptedClientId = encryptedClientID
	} else {
		clientID := new(ClientIdentification)
		if err := proto.Unmarshal(c.device.ClientID, clientID); err != nil {
			return nil, fmt.Errorf("parse device client ID: %w", err)
		}
		msg.ClientId = clientID
	}

	signature, err := c.sign(msg)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(&SignedLicenseRequest{
		Type:      SignedLicenseRequest_LICENSE_REQUEST.Enum(),
		Msg:       msg,
		Signature: signature,
	})
}

// encryptClientID encrypts the device client ID with a fresh AES privacy key,
// which is itself wrapped with the service certificate's RSA public key.
func (c *CDM) encryptClientID() (*EncryptedClientIdentification, error) {
	certificate := c.serviceCertificate.GetXDeviceCertificate()
	servicePublicKey, err := x509.ParsePKCS1PublicKey(certificate.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("parse service public key: %w", err)
	}

	var privacyKey, iv [aes.BlockSize]byte
	frand.Read(privacyKey[:])
	frand.Read(iv[:])

	block, err := aes.NewCipher(privacyKey[:])
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(c.device.ClientID, aes.BlockSize)
	encryptedClientID := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv[:]).CryptBlocks(encryptedClientID, padded)

	encryptedPrivacyKey, err := rsa.EncryptOAEP(sha1.New(), frand.Reader, servicePublicKey, privacyKey[:], nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt privacy key: %w", err)
	}

	return &EncryptedClientIdentification{
		ServiceId:                      proto.String(string(certificate.GetServiceId())),
		ServiceCertificateSerialNumber: certificate.GetSerialNumber(),
		EncryptedClientId:              encryptedClientID,
		EncryptedClientIdIv:            iv[:],
		EncryptedPrivacyKey:            encryptedPrivacyKey,
	}, nil
}

// sign returns the RSA-PSS (SHA-1) signature of msg with the device key.
func (c *CDM) sign(msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	hash := sha1.Sum(data)
	return rsa.SignPSS(frand.Reader, c.device.PrivateKey, crypto.SHA1, hash[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
}

// DecryptLicenseKeys decrypts every key container in licenseResponse.
// licenseRequest must be the exact bytes returned by BuildLicenseRequest,
// because the key-wrapping key is derived from it.
func (c *CDM) DecryptLicenseKeys(licenseRequest []byte, licenseResponse []byte) ([]Key, error) {
	var license SignedLicense
	if err := proto.Unmarshal(licenseResponse, &license); err != nil {
		return nil, fmt.Errorf("parse license response: %w", err)
	}
	var request SignedLicenseRequest
	if err := proto.Unmarshal(licenseRequest, &request); err != nil {
		return nil, fmt.Errorf("parse license request: %w", err)
	}
	requestMsg, err := proto.Marshal(request.Msg)
	if err != nil {
		return nil, err
	}

	sessionKey, err := rsa.DecryptOAEP(sha1.New(), frand.Reader, c.device.PrivateKey, license.SessionKey, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt license session key: %w", err)
	}
	encryptionKey, err := deriveEncryptionKey(sessionKey, requestMsg)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	containers := license.GetMsg().GetKey()
	keys := make([]Key, 0, len(containers))
	for _, container := range containers {
		value, err := decryptKeyContainer(block, container)
		if err != nil {
			return nil, err
		}
		keys = append(keys, Key{
			ID:    container.GetId(),
			Type:  container.GetType(),
			Value: value,
		})
	}
	return keys, nil
}

// deriveEncryptionKey derives the key that wraps the content keys in a
// license: AES-CMAC(sessionKey, 0x01 || "ENCRYPTION" || 0x00 || requestMsg || uint32be(128)).
func deriveEncryptionKey(sessionKey []byte, requestMsg []byte) ([]byte, error) {
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid license session key: %w", err)
	}
	var derivationContext []byte
	derivationContext = append(derivationContext, 0x01)
	derivationContext = append(derivationContext, "ENCRYPTION"...)
	derivationContext = append(derivationContext, 0x00)
	derivationContext = append(derivationContext, requestMsg...)
	derivationContext = append(derivationContext, 0x00, 0x00, 0x00, 0x80) // key size in bits
	return cmac.Sum(derivationContext, block, block.BlockSize())
}

// decryptKeyContainer AES-CBC decrypts and unpads one key container.
func decryptKeyContainer(block cipher.Block, container *License_KeyContainer) ([]byte, error) {
	iv, encrypted := container.GetIv(), container.GetKey()
	if len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("key container IV has invalid length %d", len(iv))
	}
	if len(encrypted) == 0 || len(encrypted)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("key container has invalid length %d", len(encrypted))
	}
	decrypted := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, encrypted)
	return pkcs7Unpad(decrypted, block.BlockSize())
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - len(data)%blockSize
	padded := make([]byte, len(data), len(data)+padLen)
	copy(padded, data)
	return append(padded, bytes.Repeat([]byte{byte(padLen)}, padLen)...)
}

// pkcs7Unpad strips PKCS#7 padding: the last byte holds the pad length, and
// every pad byte carries that same value.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("pkcs7: empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, errors.New("pkcs7: invalid padding length")
	}
	if !bytes.Equal(data[len(data)-padLen:], bytes.Repeat([]byte{byte(padLen)}, padLen)) {
		return nil, errors.New("pkcs7: invalid padding bytes")
	}
	return data[:len(data)-padLen], nil
}
