package main

import (
	"encoding/base64"
	"net/http"
	"path/filepath"
	"strings"
)

const spliceMediaPrefix = "/splice-media/"

var spliceMediaExtensions = map[string]bool{
	".m4a": true, ".mp3": true, ".wav": true, ".flac": true, ".aac": true,
}

func spliceMediaURL(path string) string {
	if path == "" {
		return ""
	}
	clean := filepath.Clean(path)
	enc := base64.RawURLEncoding.EncodeToString([]byte(clean))
	return spliceMediaPrefix + enc
}

func spliceMediaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, spliceMediaPrefix) {
			next.ServeHTTP(w, r)
			return
		}
		enc := strings.TrimPrefix(r.URL.Path, spliceMediaPrefix)
		raw, err := base64.RawURLEncoding.DecodeString(enc)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		path := filepath.Clean(string(raw))
		ext := strings.ToLower(filepath.Ext(path))
		if !spliceMediaExtensions[ext] {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	})
}
