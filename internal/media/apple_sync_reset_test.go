package media

import (
	"runtime"
	"testing"
)

func TestResetAppleSyncStack_nonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows-only negative path")
	}
	res := ResetAppleSyncStack(false)
	if res.OK {
		t.Fatal("expected not ok on non-windows")
	}
	if res.Summary == "" {
		t.Fatal("expected summary")
	}
}
