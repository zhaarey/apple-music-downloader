package alacfix

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// alacfix.go — Patch malformed ALAC packets in an .m4a/.mp4 file in place.
//
// Some ALAC encoders emit packets that are missing the TYPE_END (3-bit value 7)
// terminator at the end of the bitstream. FFmpeg's decoder rejects them with
// errors like "invalid element channel count" or "Syntax element 4 is not
// implemented" because after the legitimate first element it tries to parse
// the garbage tail as a new element header.
//
// This file walks the ISO BMFF container, locates every audio track that
// uses the ALAC codec (by checking the format code in stsd's sample entry,
// not by string matching), parses each packet far enough to find where the
// element body ends, and — if the tail does not already start with TYPE_END —
// overwrites the 3 bits there with 111 and zero-pads to the end of the
// packet. Packet sizes do not change, so the container itself is not rewritten.
//
// Tracks that use any other codec (AAC, FLAC, etc.) are silently skipped.

// ---------- Bit reader ------------------------------------------------------

type bitReader struct {
	buf   []byte
	pos   int // bit position from MSB of buf[0]
	nbits int
}

var errEOF = errors.New("bit reader EOF")

func newBitReader(buf []byte) *bitReader {
	return &bitReader{buf: buf, nbits: len(buf) * 8}
}

func (b *bitReader) left() int { return b.nbits - b.pos }

func (b *bitReader) read(n int) (uint32, error) {
	if n == 0 {
		return 0, nil
	}
	if b.pos+n > b.nbits {
		return 0, errEOF
	}
	var v uint32
	p := b.pos
	for i := 0; i < n; i++ {
		v = (v << 1) | uint32((b.buf[p>>3]>>(7-uint(p&7)))&1)
		p++
	}
	b.pos = p
	return v, nil
}

func (b *bitReader) show(n int) (uint32, error) {
	save := b.pos
	v, err := b.read(n)
	b.pos = save
	return v, err
}

func (b *bitReader) skip(n int) error {
	if b.pos+n > b.nbits {
		return errEOF
	}
	b.pos += n
	return nil
}

func (b *bitReader) readSigned(n int) (int32, error) {
	v, err := b.read(n)
	if err != nil {
		return 0, err
	}
	if v&(1<<uint(n-1)) != 0 {
		return int32(v) - int32(1<<uint(n)), nil
	}
	return int32(v), nil
}

func (b *bitReader) unary09() (uint32, error) {
	cnt := uint32(0)
	for cnt < 9 {
		v, err := b.read(1)
		if err != nil {
			return 0, err
		}
		if v == 0 {
			return cnt, nil
		}
		cnt++
	}
	return 9, nil
}

func avLog2(x uint32) int {
	if x == 0 {
		return 0
	}
	r := 0
	for x > 1 {
		x >>= 1
		r++
	}
	return r
}

// ---------- ALAC element body scanner --------------------------------------
// Mirrors libavcodec/alac.c: decode_scalar, rice_decompress, decode_element.

type alacParams struct {
	maxSamplesPerFrame uint32
	sampleSize         uint8
	riceHistoryMult    uint8
	riceInitialHistory uint8
	riceLimit          uint8
	channels           uint8
}

func decodeScalar(br *bitReader, k int, bps int) (uint32, error) {
	x, err := br.unary09()
	if err != nil {
		return 0, err
	}
	if x > 8 {
		return br.read(bps)
	}
	if k != 1 {
		extrabits, err := br.show(k)
		if err != nil {
			return 0, err
		}
		x = (x << uint(k)) - x
		if extrabits > 1 {
			x += extrabits - 1
			if err := br.skip(k); err != nil {
				return 0, err
			}
		} else {
			if err := br.skip(k - 1); err != nil {
				return 0, err
			}
		}
	}
	return x, nil
}

func riceDecompress(br *bitReader, nbSamples int, bps int, rhmEff uint32, p *alacParams) error {
	history := uint32(p.riceInitialHistory)
	signMod := uint32(0)
	limit := int(p.riceLimit)
	cap := nbSamples*4 + 100
	iters := 0
	for i := 0; i < nbSamples; {
		iters++
		if iters > cap {
			return errors.New("rice runaway")
		}
		if br.left() <= 0 {
			return errEOF
		}
		k := avLog2((history >> 9) + 3)
		if k > limit {
			k = limit
		}
		x, err := decodeScalar(br, k, bps)
		if err != nil {
			return err
		}
		x = x + signMod
		signMod = 0
		if x > 0xFFFF {
			history = 0xFFFF
		} else {
			history = history + x*rhmEff - ((history * rhmEff) >> 9)
		}
		if history < 128 && (i+1) < nbSamples {
			k2 := 7 - avLog2(history) + int((history+16)>>6)
			if k2 > limit {
				k2 = limit
			}
			blockSize, err := decodeScalar(br, k2, 16)
			if err != nil {
				return err
			}
			if blockSize > 0 {
				if int(blockSize) >= nbSamples-i {
					blockSize = uint32(nbSamples - i - 1)
				}
				i += int(blockSize)
			}
			if blockSize <= 0xFFFF {
				signMod = 1
			}
			history = 0
		}
		i++
	}
	return nil
}

