package app

import (
	"path/filepath"
	"strings"
	"testing"

	ampapi "amdl/internal/amp-api"
	"amdl/internal/config"
	"amdl/internal/model"
)

func classicalTestTrack() *model.Track {
	track := &model.Track{ID: "1873004347"}
	track.Resp.Attributes.DiscNumber = 1
	track.Resp.Attributes.TrackNumber = 5
	track.AlbumData = ampapi.AlbumRespData{ID: "1873004116"}
	track.AlbumData.Attributes.Name = "Beethoven: Piano Sonatas"
	track.AlbumData.Attributes.ArtistName = "Christian Leotta"
	track.AlbumData.Attributes.ReleaseDate = "2014"
	track.Classical = &model.ClassicalContext{
		RecordingID:   "ludwig-van-beethoven-1770-pp193-1873004116",
		WorkTitle:     "Piano Sonata No. 16 in G Major, Op. 31/1",
		Composer:      "Ludwig van Beethoven",
		MovementTitle: "I. Allegro vivace",
		Position:      1,
		Count:         3,
	}
	return track
}

func TestClassicalFolderKeepsSlashInsideComponent(t *testing.T) {
	r := NewRunner(config.ConfigSet{LimitMax: 200})
	root := filepath.Join("root", "ALAC")
	dir, err := r.classicalFolder(root, classicalTestTrack())
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(rel, string(filepath.Separator))
	want := []string{
		"Ludwig van Beethoven",
		"Beethoven_ Piano Sonatas (2014) [Christian Leotta] [1873004116]",
		"Piano Sonata No. 16 in G Major, Op. 31_1 [ludwig-van-beethoven-1770-pp193-1873004116]",
	}
	if strings.Join(parts, "|") != strings.Join(want, "|") {
		t.Fatalf("folder components = %q, want %q", parts, want)
	}
}

func TestClassicalFileName(t *testing.T) {
	r := NewRunner(config.ConfigSet{LimitMax: 200})
	name, err := r.classicalFileName(classicalTestTrack())
	if err != nil {
		t.Fatal(err)
	}
	if name != "1-5 - I. Allegro vivace" {
		t.Fatalf("file name = %q", name)
	}
}

func TestClassicalTemplateRejectsUnknownPlaceholder(t *testing.T) {
	r := NewRunner(config.ConfigSet{LimitMax: 200, ClassicalFolderFormat: "{Composer}/{Nope}"})
	if _, err := r.classicalFolder("root", classicalTestTrack()); err == nil {
		t.Fatal("expected error for unknown placeholder")
	}
	r.Config.ClassicalFileFormat = "{MovementTitle}/x"
	if _, err := r.classicalFileName(classicalTestTrack()); err == nil {
		t.Fatal("expected error for a directory separator in the file template")
	}
}
