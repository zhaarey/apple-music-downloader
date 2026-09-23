package download

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadSegmentsWritesInOrder(t *testing.T) {
	payloads := []string{"zero", "one", "two"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("index") {
		case "0":
			_, _ = w.Write([]byte("zero"))
		case "1":
			_, _ = w.Write([]byte("one"))
		case "2":
			_, _ = w.Write([]byte("two"))
		}
	}))
	defer server.Close()

	resolved := make([]string, len(payloads))
	for i := range payloads {
		resolved[i] = fmt.Sprintf("%s/segment?index=%d", server.URL, i)
	}

	var output bytes.Buffer
	var progressCalls int32
	err := DownloadSegments(context.Background(), server.Client(), resolved, &output, SegmentConfig{
		Concurrency: 3,
		Progress: func(n int) {
			atomic.AddInt32(&progressCalls, 1)
		},
	})
	if err != nil {
		t.Fatalf("DownloadSegments() error = %v", err)
	}
	if progressCalls != 3 {
		t.Fatalf("progress calls = %d, want 3", progressCalls)
	}
	if got := output.String(); got != "zeroonetwo" {
		t.Fatalf("output = %q, want %q", got, "zeroonetwo")
	}
}

func TestWriteSegmentsInOrderRejectsDuplicate(t *testing.T) {
	results := make(chan segmentResult, 2)
	results <- segmentResult{Segment: Segment{Index: 1, Data: []byte("first")}}
	results <- segmentResult{Segment: Segment{Index: 1, Data: []byte("duplicate")}}
	close(results)

	var output bytes.Buffer
	err := writeSegmentsInOrder(results, &output, 2, nil)
	if err == nil {
		t.Fatal("writeSegmentsInOrder() error = nil, want duplicate segment error")
	}
	if output.Len() != 0 {
		t.Fatalf("output = %q, want empty", output.String())
	}
}

func TestDownloadSegmentsRetries(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	var output bytes.Buffer
	err := DownloadSegments(context.Background(), server.Client(), []string{server.URL}, &output, SegmentConfig{
		MaxRetries: 1,
		RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("DownloadSegments() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if output.String() != "ok" {
		t.Fatalf("output = %q, want %q", output.String(), "ok")
	}
}

func TestDownloadSegmentsHTTPFailure(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var output bytes.Buffer
	err := DownloadSegments(context.Background(), server.Client(), []string{server.URL}, &output, SegmentConfig{
		MaxRetries: 2,
		RetryDelay: time.Millisecond,
	})
	if err == nil {
		t.Fatal("DownloadSegments() error = nil, want HTTP failure")
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

func TestDownloadSegmentsWriteFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	err := DownloadSegments(context.Background(), server.Client(), []string{server.URL}, failingWriter{}, SegmentConfig{})
	if err == nil {
		t.Fatal("DownloadSegments() error = nil, want write failure")
	}
	if !strings.Contains(err.Error(), "write segment 0") {
		t.Fatalf("write failure context missing: %v", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("disk full")
}

func TestDownloadSegmentsBoundsLookahead(t *testing.T) {
	const total = 20
	release := make(chan struct{})
	var started int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("index") == "0" {
			<-release
		} else {
			atomic.AddInt32(&started, 1)
		}
		_, _ = w.Write([]byte("x"))
	}))
	defer server.Close()

	urls := make([]string, total)
	for i := range urls {
		urls[i] = fmt.Sprintf("%s/segment?index=%d", server.URL, i)
	}

	done := make(chan error, 1)
	var output bytes.Buffer
	go func() {
		done <- DownloadSegments(context.Background(), server.Client(), urls, &output, SegmentConfig{Concurrency: 2})
	}()

	time.Sleep(200 * time.Millisecond)
	// Window is Concurrency*2 = 4 segments, one of which is the stalled
	// segment 0, so at most 3 later segments may be fetched ahead of it.
	if got := atomic.LoadInt32(&started); got > 3 {
		t.Fatalf("fetched %d segments ahead of a stalled segment, want at most 3", got)
	}
	close(release)

	if err := <-done; err != nil {
		t.Fatalf("DownloadSegments() error = %v", err)
	}
	if output.Len() != total {
		t.Fatalf("output length = %d, want %d", output.Len(), total)
	}
}
