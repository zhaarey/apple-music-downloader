package media

import (
	"runtime"
	"testing"
)

func TestReleaseAppleSyncLock_nonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows-only negative path")
	}
	res := ReleaseAppleSyncLock(false, false)
	if res.OK {
		t.Fatal("expected not ok on non-windows")
	}
	if res.Summary == "" {
		t.Fatal("expected summary")
	}
}
