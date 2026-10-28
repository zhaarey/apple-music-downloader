// Package display formats Apple Music metadata for terminal output.
package display

import "strings"

const (
	explicitBadge = "[E]"
	cleanBadge    = "[C]"
	masterBadge   = "[M]"
)

// Badges returns the display-only badges for a track or album. The result has
// no trailing space and is empty when no badges apply.
func Badges(contentRating string, isAppleDigitalMaster, isMasteredForItunes bool) string {
	badges := make([]string, 0, 2)
	switch contentRating {
	case "explicit":
		badges = append(badges, explicitBadge)
	case "clean":
		badges = append(badges, cleanBadge)
	}
	if isAppleDigitalMaster || isMasteredForItunes {
		badges = append(badges, masterBadge)
	}
	return strings.Join(badges, " ")
}

// Label prefixes name with its display badges, if any.
func Label(name, contentRating string, isAppleDigitalMaster, isMasteredForItunes bool) string {
	if badges := Badges(contentRating, isAppleDigitalMaster, isMasteredForItunes); badges != "" {
		return badges + " " + name
	}
	return name
}
