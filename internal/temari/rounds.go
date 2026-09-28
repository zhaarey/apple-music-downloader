// Package temari is a pure-Go port of Temari
// (https://github.com/WorldObservationLog/Temari, MIT): Apple Music FairPlay
// SAMPLE-AES sample decryption driven by a key template (ctx table + state
// + entry registers) obtained from a 40020-style key server.
//
// The round chain in rounds_gen.go is transpiled from upstream
// crate/src/rounds_gen.rs; this file ports crate/src/rounds.rs.
package temari

import "encoding/binary"

// CtxSize is the size of the ctx lookup table in a template.
const CtxSize = 0x8000

// StSize is the number of state slots kept; the round chain only reads
// st[0..=1920].
const StSize = 2048

// St is the per-sample round-chain state.
type St = [StSize]uint32

type r1Entry struct {
	rdx, rcx, rax, r9, rbp uint32
}

// Template is a decryption template. It is immutable after construction and
// safe for concurrent use.
type Template struct {
	ctx     *ctxTables
	stInit  St
	r1Entry r1Entry
}

type round1MidOut struct {
	rax, r11, r12, r13, r15, r8, r14, rbx, r10, rcx uint32
}

type round1TailOut struct {
	rdi, rsi, rdx, rcx, r8, r9, rax, rbx, r10, r11, r12, r13, r14, r15 uint32
}

type round2Vars struct {
	// [st[0x48], st[0x250], st[0x290]] captured at the snapshot point.
	cp12             [3]uint32
	v171, v187, v189 uint32
}

// ctxTables holds the lookup tables sliced out of the template ctx blob.
// The round chain only ever indexes ctx as <constant base> + <byte>, so each
// table is its own 256-entry array: lookups are a single load with no bounds
// check, and u32 tables are pre-decoded from little-endian bytes.
type ctxTables struct {
	// u32 tables (t4), named by byte offset in ctx.
	t12272, t13328, t14672, t15728, t17040, t18080, t19136, t20208 [256]uint32
	// byte tables (ctxb), named by byte offset in ctx.
	s12000, s16768, s21248 [256]byte
	c14573                 byte
}

func newCtxTables(ctx []byte) *ctxTables {
	c := new(ctxTables)
	for _, t := range []struct {
		off int
		tab *[256]uint32
	}{
		{12272, &c.t12272}, {13328, &c.t13328}, {14672, &c.t14672}, {15728, &c.t15728},
		{17040, &c.t17040}, {18080, &c.t18080}, {19136, &c.t19136}, {20208, &c.t20208},
	} {
		for i := range t.tab {
			t.tab[i] = binary.LittleEndian.Uint32(ctx[t.off+i*4:])
		}
	}
	copy(c.s12000[:], ctx[12000:])
	copy(c.s16768[:], ctx[16768:])
	copy(c.s21248[:], ctx[21248:])
	c.c14573 = ctx[14573]
	return c
}

// t4 is a u32 table lookup by the low byte of idx.
func t4(tab *[256]uint32, idx uint32) uint32 {
	return tab[uint8(idx)]
}

// ctxb is a byte table lookup by the low byte of idx.
func ctxb(tab *[256]byte, idx uint32) uint32 {
	return uint32(tab[uint8(idx)])
}

// processBlock runs R1→R2→R3→CBC for block bi of ct and writes the
// plaintext into out16.
func processBlock(tmpl *Template, st *St, ct []byte, bi int, out16 *[16]byte) {
	b := uint32(bi)
	ctx := tmpl.ctx

	// Round-1 entry registers: rdi/rsi encode the block address; rdx, r9 and
	// rbp from the template (and the zeroed r8, rbx, r10-r15) are never read
	// by the simplified round chain.
	mid := round1Mid(ctx, st, ct, 0x1EB2C6B4^(b<<4), 0x8+(b<<4), tmpl.r1Entry.rcx, tmpl.r1Entry.rax)
	r2 := round1Tail(ctx, st, mid.rax, mid.r13&0xFF, mid.r15&0xFF, mid.r8&0xFF, mid.r14&0xFF)
	r2v := round2Sub6400(ctx, st, r2.rdi, r2.rsi, r2.rdx, r2.rcx, r2.r8, r2.r9, r2.rax, r2.rbx,
		r2.r10, r2.r11, r2.r13, r2.r14, r2.r15, 0)

	// R3 boundary parameters; cp12 = [st[0x48], st[0x250], st[0x290]].
	cp12 := &r2v.cp12
	r8p := cp12[2] ^ st[0x5B8] ^ cp12[1]
	v6 := t4(&ctx.t18080, st[0x390]^0x2B) ^
		st[0x5F0] ^
		t4(&ctx.t19136, ((r8p>>24)&0xFF)^0x29) ^
		t4(&ctx.t12272, ((st[0x298]>>16)&0xFF)^0xD6)
	v9 := t4(&ctx.t19136, cp12[0]) ^
		st[0x540] ^
		t4(&ctx.t12272, ((r2v.v171>>16)&0xFF)^0x69)
	v11 := t4(&ctx.t14672, (st[0x298]&0xFF)^0x57) ^
		st[0x538] ^
		t4(&ctx.t18080, ((r8p>>8)&0xFF)^0x2F)

	a2 := st[0x270]
	a6 := st[0x280]
	round3Sub8000(ctx, st, a2, r2v.v189, r8p&0xFF, (r8p>>16)&0xFFFF, a6, v6,
		(r8p>>16)&0xFF, v9, r2v.v187&0xFF, v11, r2v.v189, out16)

	// CBC state pass-through
	st[0x108] = st[0x180]
	st[0x220] += 0x10
}

// DecryptInto decrypts sample into out (len(out) must be >= len(sample)).
// The 16-byte-aligned prefix is decrypted; a non-aligned tail passes through
// unchanged. sample and out may be the same slice: each ciphertext block is
// fully consumed before its plaintext is written.
func (t *Template) DecryptInto(out, sample []byte) {
	head := len(sample) / 16 * 16
	if head > 0 {
		st := t.stInit
		ct := sample[:head]
		var blk [16]byte
		for bi := 0; bi < head/16; bi++ {
			processBlock(t, &st, ct, bi, &blk)
			copy(out[bi*16:bi*16+16], blk[:])
		}
	}
	copy(out[head:len(sample)], sample[head:])
}

// Decrypt decrypts one sample, returning equal-length plaintext.
func (t *Template) Decrypt(sample []byte) []byte {
	out := make([]byte, len(sample))
	t.DecryptInto(out, sample)
	return out
}
