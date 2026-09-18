package fmp4unfrag

import (
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
)

func TestCollectSampleGroupRunsIgnoresRollGroup(t *testing.T) {
	traf := &mp4.TrafBox{
		Sbgp: &mp4.SbgpBox{GroupingType: "roll"},
	}
	td := &trackData{}

	if err := collectSampleGroupRuns(td, traf, 0); err != nil {
		t.Fatal(err)
	}
	if len(td.SampleGroups) != 0 {
		t.Fatalf("expected roll group to be ignored, got %d runs", len(td.SampleGroups))
	}
}

func TestMakeTagCompatibleFtyp(t *testing.T) {
	original := mp4.NewFtyp("isom", 1, []string{"isom", "iso5", "hlsf", "cmfc", "ccea"})
	ftyp, err := makeTagCompatibleFtyp(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ftyp.MajorBrand() != fixedFtypMajorBrand {
		t.Fatalf("expected major brand %q, got %q", fixedFtypMajorBrand, ftyp.MajorBrand())
	}
	if ftyp.MinorVersion() != fixedFtypMinorVersion {
		t.Fatalf("expected minor version %d, got %d", fixedFtypMinorVersion, ftyp.MinorVersion())
	}

	disallowed := map[string]bool{
		"cmfc": true,
		"ccea": true,
		"hlsf": true,
	}
	for _, b := range ftyp.CompatibleBrands() {
		if disallowed[b] {
			t.Errorf("compatible brands must not contain fragmented/CMAF brand %q", b)
		}
	}

	expectedBrands := map[string]bool{
		"M4A ": false,
		"mp42": false,
		"isom": false,
	}
	for _, b := range ftyp.CompatibleBrands() {
		if _, ok := expectedBrands[b]; ok {
			expectedBrands[b] = true
		}
	}
	for brand, found := range expectedBrands {
		if !found {
			t.Errorf("expected compatible brand %q to be present", brand)
		}
	}
}

func TestMakeTagCompatibleFtypNil(t *testing.T) {
	_, err := makeTagCompatibleFtyp(nil)
	if err == nil {
		t.Fatal("expected error when original ftyp is nil, got nil")
	}
}

