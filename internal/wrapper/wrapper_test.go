package wrapper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestClientErrNotConfigured(t *testing.T) {
	c := New("")
	if _, err := c.Status(); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Status() error = %v, want ErrNotConfigured", err)
	}
	if _, err := c.M3U8("123"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("M3U8() error = %v, want ErrNotConfigured", err)
	}
	if _, err := c.Key("123", "uri"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Key() error = %v, want ErrNotConfigured", err)
	}
	if _, err := c.Lyrics("123", "en-US", false); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Lyrics() error = %v, want ErrNotConfigured", err)
	}
	if _, err := c.Webplayback("123"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Webplayback() error = %v, want ErrNotConfigured", err)
	}
	if _, err := c.PlayReadyLicense("123", "chall", "uri"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("PlayReadyLicense() error = %v, want ErrNotConfigured", err)
	}
}

func TestStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Response[StatusData]{
			Code: 0,
			Msg:  "ok",
			Data: StatusData{Regions: []string{"us", "jp"}},
		})
	}))
	defer server.Close()

	regions, err := GetStatus(server.URL)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	want := []string{"us", "jp"}
	if !reflect.DeepEqual(regions, want) {
		t.Fatalf("GetStatus() = %v, want %v", regions, want)
	}
}

func TestM3U8(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/m3u8" || r.URL.Query().Get("adamId") != "999" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Response[M3U8Data]{
			Code: 0,
			Msg:  "ok",
			Data: M3U8Data{M3u8: "https://example.com/playlist.m3u8"},
		})
	}))
	defer server.Close()

	m3u8, err := GetM3U8(server.URL, "999")
	if err != nil {
		t.Fatalf("GetM3U8() error = %v", err)
	}
	if m3u8 != "https://example.com/playlist.m3u8" {
		t.Fatalf("GetM3U8() = %q", m3u8)
	}
}

func TestKeyPrefetch(t *testing.T) {
	// Should not hit network even if server URL is empty or invalid
	data, err := GetKeyTemplateJSON("http://invalid", "0", PrefetchKey)
	if err != nil {
		t.Fatalf("GetKeyTemplateJSON(prefetch) error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("GetKeyTemplateJSON(prefetch) returned empty bytes")
	}

	tmpl, err := GetKeyTemplate("http://invalid", "0", PrefetchKey)
	if err != nil {
		t.Fatalf("GetKeyTemplate(prefetch) error = %v", err)
	}
	if tmpl.RCX == "" {
		t.Fatal("GetKeyTemplate(prefetch) returned empty template")
	}
}

func TestKeyNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key" || r.URL.Query().Get("adamId") != "123" || r.URL.Query().Get("uri") != "skd://test" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Response[KeyTemplate]{
			Code: 0,
			Msg:  "ok",
			Data: KeyTemplate{Ctx: "ctx-val", State: "state-val"},
		})
	}))
	defer server.Close()

	tmpl, err := GetKeyTemplate(server.URL, "123", "skd://test")
	if err != nil {
		t.Fatalf("GetKeyTemplate() error = %v", err)
	}
	if tmpl.Ctx != "ctx-val" || tmpl.State != "state-val" {
		t.Fatalf("unexpected template: %+v", tmpl)
	}

	body, err := GetKeyTemplateJSON(server.URL, "123", "skd://test")
	if err != nil {
		t.Fatalf("GetKeyTemplateJSON() error = %v", err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}

func TestLyrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lyrics" || r.URL.Query().Get("adamId") != "555" || r.URL.Query().Get("language") != "zh-CN" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("syllable") != "1" {
			t.Errorf("expected syllable=1, got %q", r.URL.Query().Get("syllable"))
		}
		_ = json.NewEncoder(w).Encode(Response[LyricsData]{
			Code: 0,
			Msg:  "ok",
			Data: LyricsData{Lyrics: "[00:01.00]test lyrics"},
		})
	}))
	defer server.Close()

	lyrics, err := GetLyrics(server.URL, "555", "zh-CN", true)
	if err != nil {
		t.Fatalf("GetLyrics() error = %v", err)
	}
	if lyrics != "[00:01.00]test lyrics" {
		t.Fatalf("unexpected lyrics: %q", lyrics)
	}
}

