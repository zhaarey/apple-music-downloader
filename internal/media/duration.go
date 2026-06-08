package media

import "fmt"

// FormatDuration formats milliseconds as m:ss or h:mm:ss.
func FormatDuration(ms int, includeHours bool) string {
	if ms < 0 {
		ms = 0
	}
	totalSec := ms / 1000
	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60
	if hours > 0 || includeHours {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
