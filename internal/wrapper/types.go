package wrapper

// Response is the standard envelope returned by wrapper-lite endpoints.
type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// StatusData represents data returned by /status.
type StatusData struct {
	Regions []string `json:"regions"`
}

// M3U8Data represents data returned by /m3u8.
type M3U8Data struct {
	M3u8 string `json:"m3u8"`
}

// KeyTemplate represents decryption parameters returned by /key.
type KeyTemplate struct {
	Ctx   string `json:"ctx"`
	State string `json:"state"`
	RCX   string `json:"rcx"`
	RAX   string `json:"rax"`
	RDX   string `json:"rdx"`
	R9    string `json:"r9"`
	RBP   string `json:"rbp"`
}

// LyricsData represents data returned by /lyrics.
type LyricsData struct {
	Code     int    `json:"code,omitempty"`
	AdamId   string `json:"adamId,omitempty"`
	Syllable string `json:"syllable,omitempty"`
	Lyrics   string `json:"lyrics"`
}

// WebplaybackData represents data returned by /webplayback.
type WebplaybackData struct {
	M3u8 string `json:"m3u8"`
}

// LicenseData represents data returned by /license.
type LicenseData struct {
	AdamId  string `json:"adamId"`
	License string `json:"license"`
	Renew   int    `json:"renew"`
}

// PlaybackLicense is the wrapper-lite /license response envelope.
type PlaybackLicense = Response[LicenseData]

// LicenseRequest is the payload sent to /license.
type LicenseRequest struct {
	AdamId    string `json:"adamId"`
	Challenge string `json:"challenge"`
	URI       string `json:"uri"`
	DRMType   string `json:"drm-type,omitempty"`
}
