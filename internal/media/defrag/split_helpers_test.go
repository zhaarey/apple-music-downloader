package fmp4unfrag_test

import (
	"io"
	"os"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
)

// readTopLevel walks top-level boxes itself, because mp4ff's DecodeFile
// rejects progressive files with more than one non-empty mdat.
func readTopLevel(t *testing.T, file *os.File) ([]string, *mp4.MoovBox) {
	t.Helper()
	var types []string
	var moov *mp4.MoovBox
	var pos uint64
	for {
		if _, err := file.Seek(int64(pos), io.SeekStart); err != nil {
			t.Fatal(err)
		}
		hdr, err := mp4.DecodeHeader(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		types = append(types, hdr.Name)
		if hdr.Name == "moov" {
			if _, err := file.Seek(int64(pos), io.SeekStart); err != nil {
				t.Fatal(err)
			}
			box, err := mp4.DecodeBox(pos, file)
			if err != nil {
				t.Fatal(err)
			}
			moov = box.(*mp4.MoovBox)
		}
		pos += hdr.Size
	}
	if moov == nil {
		t.Fatal("no moov box")
	}
	return types, moov
}