// scanOneElement consumes one element from br. Returns (channels_used,
// is_end_tag, error). If an unsupported element tag is hit, returns an error.
func scanOneElement(br *bitReader, p *alacParams) (int, bool, error) {
	elem, err := br.read(3)
	if err != nil {
		return 0, false, err
	}
	if elem == 7 {
		return 0, true, nil
	}
	if elem > 1 && elem != 3 {
		return 0, false, fmt.Errorf("unsupported element tag %d", elem)
	}
	channels := 1
	if elem == 1 {
		channels = 2
	}
	if err := br.skip(4); err != nil {
		return 0, false, err
	}
	if err := br.skip(12); err != nil {
		return 0, false, err
	}
	hasSize, err := br.read(1)
	if err != nil {
		return 0, false, err
	}
	extraBitsRaw, err := br.read(2)
	if err != nil {
		return 0, false, err
	}
	extraBits := int(extraBitsRaw) << 3
	bps := int(p.sampleSize) - extraBits + channels - 1
	if bps > 32 || bps < 1 {
		return 0, false, fmt.Errorf("bad bps %d", bps)
	}
	notCompressed, err := br.read(1)
	if err != nil {
		return 0, false, err
	}
	isCompressed := notCompressed == 0
	var outputSamples uint32
	if hasSize != 0 {
		outputSamples, err = br.read(32)
		if err != nil {
			return 0, false, err
		}
	} else {
		outputSamples = p.maxSamplesPerFrame
	}
	if outputSamples == 0 || outputSamples > p.maxSamplesPerFrame {
		return 0, false, fmt.Errorf("bad output_samples %d", outputSamples)
	}

	if isCompressed {
		if _, err := br.read(8); err != nil { // decorr_shift
			return 0, false, err
		}
		if _, err := br.read(8); err != nil { // decorr_left_weight
			return 0, false, err
		}
		rhms := make([]uint32, channels)
		for c := 0; c < channels; c++ {
			if _, err := br.read(4); err != nil { // pred_type
				return 0, false, err
			}
			lpcQuant, err := br.read(4)
			if err != nil {
				return 0, false, err
			}
			rhm, err := br.read(3)
			if err != nil {
				return 0, false, err
			}
			lpcOrder, err := br.read(5)
			if err != nil {
				return 0, false, err
			}
			if lpcOrder >= p.maxSamplesPerFrame || lpcQuant == 0 {
				return 0, false, fmt.Errorf("bad lpc")
			}
			for j := uint32(0); j < lpcOrder; j++ {
				if _, err := br.readSigned(16); err != nil {
					return 0, false, err
				}
			}
			rhms[c] = rhm
		}
		if extraBits != 0 {
			need := int(outputSamples) * channels * extraBits
			if br.left() < need {
				return 0, false, errEOF
			}
			if err := br.skip(need); err != nil {
				return 0, false, err
			}
		}
		for c := 0; c < channels; c++ {
			rhmEff := (rhms[c] * uint32(p.riceHistoryMult)) / 4
			if err := riceDecompress(br, int(outputSamples), bps, rhmEff, p); err != nil {
				return 0, false, err
			}
		}
	} else {
		need := int(outputSamples) * channels * int(p.sampleSize)
		if br.left() < need {
			return 0, false, errEOF
		}
		if err := br.skip(need); err != nil {
			return 0, false, err
		}
	}
	return channels, false, nil
}

// findBodyEndBit returns the bit position right after the last non-END
// element body. -1 means parse failure.
func findBodyEndBit(packet []byte, p *alacParams) int {
	br := newBitReader(packet)
	chUsed := 0
	lastEnd := -1
	for br.left() >= 3 {
		nCh, isEnd, err := scanOneElement(br, p)
		if err != nil {
			return -1
		}
		if isEnd {
			return br.pos
		}
		lastEnd = br.pos
		chUsed += nCh
		if chUsed >= int(p.channels) {
			return lastEnd
		}
	}
	return lastEnd
}

