package version

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

var (
	// Version is the current application version (injected at build time via -ldflags).
	Version = "dev"
	// Commit is the git commit hash (injected at build time via -ldflags).
	Commit = "none"
	// BuildDate is the build timestamp (injected at build time via -ldflags).
	BuildDate = "unknown"
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf("amdl %s (commit: %s, built at: %s, %s/%s, %s)",
		Version, Commit, BuildDate, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

// IsDev reports whether the running binary was built from source/dev mode.
func IsDev() bool {
	return Version == "" || Version == "dev" || strings.HasPrefix(Version, "dev-")
}

// Compare compares two semantic version strings (e.g. "v1.2.3" vs "v1.3.0").
// Returns:
//   -1 if v1 < v2
//    0 if v1 == v2
//    1 if v1 > v2
// If either version is invalid or "dev", standard comparison handles it gracefully.
func Compare(v1, v2 string) int {
	v1 = strings.TrimPrefix(strings.TrimSpace(v1), "v")
	v2 = strings.TrimPrefix(strings.TrimSpace(v2), "v")

	if v1 == v2 {
		return 0
	}
	if v1 == "dev" || v1 == "" {
		return -1 // dev is treated as older than any tagged release
	}
	if v2 == "dev" || v2 == "" {
		return 1
	}

	parts1 := splitVersion(v1)
	parts2 := splitVersion(v2)

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		p1 := 0
		if i < len(parts1) {
			p1 = parts1[i]
		}
		p2 := 0
		if i < len(parts2) {
			p2 = parts2[i]
		}
		if p1 < p2 {
			return -1
		}
		if p1 > p2 {
			return 1
		}
	}

	return 0
}

func splitVersion(v string) []int {
	// Strip any prerelease or build metadata (e.g. -beta.1, +build123)
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	rawParts := strings.Split(v, ".")
	var parts []int
	for _, p := range rawParts {
		num, err := strconv.Atoi(p)
		if err != nil {
			num = 0
		}
		parts = append(parts, num)
	}
	return parts
}
