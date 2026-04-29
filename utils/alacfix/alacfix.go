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
// This program walks the ISO BMFF container to find the audio track's ALAC
// packets, parses each packet far enough to locate the end of its element
// body, and — if the tail does not already start with TYPE_END — overwrites
// the 3 bits there with 111 and zero-pads to the end of the packet. Packet
// sizes do not change, so no container rewriting is required.

// ---------- Bit reader ------------------------------------------------------

type bitReader struct {
	buf   []byte
	pos   int // bit position from MSB of buf[0]
	nbits int
}

func newBitReader(buf []byte) *bitReader {
	return &bitReader{buf: buf, nbits: len(buf) * 8}
}

func (b *bitReader) left() int { return b.nbits - b.pos }

func (b *bitReader) read(n int) (uint32, error) {
	if n == 0 {
		return 0, nil
	}
	if b.pos+n > b.nbits {
		return 0, io_EOF
	}
	var v uint32
	p := b.pos
	for i := 0; i < n; i++ {
		bit := uint32((b.buf[p>>3] >> (7 - uint(p&7))) & 1)
		v = (v << 1) | bit
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
		return io_EOF
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

var io_EOF = errors.New("bit reader EOF")

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
			return io_EOF
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
		decorrLeftWeight, err := br.read(8)
		if err != nil {
			return 0, false, err
		}
		// We don't bother validating decorr_shift>31 here — the FFmpeg check
		// uses the *shift* value, but we already discarded it; for a scanner,
		// what matters is bit position, not strict validation.
		_ = decorrLeftWeight

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
				return 0, false, io_EOF
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
			return 0, false, io_EOF
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
	bodyOff int
	endOff  int // exclusive
}

func walkAtoms(buf []byte, start, end int, recurse map[string]bool, out *[]atom) {
	p := start
	for p < end-8 {
		size := int(binary.BigEndian.Uint32(buf[p : p+4]))
		typ := string(buf[p+4 : p+8])
		hdr := 8
		if size == 1 {
			if p+16 > end {
				return
			}
			size = int(binary.BigEndian.Uint64(buf[p+8 : p+16]))
			hdr = 16
		} else if size == 0 {
			size = end - p
		}
		if size < hdr || p+size > end {
			return
		}
		body := p + hdr
		atomEnd := p + size
		*out = append(*out, atom{typ: typ, bodyOff: body, endOff: atomEnd})
		if recurse[typ] {
			off := body
			if typ == "meta" {
				off += 4 // version+flags
			}
			walkAtoms(buf, off, atomEnd, recurse, out)
		}
		p += size
	}
}

var bmffContainers = map[string]bool{
	"moov": true, "trak": true, "mdia": true, "minf": true,
	"stbl": true, "dinf": true, "udta": true, "ilst": true,
	"meta": true, "edts": true,
}

// ---------- Track metadata extraction --------------------------------------

type packetLoc struct {
	offset int64
	size   int
}

func parseAlacMagicCookie(c []byte) (alacParams, error) {
	var p alacParams
	if len(c) != 36 || string(c[4:8]) != "alac" {
		return p, errors.New("not an ALAC magic cookie")
	}
	// size(4) | 'alac'(4) | ver(4) | maxFrames(4) | compat(1) | sampleSize(1)
	// | histMult(1) | initHist(1) | riceLim(1) | channels(1) | maxRun(2)
	// | maxFrameSize(4) | avgBitrate(4) | sampleRate(4)
	p.maxSamplesPerFrame = binary.BigEndian.Uint32(c[12:16])
	p.sampleSize = c[17]
	p.riceHistoryMult = c[18]
	p.riceInitialHistory = c[19]
	p.riceLimit = c[20]
	p.channels = c[21]
	return p, nil
}

func findAudioTrack(data []byte) (alacParams, []packetLoc, error) {
	var top []atom
	walkAtoms(data, 0, len(data), bmffContainers, &top)

	var audioBody, audioEnd int
	found := false
	for _, a := range top {
		if a.typ != "trak" {
			continue
		}
		// Search inside this trak for "alac" tag.
		blob := data[a.bodyOff:a.endOff]
		if containsBytes(blob, []byte("alac")) {
			audioBody = a.bodyOff
			audioEnd = a.endOff
			found = true
			break
		}
	}
	if !found {
		return alacParams{}, nil, errors.New("no audio track found")
	}

	// Walk inside the audio trak.
	var sub []atom
	walkAtoms(data, audioBody, audioEnd, bmffContainers, &sub)

	var stsz, stco, stsc *atom
	is64 := false
	for i := range sub {
		switch sub[i].typ {
		case "stsz":
			stsz = &sub[i]
		case "stco":
			stco = &sub[i]
		case "co64":
			stco = &sub[i]
			is64 = true
		case "stsc":
			stsc = &sub[i]
		}
	}
	if stsz == nil || stco == nil || stsc == nil {
		return alacParams{}, nil, errors.New("missing stsz/stco/stsc")
	}

	// Find magic cookie (36-byte block whose size==36 and tag=='alac').
	var magic []byte
	for i := audioBody; i+36 <= audioEnd; i++ {
		if string(data[i+4:i+8]) == "alac" &&
			binary.BigEndian.Uint32(data[i:i+4]) == 36 {
			magic = data[i : i+36]
			break
		}
	}
	if magic == nil {
		return alacParams{}, nil, errors.New("ALAC magic cookie not found")
	}
	params, err := parseAlacMagicCookie(magic)
	if err != nil {
		return alacParams{}, nil, err
	}

	// stsz: version+flags(4) | sample_size(4) | sample_count(4) | entries
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

	// stco / co64
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

	// stsc
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
	for i := range samplesPerChunk {
		if samplesPerChunk[i] == 0 {
			samplesPerChunk[i] = runs[len(runs)-1].samplesPerChunk
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
	return params, locs, nil
}

func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// ---------- Patcher ---------------------------------------------------------

func patchInPlace(data []byte, off int64, size int, bodyEndBit int) bool {
	totalBits := size * 8
	if bodyEndBit < 0 || bodyEndBit+3 > totalBits {
		return false
	}
	// Set 3 tag bits to 1,1,1.
	for i := 0; i < 3; i++ {
		bp := bodyEndBit + i
		bi := off + int64(bp>>3)
		mask := byte(1 << uint(7-(bp&7)))
		data[bi] |= mask
	}
	// Zero everything from bodyEndBit+3 to end of packet.
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

func Run(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	params, locs, err := findAudioTrack(data)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d ALAC packets. max_samples_per_frame=%d sample_size=%d channels=%d\n",
		len(locs), params.maxSamplesPerFrame, params.sampleSize, params.channels)

	patched := 0
	type bad struct {
		idx        int
		off        int64
		size       int
		bodyEndBit int
	}
	var report []bad

	for idx, loc := range locs {
		pkt := data[loc.offset : loc.offset+int64(loc.size)]
		bodyEnd := findBodyEndBit(pkt, &params)
		if bodyEnd < 0 {
			continue
		}
		if bodyEnd == loc.size*8 {
			continue
		}
		// Already terminated by a TYPE_END?
		br := newBitReader(pkt)
		_ = br.skip(bodyEnd)
		if br.left() >= 3 {
			tag, _ := br.show(3)
			if tag == 7 {
				continue
			}
		}
		if patchInPlace(data, loc.offset, loc.size, bodyEnd) {
			patched++
			report = append(report, bad{idx, loc.offset, loc.size, bodyEnd})
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	fmt.Printf("Patched %d packet(s).\n", patched)
	for _, r := range report {
		fmt.Printf("  packet #%d  file_offset=0x%x  size=%d  body_ends_at_bit=%d  tail_overwritten=[%d..%d)\n",
			r.idx, r.off, r.size, r.bodyEndBit, r.bodyEndBit, r.size*8)
	}
	return nil
}
