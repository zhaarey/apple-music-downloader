// Package widevinerip downloads and decrypts Widevine-protected Apple Music
// streams.
//
// The pipeline is backend-agnostic:
//
//	MediaPlaylist ──BuildPSSH──▶ key.AcquireContentKey(ExchangeFunc) ──▶ content key
//	                                                                     │
//	               DownloadDecryptedFile (songs) ◀──────────────────────┤
//	               Stream → DownloadStream (MV, station)
//
// A license backend only has to provide a key.ExchangeFunc. This package
// ships the Apple web playback backend (AppleLicenseExchange); the runv5
// subpackage provides the wrapper-lite backend.
package widevinerip

import (
	"context"
	"errors"
	"fmt"

	"amdl/internal/widevine-rip/key"
)

// AcquireContentKey requests a license for playlist's key ID through
// exchange and returns the content key.
func AcquireContentKey(ctx context.Context, playlist *MediaPlaylist, exchange key.ExchangeFunc) ([]byte, error) {
	if !playlist.Encrypted() {
		return nil, errors.New("playlist is not encrypted")
	}
	pssh, err := BuildPSSH(playlist.KeyPayload, "")
	if err != nil {
		return nil, err
	}
	contentKey, err := key.AcquireContentKey(ctx, pssh, exchange)
	if err != nil {
		return nil, fmt.Errorf("acquire content key: %w", err)
	}
	return contentKey, nil
}

// PrepareStream pairs a segmented playlist's URLs with its content key,
// ready for DownloadStream. A clear playlist needs no license, so exchange
// is not called and the stream carries no key.
func PrepareStream(ctx context.Context, playlist *MediaPlaylist, exchange key.ExchangeFunc) (Stream, error) {
	if !playlist.Encrypted() {
		return Stream{URLs: playlist.StreamURLs()}, nil
	}
	contentKey, err := AcquireContentKey(ctx, playlist, exchange)
	if err != nil {
		return Stream{}, err
	}
	return Stream{Key: contentKey, URLs: playlist.StreamURLs()}, nil
}

// DownloadDecryptedFile acquires the content key of a single-file playlist
// (a song), downloads the file and writes it decrypted to outputPath.
func DownloadDecryptedFile(ctx context.Context, playlist *MediaPlaylist, exchange key.ExchangeFunc, outputPath string) error {
	contentKey, err := AcquireContentKey(ctx, playlist, exchange)
	if err != nil {
		return err
	}
	encrypted, err := downloadWithProgress(ctx, playlist.InitSegmentURL)
	if err != nil {
		return err
	}
	fmt.Println("Downloaded")
	if err := DecryptMP4ToFile(encrypted, contentKey, outputPath); err != nil {
		fmt.Println("Decryption failed")
		return err
	}
	fmt.Println("Decrypted")
	return nil
}

// FetchStationStream prepares a station stream using Apple's license
// endpoint. licenseURL may be empty to use AppleLicenseURL.
func FetchStationStream(ctx context.Context, adamID string, playlistURL string, creds Credentials, licenseURL string) (Stream, error) {
	playlist, err := FetchMediaPlaylist(ctx, playlistURL, DefaultKeyFormat)
	if err != nil {
		return Stream{}, err
	}
	exchange := AppleLicenseExchange(creds, licenseURL, adamID, playlist.KeyURI())
	return PrepareStream(ctx, playlist, exchange)
}

// DownloadSong downloads a 256 kbps AAC song through Apple's web playback
// and license endpoints and writes it decrypted to outputPath. licenseURL
// may be empty to use AppleLicenseURL.
func DownloadSong(ctx context.Context, adamID string, outputPath string, creds Credentials, licenseURL string) error {
	playlistURL, err := FetchSongPlaylistURL(ctx, adamID, creds)
	if err != nil {
		return err
	}
	playlist, err := FetchMediaPlaylist(ctx, playlistURL, DefaultKeyFormat)
	if err != nil {
		return err
	}
	exchange := AppleLicenseExchange(creds, licenseURL, adamID, playlist.KeyURI())
	return DownloadDecryptedFile(ctx, playlist, exchange, outputPath)
}
