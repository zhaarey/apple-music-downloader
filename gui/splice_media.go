package main

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const spliceMediaPrefix = "/splice-media/"

var spliceMediaExtensions = map[string]string{
	".m4a":  "audio/mp4",
	".mp4":  "audio/mp4",
	".m4b":  "audio/mp4",
	".aac":  "audio/aac",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
}

func spliceMediaURL(path string) string {
	if path == "" {
		return ""
	}
	clean := filepath.Clean(path)
	enc := base64.RawURLEncoding.EncodeToString([]byte(clean))
	return spliceMediaPrefix + enc
}

func decodeSpliceMediaPath(urlPath string) (string, bool) {
	if !strings.HasPrefix(urlPath, spliceMediaPrefix) {
		return "", false
	}
	enc := strings.TrimPrefix(urlPath, spliceMediaPrefix)
	if enc == "" {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", false
	}
	path := filepath.Clean(string(raw))
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := spliceMediaExtensions[ext]; !ok {
		return "", false
	}
	return path, true
}

func serveSpliceMedia(w http.ResponseWriter, r *http.Request) bool {
	path, ok := decodeSpliceMediaPath(r.URL.Path)
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return true
	}
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return true
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	if ct := spliceMediaExtensions[ext]; ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return true
}

func spliceMediaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && serveSpliceMedia(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
