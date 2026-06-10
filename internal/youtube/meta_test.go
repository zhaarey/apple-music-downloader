package youtube

import (
	"testing"

	"main/utils/structs"
)

func TestResolveArtSource(t *testing.T) {
	cfg := structs.ConfigSet{}
	tests := []struct {
		name     string
		meta     DownloadMeta
		wantPath string
		wantURL  string
	}{
		{
			name: "youtube default",
			meta: DownloadMeta{ArtURL: "https://i.ytimg.com/vi/x/hqdefault.jpg"},
			wantURL: "https://i.ytimg.com/vi/x/hqdefault.jpg",
		},
		{
			name: "youtube explicit",
			meta: DownloadMeta{ArtSource: ArtSourceYouTube, ArtURL: "https://thumb.example/a.jpg"},
			wantURL: "https://thumb.example/a.jpg",
		},
		{
			name:     "custom file",
			meta:     DownloadMeta{ArtSource: ArtSourceCustom, CoverPath: `C:\covers\art.jpg`, ArtURL: "https://ignored"},
			wantPath: `C:\covers\art.jpg`,
		},
		{
			name: "none",
			meta: DownloadMeta{ArtSource: ArtSourceNone, ArtURL: "https://ignored", CoverPath: "ignored.jpg"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tags := MetaFromDownload(tc.meta, cfg)
			if tags.CoverPath != tc.wantPath {
				t.Fatalf("CoverPath = %q, want %q", tags.CoverPath, tc.wantPath)
			}
			if tags.CoverURL != tc.wantURL {
				t.Fatalf("CoverURL = %q, want %q", tags.CoverURL, tc.wantURL)
			}
		})
	}
}
