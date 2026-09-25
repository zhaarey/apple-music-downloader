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

func TestClassicalFolderRejectsPerTrackPlaceholders(t *testing.T) {
	// The folder is shared by the whole Recording, so a value that differs
	// between its tracks would put every track in the first track's folder.
	for _, p := range []string{"{MovementTitle}", "{SongId}", "{DiscNumber}", "{TrackNumber}", "{MovementIndex}"} {
		r := NewRunner(config.ConfigSet{LimitMax: 200, ClassicalFolderFormat: "{Composer}/Part " + p})
		if dir, err := r.classicalFolder("root", classicalTestTrack()); err == nil {
			t.Errorf("%s in the folder template: got folder %q, want an error", p, dir)
		}
	}
}

func TestClassicalFolderNamesAreSafeOnDisk(t *testing.T) {
	cases := []struct {
		title string
		want  string // "" means an error is expected
	}{
		{"Sonata. . .", "Sonata"},
		{"...", ""}, // would become ".." and climb out of the Composer folder
		{"Op. 1\t2", "Op. 1_2"},
		{"CON", "_CON"},
		{"nul.txt", "_nul.txt"},
		{"Com1", "_Com1"},
		{"Console", "Console"},
		{"COM10", "COM10"},
		{strings.Repeat("a", 255), strings.Repeat("a", 255)},
		{strings.Repeat("a", 256), ""},
	}
	for _, tc := range cases {
		r := NewRunner(config.ConfigSet{LimitMax: 300, ClassicalFolderFormat: "{Composer}/{WorkTitle}"})
		track := classicalTestTrack()
		track.Classical.WorkTitle = tc.title
		dir, err := r.classicalFolder("root", track)
		if tc.want == "" {
			if err == nil {
				t.Errorf("work title %q: got folder %q, want an error", tc.title, dir)
			}
			continue
		}
		if err != nil || dir != filepath.Join("root", "Ludwig van Beethoven", tc.want) {
			t.Errorf("work title %q: got %q, %v; want last folder %q", tc.title, dir, err, tc.want)
		}
	}
}

func TestClassicalFileNameLeavesRoomForSuffixes(t *testing.T) {
	// Downloads first write "<name>.m4a.part"; converters and lyrics use
	// other extensions, so the name must stay 20 units below 255.
	r := NewRunner(config.ConfigSet{LimitMax: 300, ClassicalFileFormat: "{MovementTitle}"})
	track := classicalTestTrack()
	track.Classical.MovementTitle = strings.Repeat("a", 235)
	if _, err := r.classicalFileName(track); err != nil {
		t.Fatalf("235 characters: %v", err)
	}
	track.Classical.MovementTitle = strings.Repeat("a", 236)
	if name, err := r.classicalFileName(track); err == nil {
		t.Fatalf("236 characters: got %d-character name, want an error", len(name))
	}
}
