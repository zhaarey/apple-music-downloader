package platform

import (
	"os"
	"path/filepath"
)

// SyncRepairLogPath returns where elevated repair scripts write logs.
func SyncRepairLogPath() string {
	return filepath.Join(os.TempDir(), "aura-sync-repair.log")
}
