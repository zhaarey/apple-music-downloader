package runv4

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"sync"
)


type template struct {
	ctx     []u32
	tbl     []u32
	tblOnce sync.Once
	st      St
	entry   Round1Regs

	// Values of st slots that are read every block but never written.
	c5b8, c5f0, c538, c540 u32
}

// prepare lazily builds the precomputed t4 lookup table for ctx and caches
// the read-only st constants used by decryptSample's per-block loop.
func (t *template) prepare() {
	t.tblOnce.Do(func() {
		tbl := make([]u32, len(t.ctx)/4)
		for i := range tbl {
			j := i << 2
			tbl[i] = t.ctx[j] | t.ctx[j+1]<<8 | t.ctx[j+2]<<16 | t.ctx[j+3]<<24
		}
		t.tbl = tbl
		t.c5b8 = t.st[0x5b8]
		t.c5f0 = t.st[0x5f0]
		t.c538 = t.st[0x538]
		t.c540 = t.st[0x540]
	})
}

type templateData struct {
	Ctx   string `json:"ctx"`
	State string `json:"state"`
	RCX   string `json:"rcx"`
	RAX   string `json:"rax"`
	RDX   string `json:"rdx"`
	R9    string `json:"r9"`
	RBP   string `json:"rbp"`
}

type templateResponse struct {
	Ctx   string        `json:"ctx"`
	State string        `json:"state"`
	RCX   string        `json:"rcx"`
	RAX   string        `json:"rax"`
	RDX   string        `json:"rdx"`
	R9    string        `json:"r9"`
	RBP   string        `json:"rbp"`
	Data  *templateData `json:"data"`
}

func fetchTemplate(server, adam, uri string) (*template, error) {
	// Support both old wrapper (host:port, uses /) and wrapper-lite (host:port/key).
	// If the server string contains a path (e.g. "127.0.0.1:12340/key"), use it;
	// otherwise fall back to the root path for backward compatibility.
	base := "http://" + server
	sep := "?"
	if strings.Contains(server, "/") {
		// Server already includes a path like "127.0.0.1:12340/key"
		sep = "?"
	} else {
		base += "/"
	}
	endpoint := base + sep + "adamId=" + url.QueryEscape(adam) + "&uri=" + url.QueryEscape(uri)
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("key server returned %s", resp.Status)
	}
	var respData templateResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, err
	}

	// Wrapper-lite wraps response inside "data": {"ctx": ..., "state": ...}
	// Old wrapper returns flat fields: {"ctx": ..., "state": ...}
	td := &templateData{
		Ctx:   respData.Ctx,
		State: respData.State,
		RCX:   respData.RCX,
		RAX:   respData.RAX,
		RDX:   respData.RDX,
		R9:    respData.R9,
		RBP:   respData.RBP,
	}
	if respData.Data != nil && respData.Data.Ctx != "" {
		td = respData.Data
	}

	ctxRaw, err := base64.StdEncoding.DecodeString(td.Ctx)
	if err != nil || len(ctxRaw) < 0x8000 {
		return nil, errors.New("invalid ctx in key-server response")
	}
	stateRaw, err := base64.StdEncoding.DecodeString(td.State)
	if err != nil || len(stateRaw) < 0x2000 {
		return nil, errors.New("invalid state in key-server response")
	}
	parse := func(value string) (u32, error) {
		value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
		n, err := strconv.ParseUint(value, 16, 64)
		return u32(n), err
	}
	rcx, err := parse(td.RCX)
	if err != nil {
		return nil, err
	}
	rax, err := parse(td.RAX)
	if err != nil {
		return nil, err
	}
	rdx, err := parse(td.RDX)
	if err != nil {
		return nil, err
	}
	r9, err := parse(td.R9)
	if err != nil {
		return nil, err
	}
	rbp, err := parse(td.RBP)
	if err != nil {
		return nil, err
	}

	tmpl := &template{ctx: make([]u32, len(ctxRaw))}
	for i, b := range ctxRaw {
		tmpl.ctx[i] = u32(b)
	}
	for offset := 0; offset < stSize; offset++ {
		pos := 0x2000 - offset
		tmpl.st[offset] = u32(binary.LittleEndian.Uint32(stateRaw[pos : pos+4]))
	}
	tmpl.entry = Round1Regs{rdx: rdx, rcx: rcx, rax: rax, r9: r9, rbp: rbp}
	return tmpl, nil
}

// decryptWorkspace holds buffers that are reused across samples within one
// connection, avoiding per-sample allocations in the hot path.
type decryptWorkspace struct {
	buf []byte
	out []byte
}

func decryptSample(tmpl *template, sample []byte) []byte {
	return decryptSampleInto(tmpl, sample, &decryptWorkspace{})
}

func decryptSampleInto(tmpl *template, sample []byte, ws *decryptWorkspace) []byte {
	tmpl.prepare()
	tbl := tmpl.tbl
	out := ws.out
	if cap(out) < len(sample) {
		out = make([]byte, len(sample))
	} else {
		out = out[:len(sample)]
	}
	copy(out[len(sample)/16*16:], sample[len(sample)/16*16:])
	st := tmpl.st
	for block := 0; block < len(sample)/16; block++ {
		regs := tmpl.entry
		regs.rdi = u32(0x1EB2C6B4) ^ (u32(block) << 4)
		regs.rsi = u32(8) + (u32(block) << 4)
		mid := round1Mid(tmpl.ctx, tbl, &st, sample, &regs)
		r2 := round1Tail(tmpl.ctx, tbl, &st, mid.rax, mid.r13&0xff, mid.r15&0xff, mid.r8&0xff, mid.r14&0xff)
		r2v := round2Sub6400(tmpl.ctx, tbl, &st, r2.rdi, r2.rsi, r2.rdx, r2.rcx, r2.r8, r2.r9, r2.rax, r2.rbx, r2.r10, r2.r11, r2.r13, r2.r14, r2.r15, 0)
		r8p := r2v.cp12[2] ^ tmpl.c5b8 ^ r2v.cp12[1]
		v6 := t4(tbl, 0x46a0, st[0x390]^0x2b) ^ tmpl.c5f0 ^ t4(tbl, 0x4ac0, (r8p>>24)^0x29) ^ t4(tbl, 0x2ff0, (st[0x298]>>16)^0xd6)
		v9 := t4(tbl, 0x4ac0, r2v.cp12[0]) ^ tmpl.c540 ^ t4(tbl, 0x2ff0, (r2v.v171>>16)^0x69)
		v11 := t4(tbl, 0x3950, st[0x298]^0x57) ^ tmpl.c538 ^ t4(tbl, 0x46a0, (r8p>>8)^0x2f)
		r3 := round3Sub8000(tmpl.ctx, tbl, &st, st[0x270], r2v.v189, r8p&0xff, (r8p>>16)&0xffff, st[0x280], v6, (r8p>>16)&0xff, v9, r2v.v187&0xff, v11, r2v.v189)
		base := r3.pt[0].offset
		for _, pair := range r3.pt {
			if pair.offset < base {
				base = pair.offset
			}
		}
		for _, pair := range r3.pt {
			index := pair.offset - base
			if index < 16 {
				out[block*16+int(index)] = byte(pair.value)
			}
		}
		st[0x108] = st[0x180]
		st[0x220] += 0x10
	}
	ws.out = out
	return out
}