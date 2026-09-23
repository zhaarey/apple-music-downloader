package alacfix

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeAtom(typ string, payload []byte) []byte {
	b := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(b[0:4], uint32(len(b)))
	copy(b[4:8], typ)
	copy(b[8:], payload)
	return b
}

func makeTkhd() []byte {
	payload := make([]byte, 24)
	payload[0] = 0 // version
	binary.BigEndian.PutUint32(payload[12:16], 1) // trackID = 1
	return makeAtom("tkhd", payload)
}

func makeHdlr() []byte {
	payload := make([]byte, 24)
	copy(payload[8:12], "soun")
	return makeAtom("hdlr", payload)
}

func makeAlacMagicCookie() []byte {
	cookie := make([]byte, 28)
	binary.BigEndian.PutUint32(cookie[4:8], 4096) // maxSamplesPerFrame
	cookie[9] = 16                                // sampleSize
	cookie[10] = 40                               // riceHistoryMult
	cookie[11] = 10                               // riceInitialHistory
	cookie[12] = 14                               // riceLimit
	cookie[13] = 2                                // channels
	return makeAtom("alac", cookie)
}

func makeStsd() []byte {
	alacBox := makeAlacMagicCookie()
	audioHdr := make([]byte, 28)
	sampleEntryPayload := append(audioHdr, alacBox...)
	sampleEntry := makeAtom("alac", sampleEntryPayload)

	stsdBody := make([]byte, 8)
	binary.BigEndian.PutUint32(stsdBody[4:8], 1) // entryCount = 1
	stsdBody = append(stsdBody, sampleEntry...)
	return makeAtom("stsd", stsdBody)
}

func makeStbl(packetOffset int, packetSizes []int) []byte {
	stsd := makeStsd()

	// stsz
	stszBody := make([]byte, 12+4*len(packetSizes))
	binary.BigEndian.PutUint32(stszBody[8:12], uint32(len(packetSizes))) // count
	for i, sz := range packetSizes {
		binary.BigEndian.PutUint32(stszBody[12+4*i:16+4*i], uint32(sz))
	}
	stsz := makeAtom("stsz", stszBody)

	// stsc
	stscBody := make([]byte, 20)
	binary.BigEndian.PutUint32(stscBody[4:8], 1)                           // count = 1
	binary.BigEndian.PutUint32(stscBody[8:12], 1)                          // firstChunk = 1
	binary.BigEndian.PutUint32(stscBody[12:16], uint32(len(packetSizes))) // samplesPerChunk
	binary.BigEndian.PutUint32(stscBody[16:20], 1)                         // sampleDescIndex = 1
	stsc := makeAtom("stsc", stscBody)

	// stco
	stcoBody := make([]byte, 12)
	binary.BigEndian.PutUint32(stcoBody[4:8], 1)                    // count = 1
	binary.BigEndian.PutUint32(stcoBody[8:12], uint32(packetOffset)) // chunkOff = packetOffset
	stco := makeAtom("stco", stcoBody)

	var stblPayload []byte
	stblPayload = append(stblPayload, stsd...)
	stblPayload = append(stblPayload, stsz...)
	stblPayload = append(stblPayload, stsc...)
	stblPayload = append(stblPayload, stco...)
	return makeAtom("stbl", stblPayload)
}

func makeMoov(packetOffset int, packetSizes []int) []byte {
	tkhd := makeTkhd()
	hdlr := makeHdlr()
	stbl := makeStbl(packetOffset, packetSizes)
	minf := makeAtom("minf", stbl)
	mdia := makeAtom("mdia", append(hdlr, minf...))
	trak := makeAtom("trak", append(tkhd, mdia...))
	return makeAtom("moov", trak)
}

// createTestPacket builds an ALAC uncompressed element packet.
// Total element body: 87 bits.
// If withEndTag is true, bits 87..89 are 111 (TYPE_END).
// If false, bits 87..89 are 000.
func createTestPacket(withEndTag bool) []byte {
	buf := make([]byte, 16) // 128 bits
	// elem = 1 (bits 0..2: 001)
	buf[0] = 0x20
	// bits 3..6: 0000 (skip 4)
	// bits 7..18: 000000000000 (skip 12)
	// bit 19: hasSize = 1 -> set bit 3 of buf[2]
	buf[2] |= (1 << 4)
	// bits 20..21: extraBits = 00
	// bit 22: notCompressed = 1 -> set bit 1 of buf[2]
	buf[2] |= (1 << 1)
	// bits 23..54: outputSamples = 1 (32 bits)
	// Setting outputSamples = 1: bit 54 is bit 1 of buf[6]
	buf[6] |= (1 << 1)
	// bits 55..86 (32 bits): uncompressed samples = 0
	// bit 87..89: tag
	if withEndTag {
		// byte 10 is bits 80..87. bit 87 is bit 0 of buf[10].
		buf[10] |= (1 << 0)
		// byte 11 is bits 88..95. bits 88 and 89 are bits 7 and 6 of buf[11].
		buf[11] |= (3 << 6)
	}
	return buf
}

