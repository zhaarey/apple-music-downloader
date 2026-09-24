package app

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"amdl/internal/config"

	mp4tag "github.com/itouakirai/go-mp4tag"
)

func testBox(boxType string, payload ...[]byte) []byte {
	body := bytes.Join(payload, nil)
	box := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(box[0:4], uint32(8+len(body)))
	copy(box[4:8], boxType)
	return append(box, body...)
}

// writeTestM4A creates a minimal file that go-mp4tag can tag: ftyp, a moov
// with one track and an empty ilst, and a small mdat.
func writeTestM4A(t *testing.T) string {
	t.Helper()
	stco := testBox("stco", make([]byte, 4), []byte{0, 0, 0, 1}, make([]byte, 4))
	trak := testBox("trak", testBox("mdia", testBox("minf", testBox("stbl", stco))))
	udta := testBox("udta", testBox("meta", make([]byte, 4), testBox("ilst")))
	data := append(testBox("ftyp", []byte("M4A "), make([]byte, 4)), testBox("moov", trak, udta)...)
	stcoAt := bytes.Index(data, []byte("stco")) - 4
	binary.BigEndian.PutUint32(data[stcoAt+16:], uint32(len(data)+8))
	data = append(data, testBox("mdat", []byte("AUDIO"))...)
	path := filepath.Join(t.TempDir(), "track.m4a")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWriteMP4TagsClassical(t *testing.T) {
	r := NewRunner(config.ConfigSet{})
	track := classicalTestTrack()
	track.PreType = "albums"
	track.AlbumData.Attributes.TrackCount = 23
	track.Resp.Attributes.Name = "Piano Sonata No. 16 in G Major, Op. 31/1: I. Allegro vivace"
	track.Classical.Conductor = "A; B"
	track.SavePath = writeTestM4A(t)

	if err := r.writeMP4Tags(track, ""); err != nil {
		t.Fatal(err)
	}
	m, err := mp4tag.Open(track.SavePath)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	got, err := m.Read()
	if err != nil {
		t.Fatal(err)
	}
	c := track.Classical
	if got.Title != c.MovementTitle || got.Movement != c.MovementTitle || got.Work != c.WorkTitle {
		t.Fatalf("title/movement/work = %q / %q / %q", got.Title, got.Movement, got.Work)
	}
	if got.MovementIndex != 1 || got.MovementCount != 3 || !got.ShowWorkAndMovement {
		t.Fatalf("movement numbers = %d/%d show=%v", got.MovementIndex, got.MovementCount, got.ShowWorkAndMovement)
	}
	if got.Composer != c.Composer || got.Conductor != "A; B" {
		t.Fatalf("composer/conductor = %q / %q", got.Composer, got.Conductor)
	}
	if got.Album != "Beethoven: Piano Sonatas" || got.TrackNumber != 5 || got.TrackTotal != 23 {
		t.Fatalf("parent album fields = %q %d/%d", got.Album, got.TrackNumber, got.TrackTotal)
	}
	if got.Custom[classicalIdentityTag] != classicalIdentity(track) {
		t.Fatalf("identity tag = %q", got.Custom[classicalIdentityTag])
	}
}

func TestClassicalExistingFile(t *testing.T) {
	r := NewRunner(config.ConfigSet{})
	track := classicalTestTrack()
	track.PreType = "albums"
	track.SavePath = writeTestM4A(t)
	if err := r.writeMP4Tags(track, ""); err != nil {
		t.Fatal(err)
	}
	if err := checkClassicalExisting(track.SavePath, track); err != nil {
		t.Fatalf("same recording and song should be reusable: %v", err)
	}

	other := classicalTestTrack()
	other.ID = "1873004590"
	if err := checkClassicalExisting(track.SavePath, other); err == nil {
		t.Fatal("a file for another song must be reported as a conflict")
	}

	untagged := writeTestM4A(t)
	if err := checkClassicalExisting(untagged, track); err == nil {
		t.Fatal("a file without the identity tag must be reported as a conflict")
	}
}
