package splice

import "testing"

func TestParseDuration(t *testing.T) {
	cases := map[string]int{
		"1:21":      81000,
		"2:40":      160000,
		"0:58":      58000,
		"1:42:15":   6135000,
		"2:40.5":    160500,
	}
	for in, want := range cases {
		if got := ParseDuration(in); got != want {
			t.Fatalf("ParseDuration(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestComputeTrackTimingsSequential(t *testing.T) {
	tracks := []Track{
		{Title: "A", DurationMs: 81000},
		{Title: "B", DurationMs: 160000},
		{Title: "C", DurationMs: 95000},
	}
	master := 81000 + 160000 + 95000
	timings := ComputeTrackTimings(tracks, master)
	if len(timings) != 3 {
		t.Fatalf("expected 3 timings, got %d", len(timings))
	}
	if timings[0][0] != 0 || timings[0][1] != 81000 {
		t.Fatalf("track 0 boundary mismatch: %v", timings[0])
	}
	if timings[2][1] != master {
		t.Fatalf("last track should end at master duration")
	}
}

func TestDistributeDrift(t *testing.T) {
	tracks := []Track{
		{Title: "A", DurationMs: 60000},
		{Title: "B", DurationMs: 60000},
	}
	master := 125000
	DistributeDrift(tracks, master)
	timings := ComputeTrackTimings(tracks, master)
	total := 0
	for _, tm := range timings {
		total += tm[2]
	}
	if total != master {
		t.Fatalf("after drift total=%d master=%d", total, master)
	}
}
