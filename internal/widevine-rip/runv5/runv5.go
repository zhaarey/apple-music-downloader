// Package runv5 is the wrapper-lite backend of the Widevine pipeline: web
// playback and license requests go through a wrapper-lite server instead of
// Apple directly, so no media user token is required.
package runv5

import (
	"context"
	"encoding/base64"

	widevinerip "amdl/internal/widevine-rip"
	"amdl/internal/widevine-rip/key"
	"amdl/internal/wrapper"
)

// LicenseExchange returns a key.ExchangeFunc that posts challenges for
// adamID to the wrapper-lite /license endpoint.
func LicenseExchange(liteServerURL string, adamID string, keyURI string) key.ExchangeFunc {
	client := wrapper.New(liteServerURL)
	return func(_ context.Context, challenge []byte) ([]byte, error) {
		return client.WidevineLicense(adamID, keyURI, base64.StdEncoding.EncodeToString(challenge))
	}
}

// FetchStream prepares an encrypted segmented stream (a music video video
// or audio track) for DownloadAndDecryptStream.
func FetchStream(ctx context.Context, adamID string, playlistURL string, liteServerURL string) (widevinerip.EncryptedStream, error) {
	if liteServerURL == "" {
		return widevinerip.EncryptedStream{}, wrapper.ErrNotConfigured
	}
	playlist, err := widevinerip.FetchMediaPlaylist(ctx, playlistURL, widevinerip.DefaultKeyFormat)
	if err != nil {
		return widevinerip.EncryptedStream{}, err
	}
	exchange := LicenseExchange(liteServerURL, adamID, playlist.KeyURI())
	return widevinerip.PrepareEncryptedStream(ctx, playlist, exchange)
}

// DownloadSong downloads a song's AAC-LC stream and writes it decrypted to
// outputPath. Errors from wrapper-lite (including "Unavailable") are
// returned unwrapped so callers can classify them.
func DownloadSong(ctx context.Context, adamID string, outputPath string, liteServerURL string) error {
	if liteServerURL == "" {
		return wrapper.ErrNotConfigured
	}
	playlistURL, err := wrapper.GetWebplayback(liteServerURL, adamID)
	if err != nil {
		return err
	}
	playlist, err := widevinerip.FetchMediaPlaylist(ctx, playlistURL, widevinerip.DefaultKeyFormat)
	if err != nil {
		return err
	}
	exchange := LicenseExchange(liteServerURL, adamID, playlist.KeyURI())
	return widevinerip.DownloadDecryptedFile(ctx, playlist, exchange, outputPath)
}
