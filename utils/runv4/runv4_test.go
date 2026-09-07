package runv4

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/schollz/progressbar/v3"
)

// flakyFileServer serves payload but drops the connection after cutAt bytes on
// the first full request. It also tracks how many requests are in flight at once.
type flakyFileServer struct {
	payload []byte
	cutAt   int

	mu           sync.Mutex
	requests     []string
	inflight     int32
	maxInflight  int32
	fullRequests int
}

func (s *flakyFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cur := atomic.AddInt32(&s.inflight, 1)
	defer atomic.AddInt32(&s.inflight, -1)
	s.mu.Lock()
	if cur > s.maxInflight {
		s.maxInflight = cur
	}
	s.requests = append(s.requests, r.Header.Get("Range"))
	s.mu.Unlock()

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		s.mu.Lock()
		s.fullRequests++
		first := s.fullRequests == 1
		s.mu.Unlock()

		w.Header().Set("Content-Length", strconv.Itoa(len(s.payload)))
		w.WriteHeader(http.StatusOK)
		if first {
			w.Write(s.payload[:s.cutAt])
			w.(http.Flusher).Flush()
			// Simulate the network dying mid-body.
			hj, ok := w.(http.Hijacker)
			if !ok {
				panic("server does not support hijacking")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				panic(err)
			}
			conn.Close()
			return
		}
		w.Write(s.payload)
		return
	}

	start, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(rangeHeader, "bytes="), "-"))
	if err != nil {
		http.Error(w, "bad range", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(s.payload)-1, len(s.payload)))
	w.Header().Set("Content-Length", strconv.Itoa(len(s.payload)-start))
	w.WriteHeader(http.StatusPartialContent)
	w.Write(s.payload[start:])
}

func silentBar(n int64) *progressbar.ProgressBar {
	return progressbar.NewOptions64(n, progressbar.OptionSetWriter(io.Discard))
}

func TestDownloadWithResumeConsumesFirstResponseAndResumes(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789abcdef"), 4096) // 64 KiB
	srv := &flakyFileServer{payload: payload, cutAt: 20000}
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := ts.Client()
	header := make(http.Header)

	ctx := context.Background()
	firstCtx, firstCancel := context.WithCancel(ctx)
	defer firstCancel()
	req, err := http.NewRequestWithContext(firstCtx, "GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header = header
	first, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Body.Close()

	buf, err := downloadWithResume(ctx, client, ts.URL, header, first, firstCancel, silentBar(first.ContentLength))
	if err != nil {
		t.Fatalf("downloadWithResume: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), payload) {
		t.Fatalf("payload mismatch: got %d bytes, want %d", buf.Len(), len(payload))
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.maxInflight != 1 {
		t.Fatalf("expected at most one request in flight, saw %d", srv.maxInflight)
	}
	if srv.fullRequests != 1 {
		t.Fatalf("the already-open first response must be consumed instead of re-requested; saw %d full requests", srv.fullRequests)
	}
	if len(srv.requests) != 2 || srv.requests[1] != "bytes=20000-" {
		t.Fatalf("expected one full request followed by a resume from byte 20000, got %q", srv.requests)
	}
	if len(header) != 0 {
		t.Fatalf("caller's header map must not be mutated, got %v", header)
	}
}

func TestDownloadWithResumeRejectsBadFirstStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer ts.Close()

	client := ts.Client()
	firstCtx, firstCancel := context.WithCancel(context.Background())
	defer firstCancel()
	req, _ := http.NewRequestWithContext(firstCtx, "GET", ts.URL, nil)
	first, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Body.Close()

	_, err = downloadWithResume(context.Background(), client, ts.URL, make(http.Header), first, firstCancel, silentBar(first.ContentLength))
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected a 403 error, got %v", err)
	}
}
