// Package key acquires Widevine content keys: it runs one CDM license
// session and delegates the network round trip to a pluggable ExchangeFunc,
// so each license backend (Apple, wrapper-lite, ...) only supplies transport.
package key

import (
	"context"
	"errors"
	"fmt"

	wv "amdl/internal/widevine-rip/cdm"
)

// ExchangeFunc sends a serialized Widevine license request (the challenge)
// to a license server and returns the serialized SignedLicense it answers
// with, already unwrapped from any backend-specific envelope.
type ExchangeFunc func(ctx context.Context, challenge []byte) ([]byte, error)

// AcquireContentKey obtains the content key for pssh using the built-in
// device.
func AcquireContentKey(ctx context.Context, pssh []byte, exchange ExchangeFunc) ([]byte, error) {
	device, err := wv.DefaultDevice()
	if err != nil {
		return nil, fmt.Errorf("load Widevine device: %w", err)
	}
	return AcquireContentKeyWithDevice(ctx, device, pssh, exchange)
}

// AcquireContentKeyWithDevice obtains the content key for pssh using device.
func AcquireContentKeyWithDevice(ctx context.Context, device *wv.Device, pssh []byte, exchange ExchangeFunc) ([]byte, error) {
	if exchange == nil {
		return nil, errors.New("no license exchange configured")
	}
	cdm, err := wv.NewCDM(device, pssh)
	if err != nil {
		return nil, fmt.Errorf("create Widevine session: %w", err)
	}
	challenge, err := cdm.BuildLicenseRequest()
	if err != nil {
		return nil, fmt.Errorf("build license request: %w", err)
	}
	license, err := exchange(ctx, challenge)
	if err != nil {
		return nil, err
	}
	keys, err := cdm.DecryptLicenseKeys(challenge, license)
	if err != nil {
		return nil, fmt.Errorf("decrypt license keys: %w", err)
	}
	return selectContentKey(keys)
}

// selectContentKey returns the last CONTENT key in the license. Apple
// licenses carry exactly one; other key types (e.g. SIGNING) are skipped.
func selectContentKey(keys []wv.Key) ([]byte, error) {
	var contentKey []byte
	for _, key := range keys {
		if key.Type == wv.License_KeyContainer_CONTENT {
			contentKey = key.Value
		}
	}
	if len(contentKey) == 0 {
		return nil, errors.New("license contains no content key")
	}
	return contentKey, nil
}