func TestLyricsNotFound(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   map[string]any
		wantErr    error
	}{
		{
			name:       "top-level code 404",
			statusCode: http.StatusOK,
			response: map[string]any{
				"code": 404,
				"msg":  "lyrics not found",
			},
			wantErr: ErrLyricsNotFound,
		},
		{
			name:       "data code 404",
			statusCode: http.StatusOK,
			response: map[string]any{
				"code": 0,
				"msg":  "ok",
				"data": map[string]any{
					"code": 404,
					"msg":  "song not found",
				},
			},
			wantErr: ErrLyricsNotFound,
		},
		{
			name:       "data code 404 string",
			statusCode: http.StatusOK,
			response: map[string]any{
				"code": 0,
				"msg":  "ok",
				"data": map[string]any{
					"code": "404",
				},
			},
			wantErr: ErrLyricsNotFound,
		},
		{
			name:       "data code 404 with lyrics struct",
			statusCode: http.StatusOK,
			response: map[string]any{
				"code": 404,
				"data": map[string]any{
					"code":   404,
					"lyrics": "",
				},
			},
			wantErr: ErrLyricsNotFound,
		},
		{
			name:       "http status 404 with json code 404",
			statusCode: http.StatusNotFound,
			response: map[string]any{
				"code": 404,
				"msg":  "not found",
			},
			wantErr: ErrLyricsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			lyrics, err := GetLyrics(server.URL, "123", "en-US", false)
			if lyrics != "" {
				t.Errorf("expected empty lyrics, got %q", lyrics)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetLyrics() error = %v, want %v", err, tt.wantErr)
			}
			if !errors.Is(err, ErrNoLyrics) {
				t.Fatalf("expected errors.Is(err, ErrNoLyrics) to be true, got %v", err)
			}
			if err.Error() != "no lyrics available for this song" {
				t.Fatalf("expected error message 'no lyrics available for this song', got %q", err.Error())
			}
		})
	}
}

func TestLyricsOtherError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 500,
			"msg":  "missing music/dev token, run --login first",
		})
	}))
	defer server.Close()

	_, err := GetLyrics(server.URL, "123", "en-US", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrLyricsNotFound) {
		t.Fatalf("expected non-404 error, got ErrLyricsNotFound: %v", err)
	}
	if !strings.Contains(err.Error(), "code=500") {
		t.Fatalf("unexpected error format: %v", err)
	}
}

func TestWebplayback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adamID := r.URL.Query().Get("adamId")
		if adamID == "unavailable" {
			_ = json.NewEncoder(w).Encode(Response[WebplaybackData]{
				Code: 0,
				Msg:  "ok",
				Data: WebplaybackData{M3u8: ""},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(Response[WebplaybackData]{
			Code: 0,
			Msg:  "ok",
			Data: WebplaybackData{M3u8: "https://playback.m3u8"},
		})
	}))
	defer server.Close()

	m3u8, err := GetWebplayback(server.URL, "100")
	if err != nil {
		t.Fatalf("GetWebplayback() error = %v", err)
	}
	if m3u8 != "https://playback.m3u8" {
		t.Fatalf("unexpected m3u8: %q", m3u8)
	}

	_, err = GetWebplayback(server.URL, "unavailable")
	if err == nil || err.Error() != "Unavailable" {
		t.Fatalf("GetWebplayback(unavailable) error = %v, want 'Unavailable'", err)
	}
}

func TestPlayReadyLicense(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/license" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req LicenseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.DRMType != "pr" || req.AdamId != "777" || req.Challenge != "pr-chall" || req.URI != "data:pr" {
			t.Fatalf("unexpected request: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(Response[LicenseData]{
			Code: 0,
			Msg:  "ok",
			Data: LicenseData{
				AdamId:  "777",
				License: base64.StdEncoding.EncodeToString([]byte("<license-xml/>")),
			},
		})
	}))
	defer server.Close()

	lic, err := GetPlayReadyLicense(server.URL, "777", "pr-chall", "data:pr")
	if err != nil {
		t.Fatalf("GetPlayReadyLicense() error = %v", err)
	}
	if string(lic) != "<license-xml/>" {
		t.Fatalf("unexpected license: %q", lic)
	}
}

func TestWidevineHooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["adamId"] != "888" || req["uri"] != "prefix,pssh" {
			t.Fatalf("unexpected body: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(Response[LicenseData]{
			Code: 0,
			Msg:  "ok",
			Data: LicenseData{
				AdamId:  "888",
				License: base64.StdEncoding.EncodeToString([]byte("binary-lic")),
			},
		})
	}))
	defer server.Close()

	cl := resty.New()
	ctx := context.Background()
	ctx = context.WithValue(ctx, "adamId", "888")
	ctx = context.WithValue(ctx, "uriPrefix", "prefix")
	ctx = context.WithValue(ctx, "pssh", "pssh")

	resp, err := WidevineBeforeRequest(cl, ctx, server.URL+"/license", []byte("raw-challenge"))
	if err != nil {
		t.Fatalf("WidevineBeforeRequest() error = %v", err)
	}

	lic, err := WidevineAfterRequest(resp)
	if err != nil {
		t.Fatalf("WidevineAfterRequest() error = %v", err)
	}
	if string(lic) != "binary-lic" {
		t.Fatalf("WidevineAfterRequest() = %q, want binary-lic", lic)
	}
}

func TestErrorCodeReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 404,
			"msg":  "song not found",
		})
	}))
	defer server.Close()

	_, err := GetM3U8(server.URL, "000")
	if err == nil {
		t.Fatal("expected error on code != 0")
	}
	if err.Error() != "lite-server /m3u8 returned code=404 msg=song not found" {
		t.Fatalf("unexpected error format: %v", err)
	}
}
