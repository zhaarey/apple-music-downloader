package app

import (
	"os"
	"path/filepath"
	"testing"

	"amdl/internal/config"
	"amdl/internal/model"
)

// classicalRipTrack returns the test track ready for ripTrack, saving into a
// fresh folder.
func classicalRipTrack(t *testing.T) *model.Track {
	t.Helper()
	track := classicalTestTrack()
	track.PreType, track.PreID = "albums", track.AlbumData.ID
	track.TaskNum, track.TaskTotal = 1, 3
	track.SaveDir = t.TempDir()
	return track
}

// putFinishedFile leaves a finished, tagged file where ripTrack looks for
// the track, as an earlier run does.
func putFinishedFile(t *testing.T, r *Runner, track *model.Track) {
	t.Helper()
	track.SavePath = writeTestM4A(t)
	if err := r.writeMP4Tags(track, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(track.SavePath, filepath.Join(track.SaveDir, "1-5 - I. Allegro vivace.m4a")); err != nil {
		t.Fatal(err)
	}
}

func TestRipTrackClassicalSkipKeepsAlbumProgress(t *testing.T) {
	cases := []struct {
		name  string
		cfg   config.ConfigSet
		setup func(t *testing.T, r *Runner, track *model.Track)
	}{
		{"finished m4a", config.ConfigSet{LimitMax: 200}, putFinishedFile},
		{
			"converted file",
			config.ConfigSet{LimitMax: 200, ConvertAfterDownload: true, ConvertFormat: "flac"},
			func(t *testing.T, _ *Runner, track *model.Track) {
				if err := os.WriteFile(filepath.Join(track.SaveDir, "1-5 - I. Allegro vivace.flac"), []byte("flac"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRunner(tc.cfg)
			track := classicalRipTrack(t)
			tc.setup(t, r, track)

			r.ripTrack(track, "", "")

			if r.State.Counter.Success != 1 || r.State.Counter.Error != 0 {
				t.Fatalf("success/error = %d/%d, want 1/0", r.State.Counter.Success, r.State.Counter.Error)
			}
			// ripAlbum reads OKDict[albumID] as album track numbers, so
			// Recording positions there would skip unrelated album tracks.
			if got := r.State.OKDict[track.AlbumData.ID]; len(got) != 0 {
				t.Fatalf("OKDict[album] = %v, want no entries", got)
			}
		})
	}
}
