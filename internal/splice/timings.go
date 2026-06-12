package splice

const (
	SnapMs     = 10
	MinTrackMs = 1000
)

// ComputeTrackTimings returns (start_ms, end_ms, duration_ms) per track.
func ComputeTrackTimings(tracks []Track, masterDurationMs int) [][3]int {
	if len(tracks) == 0 {
		return nil
	}
	starts := resolveStarts(tracks, masterDurationMs)
	result := make([][3]int, 0, len(tracks))
	for i := range tracks {
		start := starts[i]
		end := 0
		if i < len(tracks)-1 {
			end = starts[i+1]
		} else if masterDurationMs > 0 {
			end = masterDurationMs
		} else {
			end = start + tracks[i].DurationMs
		}
		duration := end - start
		if duration < 0 {
			duration = 0
		}
		result = append(result, [3]int{start, end, duration})
	}
	return result
}

func resolveStarts(tracks []Track, masterDurationMs int) []int {
	n := len(tracks)
	starts := make([]int, n)
	starts[0] = 0
	for i := 1; i < n; i++ {
		if tracks[i].StartMs != nil {
			starts[i] = maxInt(0, *tracks[i].StartMs)
		} else {
			starts[i] = starts[i-1] + tracks[i-1].DurationMs
		}
	}
	if masterDurationMs > 0 {
		for i := 1; i < n; i++ {
			maxPos := masterDurationMs - MinTrackMs*(n-i)
			if starts[i] > maxPos {
				starts[i] = maxPos
			}
			minPos := starts[i-1] + MinTrackMs
			if starts[i] < minPos {
				starts[i] = minPos
			}
		}
	}
	return starts
}

// RebuildStartsFromDurations resets start_ms from cumulative durations.
func RebuildStartsFromDurations(tracks []Track, masterDurationMs int) {
	if len(tracks) == 0 {
		return
	}
	zero := 0
	tracks[0].StartMs = &zero
	current := tracks[0].DurationMs
	for i := 1; i < len(tracks); i++ {
		s := current
		tracks[i].StartMs = &s
		if i < len(tracks)-1 {
			current += tracks[i].DurationMs
		} else if masterDurationMs > 0 {
			lastDur := masterDurationMs - s
			if lastDur < MinTrackMs {
				lastDur = MinTrackMs
			}
			tracks[i].DurationMs = lastDur
		}
	}
}

// ApplyTimingsToTracks writes resolved start/duration back onto tracks.
func ApplyTimingsToTracks(tracks []Track, timings [][3]int) {
	for i := range tracks {
		if i >= len(timings) {
			break
		}
		start := timings[i][0]
		duration := timings[i][2]
		tracks[i].StartMs = &start
		tracks[i].DurationMs = duration
	}
}

// SetBoundary moves the cut between track boundaryIndex-1 and boundaryIndex.
func SetBoundary(tracks []Track, boundaryIndex, positionMs, masterDurationMs int, snapMs *int) {
	if boundaryIndex <= 0 || boundaryIndex >= len(tracks) {
		return
	}
	positionMs = snap(positionMs, snapMs)
	prevStart := 0
	if tracks[boundaryIndex-1].StartMs != nil {
		prevStart = *tracks[boundaryIndex-1].StartMs
	}
	minPos := prevStart + MinTrackMs
	maxPos := positionMs
	if boundaryIndex < len(tracks)-1 {
		if tracks[boundaryIndex+1].StartMs != nil {
			maxPos = *tracks[boundaryIndex+1].StartMs - MinTrackMs
		} else {
			maxPos = masterDurationMs - MinTrackMs*(len(tracks)-boundaryIndex-1)
		}
	} else if masterDurationMs > 0 {
		maxPos = masterDurationMs - MinTrackMs
	}
	if positionMs < minPos {
		positionMs = minPos
	}
	if positionMs > maxPos {
		positionMs = maxPos
	}
	tracks[boundaryIndex].StartMs = &positionMs
	timings := ComputeTrackTimings(tracks, masterDurationMs)
	ApplyTimingsToTracks(tracks, timings)
}

// SetTrackStart moves the start of track row (same as SetBoundary for row > 0).
func SetTrackStart(tracks []Track, row, startMs, masterDurationMs int, snapMs *int) {
	if row == 0 && len(tracks) > 0 {
		zero := 0
		tracks[0].StartMs = &zero
		return
	}
	SetBoundary(tracks, row, startMs, masterDurationMs, snapMs)
}

// SetTrackDuration updates duration for a row and shifts the next boundary.
func SetTrackDuration(tracks []Track, row, durationMs, masterDurationMs int) {
	if row < 0 || row >= len(tracks) {
		return
	}
	if durationMs < MinTrackMs {
		durationMs = MinTrackMs
	}
	tracks[row].DurationMs = durationMs
	if row+1 < len(tracks) {
		start := 0
		if tracks[row].StartMs != nil {
			start = *tracks[row].StartMs
		}
		next := start + tracks[row].DurationMs
		tracks[row+1].StartMs = &next
	}
	timings := ComputeTrackTimings(tracks, masterDurationMs)
	ApplyTimingsToTracks(tracks, timings)
}

// DistributeDrift spreads timing gap between sum(durations) and master across tracks.
func DistributeDrift(tracks []Track, masterDurationMs int) int {
	if len(tracks) == 0 || masterDurationMs <= 0 {
		return 0
	}
	timings := ComputeTrackTimings(tracks, masterDurationMs)
	total := 0
	for _, t := range timings {
		total += t[2]
	}
	gap := masterDurationMs - total
	if absInt(gap) <= SnapMs {
		RebuildStartsFromDurations(tracks, masterDurationMs)
		timings = ComputeTrackTimings(tracks, masterDurationMs)
		ApplyTimingsToTracks(tracks, timings)
		return gap
	}
	if total <= 0 {
		return 0
	}
	remaining := gap
	for i := range tracks {
		if i == len(tracks)-1 {
			tracks[i].DurationMs = maxInt(MinTrackMs, tracks[i].DurationMs+remaining)
		} else {
			share := int(float64(gap) * (float64(tracks[i].DurationMs) / float64(total)))
			tracks[i].DurationMs = maxInt(MinTrackMs, tracks[i].DurationMs+share)
			remaining -= share
		}
	}
	RebuildStartsFromDurations(tracks, masterDurationMs)
	timings = ComputeTrackTimings(tracks, masterDurationMs)
	ApplyTimingsToTracks(tracks, timings)
	return gap
}

func snap(ms int, snapMs *int) int {
	s := SnapMs
	if snapMs != nil {
		s = *snapMs
	}
	if s <= 0 {
		return ms
	}
	return ((ms + s/2) / s) * s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
