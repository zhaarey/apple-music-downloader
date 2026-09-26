package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	ampapi "amdl/internal/amp-api"
	"amdl/internal/classical"
	"amdl/internal/model"
)

// catalogAlbum loads a captured Catalog album response.
func catalogAlbum(t *testing.T, file, storefront, id, language string) *model.Album {
	t.Helper()
	data, err := os.ReadFile("../classical/testdata/catalog/" + file)
	if err != nil {
		t.Fatal(err)
	}
	var resp ampapi.AlbumResp
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}
	album := model.NewAlbum(storefront, id)
	album.Language = language
	if err := album.SetResp(resp); err != nil {
		t.Fatal(err)
	}
	return album
}

func beethovenAlbum(t *testing.T) *model.Album {
	return catalogAlbum(t, "us_beethoven_en-US.json", "us", "1873004116", "en-US")
}

func beethovenRecording(ids ...string) *classical.Recording {
	rec := &classical.Recording{
		Request:   classical.Request{Storefront: "us", Language: "en-US", RecordingID: "ludwig-van-beethoven-1770-pp193-1873004116"},
		AlbumID:   "1873004116",
		WorkTitle: "Piano Sonata No. 16 in G Major, Op. 31/1",
		Composer:  "Ludwig van Beethoven",
	}
	for i, id := range ids {
		rec.Tracks = append(rec.Tracks, classical.Track{
			SongID:     id,
			Title:      "movement " + id,
			Position:   i + 1,
			Count:      len(ids),
			Conductors: []string{"A", "B", "A"},
		})
	}
	return rec
}

func TestBuildRecordingTracksSelectsInRecordingOrder(t *testing.T) {
	album := beethovenAlbum(t)
	// The album's last track shows songs are picked by ID, not by position.
	last := album.Tracks[len(album.Tracks)-1]
	rec := beethovenRecording("1873004590", "1873004347", last.ID)

	tracks, err := buildRecordingTracks(rec, album)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 3 {
		t.Fatalf("got %d tracks, want 3", len(tracks))
	}
	for i, track := range tracks {
		want := rec.Tracks[i]
		catalog := album.Tracks[indexOfTrack(album, want.SongID)]
		if track.ID != want.SongID || track.TaskNum != i+1 || track.TaskTotal != 3 {
			t.Fatalf("track %d = %s task %d/%d", i, track.ID, track.TaskNum, track.TaskTotal)
		}
		if track.Resp.Attributes.TrackNumber != catalog.Resp.Attributes.TrackNumber || track.DiscTotal != catalog.DiscTotal {
			t.Fatalf("track %s lost parent numbering", track.ID)
		}
		c := track.Classical
		if c == nil || c.Position != i+1 || c.Count != 3 || c.MovementTitle != want.Title || c.WorkTitle != rec.WorkTitle {
			t.Fatalf("classical context = %#v", c)
		}
		if c.Conductor != "A; B" || c.RecordingID != rec.Request.RecordingID {
			t.Fatalf("classical context = %#v", c)
		}
	}
	if album.Tracks[0].Classical != nil {
		t.Fatal("parent album tracks must not be modified")
	}
}

func TestBuildRecordingTracksNamesEveryConductorOfTheRecording(t *testing.T) {
	rec := beethovenRecording("1873004347", "1873004590")
	// One performance, but only some tracks carry the conductor credit.
	rec.Tracks[0].Conductors = nil
	rec.Tracks[1].Conductors = []string{"B", "A"}

	tracks, err := buildRecordingTracks(rec, beethovenAlbum(t))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"", "B; A"} {
		if c := tracks[i].Classical; c.Conductor != want || c.RecordingConductor != "B; A" {
			t.Fatalf("track %d: conductor %q, recording conductor %q; want %q, %q", i, c.Conductor, c.RecordingConductor, want, "B; A")
		}
	}
}

func TestBuildRecordingTracksReportsMissingSongs(t *testing.T) {
	_, err := buildRecordingTracks(beethovenRecording("1873004347", "111", "222"), beethovenAlbum(t))
	if err == nil || !strings.Contains(err.Error(), "111") || !strings.Contains(err.Error(), "222") {
		t.Fatalf("err = %v, want both missing ids", err)
	}
}

func TestBuildRecordingTracksRejectsOtherAlbum(t *testing.T) {
	rec := beethovenRecording("1873004347")
	rec.AlbumID = "1"
	if _, err := buildRecordingTracks(rec, beethovenAlbum(t)); err == nil {
		t.Fatal("expected album mismatch error")
	}
}

func TestAlbumSetRespRejectsEmptyResponse(t *testing.T) {
	if err := model.NewAlbum("us", "1").SetResp(ampapi.AlbumResp{}); err == nil {
		t.Fatal("expected error for empty album response")
	}
}

func indexOfTrack(album *model.Album, id string) int {
	for i, track := range album.Tracks {
		if track.ID == id {
			return i
		}
	}
	return -1
}
