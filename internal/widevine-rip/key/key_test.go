package key

import (
	"bytes"
	"context"
	"errors"
	"testing"

	wv "amdl/internal/widevine-rip/cdm"
)

func TestSelectContentKey(t *testing.T) {
	keys := []wv.Key{
		{Type: wv.License_KeyContainer_SIGNING, Value: []byte("signing")},
		{Type: wv.License_KeyContainer_CONTENT, Value: []byte("content")},
	}
	got, err := selectContentKey(keys)
	if err != nil || !bytes.Equal(got, []byte("content")) {
		t.Fatalf("selectContentKey() = %q, %v", got, err)
	}
	if _, err := selectContentKey(keys[:1]); err == nil {
		t.Fatal("expected license without content key to fail")
	}
}

func TestAcquireContentKeyReturnsExchangeError(t *testing.T) {
	wantErr := errors.New("license server down")
	pssh := make([]byte, wv.PSSHBoxHeaderSize)
	_, err := AcquireContentKey(context.Background(), pssh, func(context.Context, []byte) ([]byte, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("AcquireContentKey() error = %v, want %v", err, wantErr)
	}
}
