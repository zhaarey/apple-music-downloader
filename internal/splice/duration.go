package splice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var forbiddenFilename = regexp.MustCompile(`[<>:"|?*]`)

// ParseDuration parses m:ss, h:mm:ss, or mm:ss.ms to milliseconds.
func ParseDuration(value string) int {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0
	}
	fractionalMs := 0
	if idx := strings.LastIndex(text, "."); idx >= 0 {
		frac := text[idx+1:]
		text = text[:idx]
		if f, err := strconv.ParseFloat("0."+frac, 64); err == nil {
			fractionalMs = int(f * 1000)
		}
	}
	parts := strings.Split(text, ":")
	if len(parts) == 1 {
		if n, err := strconv.Atoi(parts[0]); err == nil {
			return n*1000 + fractionalMs
		}
		return 0
	}
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0
		}
		nums = append(nums, n)
	}
	var hours, minutes, seconds int
	switch len(nums) {
	case 2:
		minutes, seconds = nums[0], nums[1]
	case 3:
		hours, minutes, seconds = nums[0], nums[1], nums[2]
	default:
		return 0
	}
	return ((hours*3600)+(minutes*60)+seconds)*1000 + fractionalMs
}

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

// SanitizeFilename makes a string safe for filenames.
func SanitizeFilename(name string) string {
	s := strings.TrimSpace(name)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = forbiddenFilename.ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.Trim(s, ". ")
}
