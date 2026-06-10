package platform

import (
	"os"
	"path/filepath"
)

// ApplePurgeLogPath is where the Windows deep-purge script writes details.
func ApplePurgeLogPath() string {
	return filepath.Join(os.TempDir(), "aura-apple-purge.log")
}