func buildTestMP4Multi(t *testing.T, withEndTags []bool) string {
	t.Helper()
	var pktPayload []byte
	var sizes []int
	for _, endTag := range withEndTags {
		pkt := createTestPacket(endTag)
		pktPayload = append(pktPayload, pkt...)
		sizes = append(sizes, len(pkt))
	}
	placeholderMoov := makeMoov(0, sizes)
	moovSize := len(placeholderMoov)
	mdatHdrSize := 8
	packetOffset := moovSize + mdatHdrSize

	moov := makeMoov(packetOffset, sizes)
	mdat := makeAtom("mdat", pktPayload)

	var full []byte
	full = append(full, moov...)
	full = append(full, mdat...)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.m4a")
	if err := os.WriteFile(path, full, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func buildTestMP4(t *testing.T, withEndTag bool) string {
	return buildTestMP4Multi(t, []bool{withEndTag})
}

func captureStdout(f func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	outChan := make(chan string)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		outChan <- b.String()
	}()

	runErr := f()
	_ = w.Close()
	os.Stdout = oldStdout
	output := <-outChan
	return output, runErr
}

func TestAlacFix_IntactAudio(t *testing.T) {
	path := buildTestMP4(t, true)

	res, err := Fix(path, false)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}
	if res.TracksCount != 1 {
		t.Fatalf("expected 1 track, got %d", res.TracksCount)
	}
	if res.Patched != 0 {
		t.Fatalf("expected 0 patched, got %d", res.Patched)
	}

	out, err := captureStdout(func() error {
		return Run(path, false)
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	if trimmed != "Audio integrity intact" {
		t.Fatalf("expected 'Audio integrity intact', got %q", trimmed)
	}
}

func TestAlacFix_RepairedAudio(t *testing.T) {
	path := buildTestMP4(t, false)

	// First verify Fix
	res, err := Fix(path, false)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}
	if res.TracksCount != 1 {
		t.Fatalf("expected 1 track, got %d", res.TracksCount)
	}
	if res.Patched != 1 {
		t.Fatalf("expected 1 patched, got %d", res.Patched)
	}

	// Now re-create malformed file and test Run
	path2 := buildTestMP4(t, false)
	out, err := captureStdout(func() error {
		return Run(path2, false)
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	expected := "Repaired 1 affected packet"
	if trimmed != expected {
		t.Fatalf("expected %q, got %q", expected, trimmed)
	}

	// Verify that now it has integrity
	out2, err := captureStdout(func() error {
		return Run(path2, false)
	})
	if err != nil {
		t.Fatalf("Run failed on repaired file: %v", err)
	}
	if strings.TrimSpace(out2) != "Audio integrity intact" {
		t.Fatalf("expected 'Audio integrity intact', got %q", strings.TrimSpace(out2))
	}
}

func TestAlacFix_NonAlacFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "dummy.mp4")
	moov := makeAtom("moov", []byte("dummy payload"))
	if err := os.WriteFile(path, moov, 0644); err != nil {
		t.Fatal(err)
	}

	res, err := Fix(path, false)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}
	if res.TracksCount != 0 {
		t.Fatalf("expected 0 tracks, got %d", res.TracksCount)
	}

	out, err := captureStdout(func() error {
		return Run(path, false)
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty output for non-alac, got %q", out)
	}
}

func TestAlacFix_MultiplePackets(t *testing.T) {
	// 3 packets: 2 malformed, 1 intact
	path := buildTestMP4Multi(t, []bool{false, true, false})

	res, err := Fix(path, false)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}
	if res.TracksCount != 1 {
		t.Fatalf("expected 1 track, got %d", res.TracksCount)
	}
	if res.Patched != 2 {
		t.Fatalf("expected 2 patched, got %d", res.Patched)
	}

	path2 := buildTestMP4Multi(t, []bool{false, true, false})
	out, err := captureStdout(func() error {
		return Run(path2, false)
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	expected := "Repaired 2 affected packets"
	if trimmed != expected {
		t.Fatalf("expected %q, got %q", expected, trimmed)
	}
}

func TestAlacFix_OutputPathLeavesInputUntouched(t *testing.T) {
	path := buildTestMP4Multi(t, []bool{false, true, false})
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.m4a")

	res, err := Fix(path, false, out)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}
	if res.Patched != 2 {
		t.Fatalf("expected 2 patched, got %d", res.Patched)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(orig, after) {
		t.Fatal("input file was modified")
	}

	res, err = Fix(out, false)
	if err != nil {
		t.Fatalf("Fix on output failed: %v", err)
	}
	if res.Patched != 0 {
		t.Fatalf("expected repaired output, got %d packets still patched", res.Patched)
	}
}
