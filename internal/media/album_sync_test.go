package media

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zhaarey/go-mp4tag"
)

func createSilentM4A(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=stereo", "-t", "0.05", "-c:a", "aac", path)
	if err := cmd.Run(); err != nil {
		t.Skip("ffmpeg required for album sync tests")
	}
}

func writeTestJPEG(t *testing.T, path string, size int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	out, err := NormalizeCoverForApple(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
}

func tagM4A(t *testing.T, path, title, artist, album, albumArtist string) {
	t.Helper()
	mp4, err := mp4tag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer mp4.Close()
	tags := &mp4tag.MP4Tags{
		Title:       title,
		Artist:      artist,
		Album:       album,
		AlbumArtist: albumArtist,
		TrackNumber: 1,
		TrackTotal:  1,
		DiscNumber:  1,
		DiscTotal:   1,
	}
	if err := mp4.Write(tags, nil); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAlbumSync_sidecarOnlyFails(t *testing.T) {
	dir := t.TempDir()
	track := filepath.Join(dir, "01 Song.m4a")
	createSilentM4A(t, track)
	tagM4A(t, track, "Song", "Artist", "Album", "Artist")
	writeTestJPEG(t, filepath.Join(dir, "cover.jpg"), 64)

	res, err := ValidateAlbumSync("", dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ready {
		t.Fatal("expected album not ready when sidecar exists but track has no embed")
	}
	found := false
	for _, f := range res.Files {
		for _, c := range f.Checks {
			if c.ID == "sidecar_only" && !c.Pass {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected sidecar_only fail check")
	}
}

func TestPrepareAlbumForSync_embedsSidecar(t *testing.T) {
	dir := t.TempDir()
	track := filepath.Join(dir, "01 Song.m4a")
	createSilentM4A(t, track)
	tagM4A(t, track, "Song", "Artist", "Album", "Artist")
	writeTestJPEG(t, filepath.Join(dir, "cover.jpg"), 128)

	prep, err := PrepareAlbumForSync("", dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if prep.Prepared != 1 || len(prep.Errors) > 0 {
		t.Fatalf("prepare: %+v", prep)
	}
	hash, err := EmbeddedCoverHash(track)
	if err != nil || hash == "" {
		t.Fatalf("expected embedded cover after prepare: %v", err)
	}
	res, err := ValidateAlbumSync("", dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ready {
		t.Fatalf("expected ready after prepare: %s", res.Summary)
	}
}

func TestPrepareAlbumForSync_preservesMetadata(t *testing.T) {
	dir := t.TempDir()
	track := filepath.Join(dir, "01 Song.m4a")
	createSilentM4A(t, track)
	mp4, err := mp4tag.Open(track)
	if err != nil {
		t.Fatal(err)
	}
	tags := &mp4tag.MP4Tags{
		Title:       "Keep Title",
		Artist:      "Keep Artist",
		Album:       "Keep Album",
		AlbumArtist: "Keep Album Artist",
		Composer:    "Keep Composer",
		TrackNumber: 3,
		TrackTotal:  10,
	}
	if err := mp4.Write(tags, nil); err != nil {
		t.Fatal(err)
	}
	mp4.Close()
	writeTestJPEG(t, filepath.Join(dir, "cover.jpg"), 96)

	prep, err := PrepareAlbumForSync("", dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if prep.Prepared != 1 {
		t.Fatalf("expected 1 prepared, got %+v", prep)
	}

	after, err := ReadAudioTags(track)
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != "Keep Title" || after.Artist != "Keep Artist" || after.Album != "Keep Album" {
		t.Fatalf("metadata changed: %+v", after)
	}
	if after.TrackNumber != 3 || after.TrackTotal != 10 {
		t.Fatalf("track numbers changed: %+v", after)
	}

	mp4, err = mp4tag.Open(track)
	if err != nil {
		t.Fatal(err)
	}
	defer mp4.Close()
	readBack, err := mp4.Read()
	if err != nil {
		t.Fatal(err)
	}
	if readBack.Composer != "Keep Composer" {
		t.Fatalf("composer stripped: %q", readBack.Composer)
	}
}

func TestCoverHash_stable(t *testing.T) {
	data := []byte("same-bytes")
	if CoverHash(data) != CoverHash(data) {
		t.Fatal("expected stable hash")
	}
	if CoverHash(data) == CoverHash([]byte("other")) {
		t.Fatal("expected different hash for different bytes")
	}
}
