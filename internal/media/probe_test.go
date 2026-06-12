package media

import "testing"

func TestValidateAppleMusicStreams(t *testing.T) {
	tests := []struct {
		name    string
		streams []ffprobeStream
		wantErr bool
	}{
		{
			name: "valid h264 aac stereo",
			streams: []ffprobeStream{
				{CodecType: "video", CodecName: "h264"},
				{CodecType: "audio", CodecName: "aac", Channels: 2},
			},
		},
		{
			name: "valid avc1 aac mono",
			streams: []ffprobeStream{
				{CodecType: "video", CodecName: "avc1"},
				{CodecType: "audio", CodecName: "aac", Channels: 1},
			},
		},
		{
			name: "invalid video only",
			streams: []ffprobeStream{
				{CodecType: "video", CodecName: "h264"},
			},
			wantErr: true,
		},
		{
			name: "invalid h264 opus",
			streams: []ffprobeStream{
				{CodecType: "video", CodecName: "h264"},
				{CodecType: "audio", CodecName: "opus", Channels: 2},
			},
			wantErr: true,
		},
		{
			name: "invalid vp9 aac",
			streams: []ffprobeStream{
				{CodecType: "video", CodecName: "vp9"},
				{CodecType: "audio", CodecName: "aac", Channels: 2},
			},
			wantErr: true,
		},
		{
			name: "invalid aac surround",
			streams: []ffprobeStream{
				{CodecType: "video", CodecName: "h264"},
				{CodecType: "audio", CodecName: "aac", Channels: 6},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateAppleMusicStreams(tt.streams)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
