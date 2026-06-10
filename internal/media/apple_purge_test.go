package media

import (
	"runtime"
	"testing"
)

func TestRunAppleMusicDeepPurge_nonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows-only negative test")
	}
	res := RunAppleMusicDeepPurge(false)
	if res.OK {
		t.Fatal("expected not ok on non-windows")
	}
	if len(res.ManualSteps) == 0 {
		t.Fatal("expected manual steps")
	}
}
