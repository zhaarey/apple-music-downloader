package fmp4unfrag_test

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	defrag "amdl/internal/media/defrag"
	mvmedia "amdl/internal/media/mv"

	"github.com/itouakirai/go-mp4tag"
)

func concat(t *testing.T, parts ...string) string {
	t.Helper()
	var data []byte
	for _, part := range parts {
		chunk, err := os.ReadFile(part)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, chunk...)
	}
	path := filepath.Join(t.TempDir(), "stream.mp4")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func muxTestMV(t *testing.T) string {
	t.Helper()
	testdata := filepath.Join("..", "mv", "testdata")
	video := concat(t, filepath.Join(testdata, "V300", "init.mp4"), filepath.Join(testdata, "V300", "1.m4s"))
	audio := concat(t, filepath.Join(testdata, "A48", "init.mp4"), filepath.Join(testdata, "A48", "1.m4s"))
	out := filepath.Join(t.TempDir(), "mv.mp4")
	if err := mvmedia.Mux(video, audio, out); err != nil {
		t.Fatal(err)
	}
	return out
}

func topLevelBoxes(t *testing.T, path string) []string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	types, _ := readTopLevel(t, file)
	return types
}

func trackDigests(t *testing.T, path string) [][sha256.Size]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	_, moov := readTopLevel(t, file)
	var digests [][sha256.Size]byte
	for _, trak := range moov.Traks {
		ranges, err := trak.GetRangesForSampleInterval(1, trak.GetNrSamples())
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		for _, r := range ranges {
			if _, err := file.Seek(int64(r.Offset), io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err := io.CopyN(hash, file, int64(r.Size)); err != nil {
				t.Fatal(err)
			}
		}
		var digest [sha256.Size]byte
		copy(digest[:], hash.Sum(nil))
		digests = append(digests, digest)
	}
	return digests
}

func TestSplitMdatKeepsSamplesAndStaysTaggable(t *testing.T) {
	baseline := trackDigests(t, muxTestMV(t))

	old := *defrag.MaxMdatPayload
	*defrag.MaxMdatPayload = 30000 // below video+audio, above each chunk
	defer func() { *defrag.MaxMdatPayload = old }()

	out := muxTestMV(t)

	mdats := 0
	for _, typ := range topLevelBoxes(t, out) {
		if typ == "mdat" {
			mdats++
		}
	}
	if mdats != 2 {
		t.Fatalf("mdat boxes = %d, want 2", mdats)
	}

	split := trackDigests(t, out)
	if len(split) != len(baseline) {
		t.Fatalf("tracks = %d, want %d", len(split), len(baseline))
	}
	for i := range baseline {
		if split[i] != baseline[i] {
			t.Fatalf("track %d samples differ after mdat split", i)
		}
	}

	tagFile, err := mp4tag.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	err = tagFile.Write(&mp4tag.MP4Tags{
		Title: "Tagged MV",
		Pictures: []*mp4tag.MP4Picture{{
			Format: mp4tag.ImageTypeAuto,
			Data:   []byte{0x89, 'P', 'N', 'G', 0, 0, 0, 0},
		}},
	}, nil)
	_ = tagFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	tagged := trackDigests(t, out)
	for i := range baseline {
		if tagged[i] != baseline[i] {
			t.Fatalf("track %d samples moved incorrectly after tag write", i)
		}
	}
}

func TestLargeOffsetsUseCo64AndStayTaggable(t *testing.T) {
	baseline := trackDigests(t, muxTestMV(t))

	old := *defrag.StcoOffsetLimit
	*defrag.StcoOffsetLimit = 0 // every output needs 64-bit offsets
	defer func() { *defrag.StcoOffsetLimit = old }()

	out := muxTestMV(t)

	file, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	_, moov := readTopLevel(t, file)
	_ = file.Close()
	for _, trak := range moov.Traks {
		stbl := trak.Mdia.Minf.Stbl
		if stbl.Stco != nil || stbl.Co64 == nil {
			t.Fatalf("track %d: want co64 instead of stco", trak.Tkhd.TrackID)
		}
	}

	check := func(stage string) {
		t.Helper()
		got := trackDigests(t, out)
		for i := range baseline {
			if got[i] != baseline[i] {
				t.Fatalf("track %d samples differ %s", i, stage)
			}
		}
	}
	check("with co64")

	tagFile, err := mp4tag.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	err = tagFile.Write(&mp4tag.MP4Tags{
		Title: "Tagged MV",
		Pictures: []*mp4tag.MP4Picture{{
			Format: mp4tag.ImageTypeAuto,
			Data:   []byte{0x89, 'P', 'N', 'G', 0, 0, 0, 0},
		}},
	}, nil)
	_ = tagFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	check("after tag write with co64")
}
