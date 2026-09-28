package temari

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// stSlots are the 263 state slots (offsets within kdContext) carried in the
// 40020 state blob.
var stSlots = [...]uint16{
	0x30, 0x40, 0x48, 0x50, 0x58, 0x5a, 0x60, 0x68, 0x70, 0x72, 0x78, 0x80, 0x88, 0x90, 0x96,
	0x98, 0xa0, 0xb0, 0xc0, 0xc8, 0xd0, 0xd8, 0xe0, 0xe8, 0xf0, 0xf8, 0x100, 0x104, 0x108,
	0x112, 0x118, 0x120, 0x130, 0x144, 0x150, 0x152, 0x158, 0x160, 0x168, 0x170, 0x178,
	0x180, 0x188, 0x190, 0x192, 0x198, 0x1a0, 0x1a8, 0x1b0, 0x1b8, 0x1c0, 0x200, 0x208,
	0x210, 0x216, 0x218, 0x220, 0x224, 0x228, 0x230, 0x232, 0x238, 0x240, 0x248, 0x250,
	0x256, 0x258, 0x260, 0x264, 0x268, 0x270, 0x280, 0x288, 0x290, 0x298, 0x2a0, 0x2a8,
	0x2b0, 0x2b8, 0x2c0, 0x2c8, 0x2d0, 0x2d8, 0x2e0, 0x2e8, 0x2f0, 0x2f8, 0x300, 0x304,
	0x308, 0x310, 0x318, 0x320, 0x328, 0x330, 0x336, 0x338, 0x340, 0x344, 0x348, 0x350,
	0x352, 0x358, 0x360, 0x368, 0x370, 0x376, 0x378, 0x380, 0x384, 0x388, 0x390, 0x392,
	0x398, 0x3b0, 0x3b8, 0x3c0, 0x3c8, 0x3d0, 0x3e0, 0x3e8, 0x3f8, 0x400, 0x408, 0x410,
	0x416, 0x418, 0x420, 0x424, 0x428, 0x430, 0x432, 0x440, 0x448, 0x450, 0x460, 0x468,
	0x470, 0x478, 0x480, 0x488, 0x490, 0x4d8, 0x4e0, 0x4e8, 0x4f0, 0x4f8, 0x500, 0x508,
	0x510, 0x512, 0x518, 0x520, 0x528, 0x530, 0x536, 0x538, 0x540, 0x548, 0x550, 0x552,
	0x558, 0x560, 0x568, 0x570, 0x576, 0x578, 0x580, 0x584, 0x588, 0x590, 0x592, 0x598,
	0x5a0, 0x5a8, 0x5b0, 0x5b8, 0x5c0, 0x5c8, 0x5d8, 0x5e8, 0x5f0, 0x5f8, 0x600, 0x608,
	0x610, 0x616, 0x618, 0x620, 0x624, 0x628, 0x638, 0x640, 0x648, 0x656, 0x658, 0x660,
	0x664, 0x668, 0x670, 0x672, 0x678, 0x680, 0x688, 0x690, 0x696, 0x698, 0x6a8, 0x6b0,
	0x6b8, 0x6c0, 0x6c8, 0x6d8, 0x6e0, 0x6e8, 0x6f0, 0x6f8, 0x700, 0x708, 0x720, 0x728,
	0x736, 0x740, 0x752, 0x760, 0x768, 0x770, 0x778, 0x780, 0x792, 0x816, 0x824, 0x872,
	0x912, 0x944, 0x952, 0x960, 0x992, 0x1040, 0x1088, 0x1328, 0x1360, 0x1368, 0x1384,
	0x1408, 0x1416, 0x1424, 0x1432, 0x1440, 0x1448, 0x1456, 0x1480, 0x1512, 0x1528,
	0x1536, 0x1552, 0x1560, 0x1568, 0x1576, 0x1632, 0x1704, 0x1720, 0x1760,
}

// Registers holds the round-1 entry registers of a template, as hex strings
// ("0x..."). Empty fields default to 0.
type Registers struct {
	RCX, RAX, RDX, R9, RBP string
}

// FromJSON builds a Template from a 40020-style key-server JSON response body:
//
//	{ "ctx": "<b64>", "state": "<b64>",
//	  "rcx": "0x..", "rax": "0x..", "rdx": "0x..", "r9": "0x..", "rbp": "0x.." }
func FromJSON(body []byte) (*Template, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, errors.New("temari: empty JSON template")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("temari: parse JSON template: %w", err)
	}
	field := func(k string) (string, bool) {
		v, ok := raw[k]
		if !ok {
			return "", false
		}
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s, true
		}
		return string(v), true // bare value (number / null)
	}
	ctx, ok := field("ctx")
	if !ok {
		return nil, errors.New("temari: no ctx field")
	}
	state, ok := field("state")
	if !ok {
		return nil, errors.New("temari: no state field")
	}
	var regs Registers
	regs.RCX, _ = field("rcx")
	regs.RAX, _ = field("rax")
	regs.RDX, _ = field("rdx")
	regs.R9, _ = field("r9")
	regs.RBP, _ = field("rbp")
	return New(ctx, state, regs)
}

// New builds a Template from base64 ctx / state blobs and entry registers.
func New(ctxB64, stateB64 string, regs Registers) (*Template, error) {
	var e r1Entry
	for _, r := range []struct {
		dst *uint32
		s   string
	}{
		{&e.rcx, regs.RCX}, {&e.rax, regs.RAX}, {&e.rdx, regs.RDX}, {&e.r9, regs.R9}, {&e.rbp, regs.RBP},
	} {
		v, err := parseHex(r.s)
		if err != nil {
			return nil, err
		}
		*r.dst = v
	}

	ctx := base64Decode(ctxB64)
	if len(ctx) < CtxSize {
		return nil, fmt.Errorf("temari: ctx too short: %d", len(ctx))
	}
	stRaw := base64Decode(stateB64)
	if len(stRaw) < 0x2000 {
		return nil, fmt.Errorf("temari: state too short: %d", len(stRaw))
	}
	t := &Template{ctx: newCtxTables(ctx[:CtxSize]), r1Entry: e}
	for _, off := range stSlots {
		o := int(off)
		if o >= StSize {
			continue // present in the blob but never read by the round chain
		}
		if pos := 0x2000 - o; pos+4 <= len(stRaw) {
			t.stInit[o] = binary.LittleEndian.Uint32(stRaw[pos:])
		}
	}
	return t, nil
}

// parseHex parses "0x..." (prefix optional). r9/rbp are 64-bit pointers in the
// 40020 response; only the low 32 bits matter.
func parseHex(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	h := strings.TrimPrefix(s, "0x")
	v, err := strconv.ParseUint(h, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("temari: bad hex %s: %w", h, err)
	}
	return uint32(v), nil
}

// base64Decode is a lenient standard-alphabet decoder matching upstream:
// it stops at the first '=' and skips any non-alphabet byte.
func base64Decode(s string) []byte {
	out := make([]byte, 0, len(s)*3/4)
	var buf uint32
	bits := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var v uint32
		switch {
		case c == '=':
			return out
		case c >= 'A' && c <= 'Z':
			v = uint32(c - 'A')
		case c >= 'a' && c <= 'z':
			v = uint32(c-'a') + 26
		case c >= '0' && c <= '9':
			v = uint32(c-'0') + 52
		case c == '+':
			v = 62
		case c == '/':
			v = 63
		default:
			continue
		}
		buf = buf<<6 | v
		bits += 6
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(buf>>bits))
		}
	}
	return out
}
