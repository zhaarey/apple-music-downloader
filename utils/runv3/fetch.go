package runv3

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func fetchBytes(client *http.Client, rawURL string, offset, limit int64) ([]byte, error) {
	if client == nil {
		client = httpClient()
	}
	var lastErr error
	for attempt := 1; attempt <= maxDownloadTry; attempt++ {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range cdnHeaders {
			req.Header.Set(k, v)
		}
		if limit > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+limit-1))
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if limit > 0 {
			if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("HTTP %d for range request", resp.StatusCode)
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			if int64(len(data)) != limit {
				lastErr = fmt.Errorf("short range read: got %d bytes, expected %d", len(data), limit)
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return data, nil
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		if resp.ContentLength > 0 && int64(len(data)) != resp.ContentLength {
			lastErr = fmt.Errorf("short read: got %d bytes, expected %d", len(data), resp.ContentLength)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func fetchPart(rawURL string) ([]byte, error) {
	return fetchBytes(nil, rawURL, 0, 0)
}