// ---------- ISO BMFF walker -------------------------------------------------

type atom struct {
	typ     string
	hdrOff  int
	bodyOff int
	endOff  int
}

// findChild returns the first direct child atom of the given type within [start, end).
func findChild(buf []byte, start, end int, typ string) (atom, bool) {
	p := start
	for p < end-8 {
		size := int(binary.BigEndian.Uint32(buf[p : p+4]))
		atomType := string(buf[p+4 : p+8])
		hdr := 8
		if size == 1 {
			if p+16 > end {
				return atom{}, false
			}
			size = int(binary.BigEndian.Uint64(buf[p+8 : p+16]))
			hdr = 16
		} else if size == 0 {
			size = end - p
		}
		if size < hdr || p+size > end {
			return atom{}, false
		}
		if atomType == typ {
			return atom{typ: atomType, hdrOff: p, bodyOff: p + hdr, endOff: p + size}, true
		}
		p += size
	}
	return atom{}, false
}

// findAllChildren returns every direct child of the given type within [start, end).
func findAllChildren(buf []byte, start, end int, typ string) []atom {
	var out []atom
	p := start
	for p < end-8 {
		size := int(binary.BigEndian.Uint32(buf[p : p+4]))
		atomType := string(buf[p+4 : p+8])
		hdr := 8
		if size == 1 {
			if p+16 > end {
				return out
			}
			size = int(binary.BigEndian.Uint64(buf[p+8 : p+16]))
			hdr = 16
		} else if size == 0 {
			size = end - p
		}
		if size < hdr || p+size > end {
			return out
		}
		if atomType == typ {
			out = append(out, atom{typ: atomType, hdrOff: p, bodyOff: p + hdr, endOff: p + size})
		}
		p += size
	}
	return out
}

// ---------- Track metadata extraction --------------------------------------

type packetLoc struct {
	offset int64
	size   int
}

type trackData struct {
	trackID uint32
	params  alacParams
	locs    []packetLoc
}

// parseAlacMagicCookie parses the ALAC specific box payload (without atom header).
// Layout: version_flags(4) | maxFrames(4) | compat(1) | sampleSize(1)
//
//	| histMult(1) | initHist(1) | riceLim(1) | channels(1) | ...
func parseAlacMagicCookie(c []byte) (alacParams, error) {
	var p alacParams
	if len(c) < 24 {
		return p, errors.New("ALAC config too short")
	}
	p.maxSamplesPerFrame = binary.BigEndian.Uint32(c[4:8])
	p.sampleSize = c[9]
	p.riceHistoryMult = c[10]
	p.riceInitialHistory = c[11]
	p.riceLimit = c[12]
	p.channels = c[13]
	return p, nil
}

// extractAlacConfig reads the ALAC magic cookie from inside an `alac` sample
// entry (handles both direct and 'wave'-wrapped layouts).
func extractAlacConfig(data []byte, sampleEntry atom) (alacParams, error) {
	// Sample entry has a fixed 28-byte audio header before any child atoms.
	childStart := sampleEntry.bodyOff + 28
	if cfg, ok := findChild(data, childStart, sampleEntry.endOff, "alac"); ok {
		if cfg.endOff-cfg.bodyOff < 28 {
			return alacParams{}, errors.New("alac config atom too small")
		}
		return parseAlacMagicCookie(data[cfg.bodyOff : cfg.bodyOff+28])
	}
	if wave, ok := findChild(data, childStart, sampleEntry.endOff, "wave"); ok {
		if cfg, ok := findChild(data, wave.bodyOff, wave.endOff, "alac"); ok {
			if cfg.endOff-cfg.bodyOff < 28 {
				return alacParams{}, errors.New("alac config atom too small")
			}
			return parseAlacMagicCookie(data[cfg.bodyOff : cfg.bodyOff+28])
		}
	}
	return alacParams{}, errors.New("no ALAC config inside sample entry")
}

