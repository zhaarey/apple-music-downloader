package app

import (
	"context"
	"path/filepath"
	"testing"

	"amdl/internal/config"
	"amdl/internal/model"
	widevinerip "amdl/internal/widevine-rip"
)

func TestTrackGetAlbumDataError(t *testing.T) {
	track := &model.Track{}
	err := track.GetAlbumData("token")
	if err == nil {
		t.Fatal("GetAlbumData() error = nil, want request failure")
	}
}

func TestDownloadAndDecryptStreamRejectsEmptyStream(t *testing.T) {
	err := widevinerip.DownloadAndDecryptStream(context.Background(), widevinerip.EncryptedStream{}, filepath.Join(t.TempDir(), "missing.mp4"))
	if err == nil {
		t.Fatal("DownloadAndDecryptStream() error = nil, want empty stream failure")
	}
}

func TestRipTrackCountsAACLCMissingLiteServer(t *testing.T) {
	r := NewRunner(config.ConfigSet{})
	r.Config.General.LiteServer = ""

	track := &model.Track{Type: "songs", ID: "1", SaveDir: t.TempDir()}
	r.ripTrack(track, "token", "token")
	if r.State.Counter.Error != 1 {
		t.Fatalf("r.State.Counter.Error = %d, want 1", r.State.Counter.Error)
	}
}

func TestRipTrackSkipsMVWithoutLiteServer(t *testing.T) {
	r := NewRunner(config.ConfigSet{})
	r.Config.General.LiteServer = ""

	track := &model.Track{Type: "music-videos", ID: "1", SaveDir: t.TempDir()}
	r.ripTrack(track, "token", "token")
	if r.State.Counter.Success != 1 {
		t.Fatalf("r.State.Counter.Success = %d, want 1", r.State.Counter.Success)
	}
}
