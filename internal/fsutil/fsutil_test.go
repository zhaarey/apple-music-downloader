package fsutil

import (
	"os"
	"runtime"
	"testing"
)

func TestCreateOutputTempPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	f, err := CreateOutputTemp(t.TempDir(), "x-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != OutputPerm {
		t.Fatalf("mode = %o, want %o", got, OutputPerm)
	}
	if _, err := os.Stat(f.Name()); err != nil {
		t.Fatal(err)
	}
}
