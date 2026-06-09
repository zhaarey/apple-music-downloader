package media

import (
	"testing"
)

func TestValidateIPhoneSync_missingFile(t *testing.T) {
	res, err := ValidateIPhoneSync("", "C:\\does-not-exist\\track.m4a")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if res.Ready {
		t.Fatal("expected not ready")
	}
}

func TestGetAppleMusicCacheInfo(t *testing.T) {
	info := GetAppleMusicCacheInfo()
	if info.Note == "" {
		t.Fatal("expected cache note")
	}
	if info.Platform == "" {
		t.Fatal("expected platform")
	}
}