func readPacketLocations(data []byte, stbl atom) ([]packetLoc, error) {
	stsz, ok := findChild(data, stbl.bodyOff, stbl.endOff, "stsz")
	if !ok {
		return nil, errors.New("stsz missing")
	}
	stsc, ok := findChild(data, stbl.bodyOff, stbl.endOff, "stsc")
	if !ok {
		return nil, errors.New("stsc missing")
	}
	stco, ok := findChild(data, stbl.bodyOff, stbl.endOff, "stco")
	is64 := false
	if !ok {
		stco, ok = findChild(data, stbl.bodyOff, stbl.endOff, "co64")
		if !ok {
			return nil, errors.New("stco/co64 missing")
		}
		is64 = true
	}

	b := stsz.bodyOff
	defaultSize := binary.BigEndian.Uint32(data[b+4 : b+8])
	count := int(binary.BigEndian.Uint32(data[b+8 : b+12]))
	sizes := make([]uint32, count)
	if defaultSize == 0 {
		for i := 0; i < count; i++ {
			sizes[i] = binary.BigEndian.Uint32(data[b+12+4*i : b+16+4*i])
		}
	} else {
		for i := 0; i < count; i++ {
			sizes[i] = defaultSize
		}
	}

	b = stco.bodyOff
	ent := int(binary.BigEndian.Uint32(data[b+4 : b+8]))
	chunkOff := make([]int64, ent)
	p := b + 8
	if is64 {
		for i := 0; i < ent; i++ {
			chunkOff[i] = int64(binary.BigEndian.Uint64(data[p : p+8]))
			p += 8
		}
	} else {
		for i := 0; i < ent; i++ {
			chunkOff[i] = int64(binary.BigEndian.Uint32(data[p : p+4]))
			p += 4
		}
	}

	b = stsc.bodyOff
	ent = int(binary.BigEndian.Uint32(data[b+4 : b+8]))
	type stscRun struct{ firstChunk, samplesPerChunk uint32 }
	runs := make([]stscRun, ent)
	p = b + 8
	for i := 0; i < ent; i++ {
		runs[i] = stscRun{
			firstChunk:      binary.BigEndian.Uint32(data[p : p+4]),
			samplesPerChunk: binary.BigEndian.Uint32(data[p+4 : p+8]),
		}
		p += 12
	}

	samplesPerChunk := make([]uint32, len(chunkOff))
	for i, r := range runs {
		var nextFC uint32
		if i+1 < len(runs) {
			nextFC = runs[i+1].firstChunk
		} else {
			nextFC = uint32(len(chunkOff)) + 1
		}
		for c := r.firstChunk; c < nextFC && int(c-1) < len(chunkOff); c++ {
			samplesPerChunk[c-1] = r.samplesPerChunk
		}
	}
	if len(runs) > 0 {
		for i := range samplesPerChunk {
			if samplesPerChunk[i] == 0 {
				samplesPerChunk[i] = runs[len(runs)-1].samplesPerChunk
			}
		}
	}

	locs := make([]packetLoc, 0, len(sizes))
	sampleIdx := 0
	for c, choff := range chunkOff {
		cur := choff
		spc := int(samplesPerChunk[c])
		for k := 0; k < spc && sampleIdx < len(sizes); k++ {
			sz := int(sizes[sampleIdx])
			locs = append(locs, packetLoc{offset: cur, size: sz})
			cur += int64(sz)
			sampleIdx++
		}
		if sampleIdx >= len(sizes) {
			break
		}
	}
	return locs, nil
}

