package widevinerip

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itouakirai/mp4ff/mp4"
)

// decryptMP4InMemory is the previous whole-file implementation, kept as a
// reference for the streaming DecryptMP4.
func decryptMP4InMemory(t *testing.T, data []byte, key []byte) []byte {
	t.Helper()
	encrypted, err := mp4.DecodeFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	decryptInfo, err := mp4.DecryptInit(encrypted.Init)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := encrypted.Init.Encode(&out); err != nil {
		t.Fatal(err)
	}
	for _, segment := range encrypted.Segments {
		if err := mp4.DecryptSegment(segment, decryptInfo, key); err != nil && err.Error() != errNoSencBox {
			t.Fatal(err)
		}
		if err := segment.Encode(&out); err != nil {
			t.Fatal(err)
		}
	}
	return out.Bytes()
}

func mp4ffTestdata(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/itouakirai/mp4ff").Output()
	if err != nil {
		t.Skipf("locate mp4ff module: %v", err)
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), "mp4", "testdata")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("mp4ff testdata unavailable: %v", err)
	}
	return dir
}

func TestDecryptMP4MatchesInMemoryDecrypt(t *testing.T) {
	dir := mp4ffTestdata(t)
	cases := []struct {
		file string
		key  string
	}{
		{"prog_8s_enc_dashinit.mp4", "63cb5f7184dd4b689a5c5ff11ee6a328"},
		{"cbcs.mp4", "22bdb0063805260307ee5045c0f3835a"},
		{"cbcs_audio.mp4", "5ffd93861fa776e96cccd934898fc1c8"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			key, err := hex.DecodeString(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			want := decryptMP4InMemory(t, data, key)

			var got bytes.Buffer
			if err := DecryptMP4(bytes.NewReader(data), key, &got); err != nil {
				t.Fatalf("DecryptMP4: %v", err)
			}
			if !bytes.Equal(got.Bytes(), want) {
				t.Fatalf("streaming output differs from in-memory output (%d vs %d bytes)", got.Len(), len(want))
			}
		})
	}
}

func TestDecryptMP4RejectsTruncatedInput(t *testing.T) {
	dir := mp4ffTestdata(t)
	data, err := os.ReadFile(filepath.Join(dir, "cbcs.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	key, _ := hex.DecodeString("22bdb0063805260307ee5045c0f3835a")
	if err := DecryptMP4(bytes.NewReader(data[:len(data)-100]), key, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for truncated input")
	}
}
