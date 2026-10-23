package widevinerip

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadStreamClearWritesSegmentsUnchanged(t *testing.T) {
	segments := map[string][]byte{
		"/init.mp4": []byte("init"),
		"/seg0.m4s": []byte("segment-0"),
		"/seg1.m4s": []byte("segment-1"),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := segments[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "clear.mp4")
	stream := Stream{URLs: []string{server.URL + "/init.mp4", server.URL + "/seg0.m4s", server.URL + "/seg1.m4s"}}
	if err := DownloadStream(context.Background(), stream, outputPath); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("initsegment-0segment-1"); !bytes.Equal(got, want) {
		t.Fatalf("output = %q, want %q", got, want)
	}
	entries, err := os.ReadDir(filepath.Dir(outputPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("output directory has %d entries, want only the output file", len(entries))
	}
}

func TestPrepareStreamClearSkipsLicense(t *testing.T) {
	playlist := &MediaPlaylist{InitSegmentURL: "https://example.com/init.mp4", SegmentURLs: []string{"https://example.com/seg0.m4s"}}
	stream, err := PrepareStream(context.Background(), playlist, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stream.Key) != 0 || len(stream.URLs) != 2 {
		t.Fatalf("stream = %+v, want keyless stream with 2 URLs", stream)
	}
}
