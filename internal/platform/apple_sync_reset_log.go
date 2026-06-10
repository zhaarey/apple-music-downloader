package platform

import (
	"os"
	"path/filepath"
)

// AppleSyncResetLogPath is where the Windows sync-reset script writes details.
func AppleSyncResetLogPath() string {
	return filepath.Join(os.TempDir(), "aura-apple-sync-reset.log")
}

// AppleSyncUnlockLogPath is deprecated; kept for older log opens.
func AppleSyncUnlockLogPath() string {
	return AppleSyncResetLogPath()
}
