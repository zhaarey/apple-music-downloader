package temari

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"

	"amdl/internal/wrapper"
)

// goldenRandom is the hash of randomTemplatesHash, as produced by the
// unsimplified (transpiled 1:1) round chain.
const goldenRandom = "33e35f775608d81cb7879d073f69f3723b81fd1907b036aa5fc6e9e1bfc23dbc"

// goldenPrefetch is sha256(Decrypt(buf)) for the prefetch template with
// buf = 4101 bytes from rand.NewSource(1), as produced by the upstream Rust
// cdylib (Temari v0.5.1).
const goldenPrefetch = "d325a0e0dfb2d9bf0637bfd0b4addbcabea266708e5989cea4b30707f930e302"

func prefetchTemplate(t testing.TB) *Template {
	t.Helper()
	tmpl, err := FromJSON([]byte(wrapper.PrefetchTemplateJSON))
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func TestDecryptGolden(t *testing.T) {
	tmpl := prefetchTemplate(t)
	buf := make([]byte, 4101)
	rand.New(rand.NewSource(1)).Read(buf)
	sum := sha256.Sum256(tmpl.Decrypt(buf))
	if got := hex.EncodeToString(sum[:]); got != goldenPrefetch {
		t.Fatalf("golden mismatch: got %s", got)
	}
}

func TestDecryptTailAndInPlace(t *testing.T) {
	tmpl := prefetchTemplate(t)
	r := rand.New(rand.NewSource(2))
	for _, n := range []int{0, 1, 15, 16, 17, 33, 4096, 4111} {
		buf := make([]byte, n)
		r.Read(buf)
		out := tmpl.Decrypt(buf)
		if len(out) != n {
			t.Fatalf("len %d: got %d bytes", n, len(out))
		}
		head := n / 16 * 16
		if !bytes.Equal(out[head:], buf[head:]) {
			t.Fatalf("len %d: unaligned tail was modified", n)
		}
		inplace := append([]byte(nil), buf...)
		tmpl.DecryptInto(inplace, inplace)
		if !bytes.Equal(inplace, out) {
			t.Fatalf("len %d: in-place result differs", n)
		}
	}
}

func TestFromJSONMarshalledTemplate(t *testing.T) {
	var kt wrapper.KeyTemplate
	if err := json.Unmarshal([]byte(wrapper.PrefetchTemplateJSON), &kt); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(kt)
	a, err := FromJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	b := prefetchTemplate(t)
	if a.stInit != b.stInit || a.r1Entry != b.r1Entry || *a.ctx != *b.ctx {
		t.Fatal("templates differ")
	}
}

func TestFromJSONErrors(t *testing.T) {
	for _, body := range []string{``, `{}`, `{"ctx":"AAAA"}`, `{"ctx":"AAAA","state":"AAAA"}`, `not json`} {
		if _, err := FromJSON([]byte(body)); err == nil {
			t.Errorf("FromJSON(%q): expected error", body)
		}
	}
}

func BenchmarkDecrypt4K(b *testing.B) {
	tmpl := prefetchTemplate(b)
	buf := make([]byte, 4096)
	b.SetBytes(int64(len(buf)))
	for i := 0; i < b.N; i++ {
		tmpl.DecryptInto(buf, buf)
	}
}

// randomTemplate builds a template with random ctx, state and registers. The
// block-address state slots are kept from the real template so ciphertext
// reads stay in range.
func randomTemplate(r *rand.Rand, real *Template) *Template {
	ctx := make([]byte, CtxSize)
	r.Read(ctx)
	t := &Template{ctx: newCtxTables(ctx), r1Entry: r1Entry{r.Uint32(), r.Uint32(), r.Uint32(), r.Uint32(), r.Uint32()}}
	for i := range t.stInit {
		t.stInit[i] = r.Uint32()
	}
	for _, k := range []int{0x220, 0xb0, 0x98, 0x108, 0x180} {
		t.stInit[k] = real.stInit[k]
	}
	return t
}

// randomTemplatesHash decrypts random samples under 40 random templates and
// returns the sha256 of all plaintexts. It exercises the simplified MBA
// rewrites far beyond the single real template.
func randomTemplatesHash(t testing.TB) []byte {
	real := prefetchTemplate(t)
	r := rand.New(rand.NewSource(99))
	h := sha256.New()
	for i := 0; i < 40; i++ {
		tm := randomTemplate(r, real)
		for j := 0; j < 20; j++ {
			buf := make([]byte, 16*(1+r.Intn(300)))
			r.Read(buf)
			h.Write(tm.Decrypt(buf))
		}
	}
	return h.Sum(nil)
}

func TestDecryptRandomTemplatesGolden(t *testing.T) {
	if got := hex.EncodeToString(randomTemplatesHash(t)); got != goldenRandom {
		t.Fatalf("random-template golden mismatch: got %s", got)
	}
}
