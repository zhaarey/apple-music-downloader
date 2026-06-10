package platform

import (
	"os"
	"path/filepath"
)

// AppleSyncUnlockLogPath is where the Windows sync-unlock script writes details.
func AppleSyncUnlockLogPath() string {
	return filepath.Join(os.TempDir(), "aura-apple-sync-unlock.log")
}
