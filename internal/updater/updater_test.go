package updater

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello world amdl update")
	// sha256 of "hello world amdl update":
	// a7b9193796d13ec813b28b2bebb0bbd5d14dfb0ad4b78ae6eb1cfc9fa6a8b14a
	checksums := `
# test checksums
c1f1e6ce997ad8b870f0a26beebb4f546f6e7f12c8b849c10ff52b54702b0143  amdl_windows_amd64.exe
1111111111111111111111111111111111111111111111111111111111111111  amdl_linux_amd64
`
	err := VerifyChecksum(data, "amdl_windows_amd64.exe", checksums)
	if err != nil {
		t.Fatalf("expected valid checksum, got: %v", err)
	}

	err = VerifyChecksum([]byte("corrupt data"), "amdl_windows_amd64.exe", checksums)
	if err == nil {
		t.Fatalf("expected error on mismatch checksum, got nil")
	}

	err = VerifyChecksum(data, "amdl_darwin_arm64", checksums)
	if err == nil {
		t.Fatalf("expected error for missing asset in checksums, got nil")
	}
}

func TestFindBinaryAsset(t *testing.T) {
	rel := &Release{
		TagName: "v1.2.0",
		Assets: []Asset{
			{Name: "amdl_windows_amd64.exe", BrowserDownloadURL: "http://example.com/win"},
			{Name: "amdl_linux_amd64", BrowserDownloadURL: "http://example.com/linux"},
			{Name: "amdl_darwin_arm64", BrowserDownloadURL: "http://example.com/mac"},
			{Name: "checksums.txt", BrowserDownloadURL: "http://example.com/sums"},
		},
	}

	asset, err := FindBinaryAsset(rel, "windows", "amd64")
	if err != nil || asset.Name != "amdl_windows_amd64.exe" {
		t.Fatalf("failed to find windows asset: %v", err)
	}

	asset, err = FindBinaryAsset(rel, "linux", "amd64")
	if err != nil || asset.Name != "amdl_linux_amd64" {
		t.Fatalf("failed to find linux asset: %v", err)
	}

	sums, err := FindChecksumsAsset(rel)
	if err != nil || sums.Name != "checksums.txt" {
		t.Fatalf("failed to find checksums: %v", err)
	}
}

func TestInsertOptionIntoYAML(t *testing.T) {
	initialYAML := `general:
  lite-server: "http://127.0.0.1:12340"
  storefront: "us"

media:
  alac-max: 192000
`

	updated := InsertOptionIntoYAML(initialYAML, "general", `  exit-on-error: true   # Auto exit on error`)
	if !strings.Contains(updated, "exit-on-error: true") {
		t.Fatalf("expected exit-on-error to be inserted in general, got:\n%s", updated)
	}

	// Insert into non-existent section
	updated2 := InsertOptionIntoYAML(updated, "convert", `  after-download: true`)
	if !strings.Contains(updated2, "convert:") || !strings.Contains(updated2, "after-download: true") {
		t.Fatalf("expected new convert section, got:\n%s", updated2)
	}
}

func TestFindMissingOptionsAndBackup(t *testing.T) {
	tmpDir := t.TempDir()
	userConfPath := filepath.Join(tmpDir, "config.yaml")

	userConf := `general:
  lite-server: "http://127.0.0.1:12340"
`
	exampleConf := `general:
  lite-server: "http://127.0.0.1:12340"   # Lite server
  storefront: "us"                        # Default storefront

media:
  alac-max: 192000                        # Max ALAC
`

	if err := os.WriteFile(userConfPath, []byte(userConf), 0644); err != nil {
		t.Fatal(err)
	}

	missing, err := FindMissingOptions(userConfPath, exampleConf)
	if err != nil {
		t.Fatalf("FindMissingOptions failed: %v", err)
	}

	if len(missing) != 2 {
		t.Fatalf("expected 2 missing options (general.storefront, media.alac-max), got %d", len(missing))
	}

	backupPath, err := BackupConfigFile(userConfPath, "v1.2.0")
	if err != nil {
		t.Fatalf("BackupConfigFile failed: %v", err)
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("backup file %s does not exist", backupPath)
	}
}
