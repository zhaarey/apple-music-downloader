package classical

import "testing"

func TestParseRecordingURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		configLang string
		want       Request
		isRec      bool
		wantErr    bool
	}{
		{
			name:       "url language wins",
			raw:        "https://classical.music.apple.com/jp/recording/johann-pachelbel-1653-pp429-1452536848?l=en-US",
			configLang: "zh-CN",
			want:       Request{Storefront: "jp", Language: "en-US", RecordingID: "johann-pachelbel-1653-pp429-1452536848"},
			isRec:      true,
		},
		{
			name:  "uppercase storefront normalized",
			raw:   "https://classical.music.apple.com/US/recording/ludwig-van-beethoven-1770-pp193-1873004116",
			want:  Request{Storefront: "us", RecordingID: "ludwig-van-beethoven-1770-pp193-1873004116"},
			isRec: true,
		},
		{
			name:       "empty l falls back to config",
			raw:        "https://classical.music.apple.com/us/recording/abc-1?l=",
			configLang: "zh-CN",
			want:       Request{Storefront: "us", Language: "zh-CN", RecordingID: "abc-1"},
			isRec:      true,
		},
		{name: "ordinary album", raw: "https://music.apple.com/us/album/x/1"},
		{name: "classical work", raw: "https://classical.music.apple.com/us/work/johann-pachelbel-1653-pp429"},
		{name: "http rejected", raw: "http://classical.music.apple.com/us/recording/abc-1", isRec: true, wantErr: true},
		{name: "lookalike host", raw: "https://classical.music.apple.com.evil.com/us/recording/abc-1"},
		{name: "empty slug", raw: "https://classical.music.apple.com/us/recording/", isRec: true, wantErr: true},
		{name: "selector rejected", raw: "https://classical.music.apple.com/us/recording/abc-1?i=1", isRec: true, wantErr: true},
		{name: "duplicate l rejected", raw: "https://classical.music.apple.com/us/recording/abc-1?l=en&l=ja", isRec: true, wantErr: true},
		{name: "bad port rejected", raw: "https://classical.music.apple.com:8443/us/recording/abc-1", isRec: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isRec, err := ParseRecordingURL(tt.raw, tt.configLang)
			if isRec != tt.isRec || (err != nil) != tt.wantErr {
				t.Fatalf("isRec=%v err=%v, want isRec=%v wantErr=%v", isRec, err, tt.isRec, tt.wantErr)
			}
			if err != nil || !isRec {
				return
			}
			tt.want.PublicURL = tt.raw
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
