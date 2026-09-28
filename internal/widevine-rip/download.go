package widevinerip

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/schollz/progressbar/v3"

	"amdl/internal/download"
)

const (
	streamDownloadConcurrency = 10
	streamDownloadMaxRetries  = 3
)

// EncryptedStream is everything needed to download and decrypt one
// segmented stream: its content key and its URLs (init segment first).
type EncryptedStream struct {
	Key  []byte
	URLs []string
}

// DownloadAndDecryptStream downloads every segment of stream concurrently,
// then decrypts the concatenated fragmented MP4 into outputPath.
func DownloadAndDecryptStream(ctx context.Context, stream EncryptedStream, outputPath string) error {
	if len(stream.Key) == 0 {
		return errors.New("empty decryption key")
	}
	if len(stream.URLs) == 0 {
		return errors.New("no download URLs")
	}

	tempFile, err := os.CreateTemp("", "enc_stream-*.mp4")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	bar := progressbar.DefaultBytes(-1, "Downloading...")
	err = download.DownloadSegments(ctx, download.Client, stream.URLs, tempFile, download.SegmentConfig{
		Concurrency: streamDownloadConcurrency,
		MaxRetries:  streamDownloadMaxRetries,
		Progress:    func(n int) { _ = bar.Add(n) },
	})
	if err != nil {
		return fmt.Errorf("download stream: %w", err)
	}
	if _, err := tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("rewind temp file: %w", err)
	}
	fmt.Println("\nDownloaded.")

	if err := DecryptMP4ToFile(tempFile, stream.Key, outputPath); err != nil {
		return fmt.Errorf("decrypt stream: %w", err)
	}
	fmt.Println("Decrypted.")
	return nil
}

// downloadWithProgress downloads url into memory while drawing a progress bar.
func downloadWithProgress(ctx context.Context, url string) (*bytes.Buffer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create media request: %w", err)
	}
	resp, err := download.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download media: %s", resp.Status)
	}

	var buffer bytes.Buffer
	bar := progressbar.NewOptions64(
		resp.ContentLength,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{}),
	)
	if _, err := buffer.ReadFrom(io.TeeReader(resp.Body, bar)); err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	return &buffer, nil
}
