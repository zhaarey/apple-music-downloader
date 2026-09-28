package version

import (
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v1.2.0", "v1.1.9", 1},
		{"v2.0.0", "v1.99.99", 1},
		{"v1.0", "v1.0.0", 0},
		{"v1.0.0-beta.1", "v1.0.0", 0},
		{"dev", "v1.0.0", -1},
		{"v1.0.0", "dev", 1},
	}

	for _, tt := range tests {
		result := Compare(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("Compare(%q, %q) = %d; want %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}

func TestIsDev(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()

	Version = "dev"
	if !IsDev() {
		t.Errorf("expected true for 'dev'")
	}

	Version = "v1.2.3"
	if IsDev() {
		t.Errorf("expected false for 'v1.2.3'")
	}
}
