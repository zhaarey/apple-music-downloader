package mv

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/itouakirai/go-mp4tag"
)

func concatenateFiles(t *testing.T, parts ...string) string {
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

func buildTestMV(t *testing.T) string {
	t.Helper()
	videoPath := concatenateFiles(t,
		filepath.Join("testdata", "V300", "init.mp4"),
		filepath.Join("testdata", "V300", "1.m4s"),
	)
	audioPath := concatenateFiles(t,
		filepath.Join("testdata", "A48", "init.mp4"),
		filepath.Join("testdata", "A48", "1.m4s"),
	)
	outputPath := filepath.Join(t.TempDir(), "mv.mp4")
	if err := Mux(videoPath, audioPath, outputPath); err != nil {
		t.Fatal(err)
	}
	return outputPath
}

func TestMuxCreatesProgressiveTwoTrackMP4(t *testing.T) {
	outputPath := buildTestMV(t)

	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	parsed, err := mp4.DecodeFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.IsFragmented() {
		t.Fatal("expected progressive output")
	}
	if parsed.Ftyp == nil || parsed.Ftyp.MajorBrand() != baseMediaMajorBrand {
		t.Fatalf("major brand = %v, want %s", parsed.Ftyp, baseMediaMajorBrand)
	}
	for _, brand := range baseMediaCompatibleBrands {
		if !containsString(parsed.Ftyp.CompatibleBrands(), brand) {
			t.Fatalf("compatible brands %v do not contain %s", parsed.Ftyp.CompatibleBrands(), brand)
		}
	}
	if parsed.Moov == nil || len(parsed.Moov.Traks) != 2 {
		t.Fatalf("expected two tracks, got %#v", parsed.Moov)
	}
	if parsed.Moov.Mvhd == nil || parsed.Moov.Mvhd.Duration == 0 {
		t.Fatal("expected non-zero movie duration")
	}

	gotHandlers := make(map[string]uint32)
	for _, trak := range parsed.Moov.Traks {
		if trak.Tkhd == nil || trak.Mdia == nil || trak.Mdia.Hdlr == nil {
			t.Fatal("track is missing required header boxes")
		}
		if trak.GetNrSamples() == 0 {
			t.Fatalf("track %d has no samples", trak.Tkhd.TrackID)
		}
		gotHandlers[trak.Mdia.Hdlr.HandlerType] = trak.Tkhd.TrackID
	}
	if gotHandlers["vide"] != videoTrackID {
		t.Fatalf("video track ID = %d, want %d", gotHandlers["vide"], videoTrackID)
	}
	if gotHandlers["soun"] != audioTrackID {
		t.Fatalf("audio track ID = %d, want %d", gotHandlers["soun"], audioTrackID)
	}
}

func TestMuxOutputCanBeTaggedWithoutMovingSamples(t *testing.T) {
	outputPath := buildTestMV(t)
	beforeVideo := sampleDigest(t, outputPath, 0)
	beforeAudio := sampleDigest(t, outputPath, 1)

	mp4File, err := mp4tag.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	err = mp4File.Write(&mp4tag.MP4Tags{
		Title:  "Tagged MV",
		Artist: "Test Artist",
		Pictures: []*mp4tag.MP4Picture{{
			Format: mp4tag.ImageTypeAuto,
			Data:   []byte{0x89, 'P', 'N', 'G', 0, 0, 0, 0},
		}},
	}, nil)
	_ = mp4File.Close()
	if err != nil {
		t.Fatal(err)
	}

	if afterVideo := sampleDigest(t, outputPath, 0); afterVideo != beforeVideo {
		t.Fatal("video sample data changed after tag write")
	}
	if afterAudio := sampleDigest(t, outputPath, 1); afterAudio != beforeAudio {
		t.Fatal("audio sample data changed after tag write")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func sampleDigest(t *testing.T, path string, trackIndex int) [sha256.Size]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	parsed, err := mp4.DecodeFile(file, mp4.WithDecodeMode(mp4.DecModeLazyMdat))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Moov == nil || trackIndex < 0 || trackIndex >= len(parsed.Moov.Traks) {
		t.Fatalf("track index %d out of range", trackIndex)
	}
	trak := parsed.Moov.Traks[trackIndex]
	nrSamples := trak.GetNrSamples()
	ranges, err := trak.GetRangesForSampleInterval(1, nrSamples)
	if err != nil {
		t.Fatal(err)
	}

	hash := sha256.New()
	for _, dataRange := range ranges {
		if _, err := file.Seek(int64(dataRange.Offset), io.SeekStart); err != nil {
			t.Fatal(err)
		}
		if _, err := io.CopyN(hash, file, int64(dataRange.Size)); err != nil {
			t.Fatal(err)
		}
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest
}