// findAlacTracks returns one entry per audio track whose first sample entry
// in stsd has format == 'alac'. All other tracks (video, AAC audio, etc.)
// are silently skipped.
func findAlacTracks(data []byte) ([]trackData, error) {
	if len(data) < 8 {
		return nil, errors.New("file too small")
	}
	moov, ok := findChild(data, 0, len(data), "moov")
	if !ok {
		return nil, errors.New("no moov atom (not an MP4/M4A?)")
	}

	var tracks []trackData
	for _, trak := range findAllChildren(data, moov.bodyOff, moov.endOff, "trak") {
		var trackID uint32
		if tkhd, ok := findChild(data, trak.bodyOff, trak.endOff, "tkhd"); ok {
			b := tkhd.bodyOff
			version := data[b]
			if version == 0 && tkhd.endOff-b >= 20 {
				trackID = binary.BigEndian.Uint32(data[b+12 : b+16])
			} else if version == 1 && tkhd.endOff-b >= 32 {
				trackID = binary.BigEndian.Uint32(data[b+20 : b+24])
			}
		}

		mdia, ok := findChild(data, trak.bodyOff, trak.endOff, "mdia")
		if !ok {
			continue
		}
		// Require handler_type == 'soun'.
		hdlr, ok := findChild(data, mdia.bodyOff, mdia.endOff, "hdlr")
		if !ok {
			continue
		}
		hb := hdlr.bodyOff
		if hdlr.endOff-hb < 12 || string(data[hb+8:hb+12]) != "soun" {
			continue
		}
		minf, ok := findChild(data, mdia.bodyOff, mdia.endOff, "minf")
		if !ok {
			continue
		}
		stbl, ok := findChild(data, minf.bodyOff, minf.endOff, "stbl")
		if !ok {
			continue
		}
		stsd, ok := findChild(data, stbl.bodyOff, stbl.endOff, "stsd")
		if !ok {
			continue
		}
		// stsd body: version+flags(4) | entry_count(4) | sample_entries...
		b := stsd.bodyOff
		if stsd.endOff-b < 8 {
			continue
		}
		entryCount := binary.BigEndian.Uint32(data[b+4 : b+8])
		if entryCount == 0 {
			continue
		}
		entryStart := b + 8
		if entryStart+8 > stsd.endOff {
			continue
		}
		entrySize := int(binary.BigEndian.Uint32(data[entryStart : entryStart+4]))
		entryType := string(data[entryStart+4 : entryStart+8])
		if entrySize < 8 || entryStart+entrySize > stsd.endOff {
			continue
		}
		if entryType != "alac" {
			continue
		}
		sampleEntry := atom{
			typ:     entryType,
			hdrOff:  entryStart,
			bodyOff: entryStart + 8,
			endOff:  entryStart + entrySize,
		}
		params, err := extractAlacConfig(data, sampleEntry)
		if err != nil {
			return nil, fmt.Errorf("track %d: %w", trackID, err)
		}
		locs, err := readPacketLocations(data, stbl)
		if err != nil {
			return nil, fmt.Errorf("track %d: %w", trackID, err)
		}
		tracks = append(tracks, trackData{trackID: trackID, params: params, locs: locs})
	}
	return tracks, nil
}

// ---------- Patcher ---------------------------------------------------------

func patchInPlace(data []byte, off int64, size int, bodyEndBit int) bool {
	totalBits := size * 8
	if bodyEndBit < 0 || bodyEndBit+3 > totalBits {
		return false
	}
	for i := 0; i < 3; i++ {
		bp := bodyEndBit + i
		bi := off + int64(bp>>3)
		mask := byte(1 << uint(7-(bp&7)))
		data[bi] |= mask
	}
	padStart := bodyEndBit + 3
	bi := off + int64(padStart>>3)
	bitInByte := padStart & 7
	if bitInByte != 0 {
		keep := byte(0xFF<<uint(8-bitInByte)) & 0xFF
		data[bi] &= keep
		bi++
	}
	endByte := off + int64(size)
	for j := bi; j < endByte; j++ {
		data[j] = 0
	}
	return true
}

// ---------- main ------------------------------------------------------------

func Run(path string, outPath ...string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dst := path
	if len(outPath) > 0 && outPath[0] != "" {
		dst = outPath[0]
	}
	tracks, err := findAlacTracks(data)
	if err != nil {
		return err
	}
	if len(tracks) == 0 {
		return nil
	}

	type bad struct {
		trackID    uint32
		idx        int
		off        int64
		size       int
		bodyEndBit int
	}

	patched := 0
	var report []bad

	for _, td := range tracks {
		params := td.params
		fmt.Printf("Track #%d: %d packets, max_samples_per_frame=%d sample_size=%d channels=%d\n",
			td.trackID, len(td.locs), params.maxSamplesPerFrame, params.sampleSize, params.channels)

		for idx, loc := range td.locs {
			pkt := data[loc.offset : loc.offset+int64(loc.size)]
			bodyEnd := findBodyEndBit(pkt, &params)
			if bodyEnd < 0 {
				continue
			}
			if bodyEnd == loc.size*8 {
				continue
			}
			br := newBitReader(pkt)
			_ = br.skip(bodyEnd)
			if br.left() >= 3 {
				if tag, _ := br.show(3); tag == 7 {
					continue
				}
			}
			if patchInPlace(data, loc.offset, loc.size, bodyEnd) {
				patched++
				report = append(report, bad{td.trackID, idx, loc.offset, loc.size, bodyEnd})
			}
		}
	}

	if patched > 0 {
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return err
		}
		fmt.Printf("Patched %d packet(s).\n", patched)
		for _, r := range report {
			fmt.Printf("  track #%d packet #%d  file_offset=0x%x  size=%d  body_ends_at_bit=%d  tail_overwritten=[%d..%d)\n",
				r.trackID, r.idx, r.off, r.size, r.bodyEndBit, r.bodyEndBit, r.size*8)
		}
	}
	return nil
}
