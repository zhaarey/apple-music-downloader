package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Segment struct {
	Index int
	Data  []byte
}

type SegmentConfig struct {
	Concurrency int
	MaxRetries  int
	RetryDelay  time.Duration
	Progress    func(n int)
}

const DefaultSegmentConcurrency = 5

type segmentJob struct {
	Index int
	URL   string
}

type segmentResult struct {
	Segment
	err error
}

func DownloadSegments(ctx context.Context, client *http.Client, urls []string, output io.Writer, config SegmentConfig) error {
	if len(urls) == 0 {
		return nil
	}
	if config.Concurrency <= 0 {
		config.Concurrency = DefaultSegmentConcurrency
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = time.Second
	}
	if client == nil {
		client = Client
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerCount := config.Concurrency
	if workerCount > len(urls) {
		workerCount = len(urls)
	}
	results := make(chan segmentResult, workerCount*2)
	errCh := make(chan error, workerCount+1)

	writerDone := make(chan error, 1)
	go func() {
		writerDone <- writeSegmentsInOrder(results, output, len(urls), config.Progress)
	}()

	var workerWg sync.WaitGroup
	var nextJob int64
	for i := 0; i < workerCount; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for {
				index := int(atomic.AddInt64(&nextJob, 1)) - 1
				if index >= len(urls) {
					return
				}

				select {
				case results <- downloadSegment(ctx, client, segmentJob{Index: index, URL: urls[index]}, config):
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		workerWg.Wait()
		close(results)
	}()

	select {
	case err := <-errCh:
		cancel()
		<-writerDone
		return err
	case err := <-writerDone:
		return err
	}
}

func fetchSegment(ctx context.Context, client *http.Client, job segmentJob, config SegmentConfig) (Segment, error) {
	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := config.RetryDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return Segment{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.URL, nil)
		if err != nil {
			return Segment{}, fmt.Errorf("segment %d: create request: %w", job.Index, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			closeErr := resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			if closeErr != nil {
				lastErr = fmt.Errorf("%s (close response: %v)", lastErr, closeErr)
			}
			continue
		}

		data, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		switch {
		case readErr != nil:
			lastErr = fmt.Errorf("read response: %w", readErr)
		case closeErr != nil:
			lastErr = fmt.Errorf("close response: %w", closeErr)
		default:
			return Segment{Index: job.Index, Data: data}, nil
		}
	}

	return Segment{}, fmt.Errorf("segment %d: download failed after %d retries: %w", job.Index, config.MaxRetries, lastErr)
}

func downloadSegment(ctx context.Context, client *http.Client, job segmentJob, config SegmentConfig) segmentResult {
	segment, err := fetchSegment(ctx, client, job, config)
	return segmentResult{Segment: segment, err: err}
}

func writeSegmentsInOrder(results <-chan segmentResult, output io.Writer, totalSegments int, progress func(n int)) error {
	buffer := make(map[int][]byte)
	nextIndex := 0

	write := func(data []byte, index int) error {
		if _, err := output.Write(data); err != nil {
			return fmt.Errorf("write segment %d: %w", index, err)
		}
		if progress != nil {
			progress(len(data))
		}
		return nil
	}

	for result := range results {
		if result.err != nil {
			return result.err
		}
		segment := result.Segment
		if segment.Index < nextIndex {
			continue
		}
		if _, duplicate := buffer[segment.Index]; duplicate && segment.Index > nextIndex {
			return fmt.Errorf("duplicate segment %d", segment.Index)
		}
		if segment.Index == nextIndex {
			if err := write(segment.Data, segment.Index); err != nil {
				return err
			}
			nextIndex++
			for {
				data, ok := buffer[nextIndex]
				if !ok {
					break
				}
				if err := write(data, nextIndex); err != nil {
					return err
				}
				delete(buffer, nextIndex)
				nextIndex++
			}
			continue
		}
		buffer[segment.Index] = segment.Data
	}

	if nextIndex != totalSegments {
		return fmt.Errorf("incomplete segment download: expected %d segments, wrote %d", totalSegments, nextIndex)
	}
	return nil
}
