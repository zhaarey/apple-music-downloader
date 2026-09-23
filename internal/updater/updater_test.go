package updater

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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
			{Name: "amdl_android_arm64", BrowserDownloadURL: "http://example.com/android"},
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

	asset, err = FindBinaryAsset(rel, "android", "arm64")
	if err != nil || asset.Name != "amdl_android_arm64" {
		t.Fatalf("failed to find android asset: %v", err)
	}

	sums, err := FindChecksumsAsset(rel)
	if err != nil || sums.Name != "checksums.txt" {
		t.Fatalf("failed to find checksums: %v", err)
	}
}

func TestInsertOptionNested(t *testing.T) {
	input := `media:
  alac-max: 192000
  mv:
    max: 2160

# ------
# 3. Paths
# ------
paths:
  alac: "AM"
`
	want := `media:
  alac-max: 192000
  mv:
    max: 2160
    audio-type: "atmos"

# ------
# 3. Paths
# ------
paths:
  alac: "AM"
`
	if got := InsertOption(input, []string{"media", "mv", "audio-type"}, `"atmos"`, ""); got != want {
		t.Fatalf("existing nested parent: got:\n%s\nwant:\n%s", got, want)
	}

	// Missing intermediate mapping is created instead of a flat `mv.audio-type` key.
	input2 := "media:\n  alac-max: 192000\n\npaths:\n  alac: \"AM\"\n"
	want2 := "media:\n  alac-max: 192000\n  mv:\n    audio-type: \"atmos\"\n\npaths:\n  alac: \"AM\"\n"
	if got := InsertOption(input2, []string{"media", "mv", "audio-type"}, `"atmos"`, ""); got != want2 {
		t.Fatalf("missing intermediate parent: got:\n%s\nwant:\n%s", got, want2)
	}

	// Missing top-level section is appended.
	want3 := input2 + "\nconvert:\n  after-download: false\n"
	if got := InsertOption(input2, []string{"convert", "after-download"}, "false", ""); got != want3 {
		t.Fatalf("missing section: got:\n%s\nwant:\n%s", got, want3)
	}
}

func TestInsertOptionFollowsIndentAndAlignsComment(t *testing.T) {
	input := "general:\n    proxy: \"\"\nmedia:\n    alac-max: 1\n"
	got := InsertOption(input, []string{"media", "mv", "max"}, "2160", "")
	if !strings.Contains(got, "\n    mv:\n        max: 2160\n") {
		t.Fatalf("expected 4-space indentation to be followed, got:\n%s", got)
	}

	got = InsertOption(input, []string{"general", "exit-on-error"}, "false", "Exit on errors")
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "exit-on-error") {
			if idx := strings.Index(line, "#"); idx != commentColumn {
				t.Fatalf("expected comment at column %d, got %d in %q", commentColumn, idx, line)
			}
			return
		}
	}
	t.Fatalf("exit-on-error not inserted:\n%s", got)
}

func TestScanYAMLKeysQuotedHash(t *testing.T) {
	lines := splitLines("paths:\n  explicit: \"#E\"   # tag suffix\n  clean: 'a # b'\n")
	keys := scanYAMLKeys(lines)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[1].value != `"#E"` || keys[1].comment != "tag suffix" {
		t.Fatalf("unexpected split: value=%q comment=%q", keys[1].value, keys[1].comment)
	}
	if keys[2].value != `'a # b'` || keys[2].comment != "" {
		t.Fatalf("unexpected split: value=%q comment=%q", keys[2].value, keys[2].comment)
	}
}

func TestParseCustomValue(t *testing.T) {
	tests := []struct {
		input   string
		def     any
		want    string
		wantErr bool
	}{
		{"jp", "us", `"jp"`, false},
		{"true", "x", `"true"`, false}, // stays a string
		{`"a\"b"`, "x", `"a\"b"`, false},
		{"'it''s'", "x", `"it's"`, false},
		{"TRUE", false, "true", false},
		{"yes", false, "", true},
		{"96000", 192000, "96000", false},
		{"96k", 192000, "", true},
		{"a: b", 1, "", true},
	}
	for _, tt := range tests {
		got, _, err := parseCustomValue(tt.input, tt.def)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("parseCustomValue(%q, %T) = %q, %v; want %q, err=%v", tt.input, tt.def, got, err, tt.want, tt.wantErr)
		}
	}
}

// removeKeys drops the lines of the given keys (and anything nested under them).
func removeKeys(content string, paths ...string) string {
	lines := splitLines(content)
	drop := make(map[int]bool)
	for _, kl := range scanYAMLKeys(lines) {
		for _, p := range paths {
			if strings.Join(kl.path, ".") != p {
				continue
			}
			drop[kl.line] = true
			for j := kl.line + 1; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if t == "" || strings.HasPrefix(t, "#") {
					continue
				}
				if len(lines[j])-len(strings.TrimLeft(lines[j], " ")) <= kl.indent {
					break
				}
				drop[j] = true
			}
		}
	}
	var kept []string
	for i, l := range lines {
		if !drop[i] {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n") + "\n"
}

func TestRunConfigMigrationGuideRealExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "config.yaml.example"))
	if err != nil {
		t.Fatal(err)
	}
	example := string(data)

	user := removeKeys(example, "media.alac-fix", "media.mv", "metadata.lyrics.format", "convert")
	user = strings.Replace(user, `storefront: "us"`, `storefront: "jp"`, 1)
	user = strings.ReplaceAll(user, "\n", "\r\n") // Windows-edited config

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(user), 0644); err != nil {
		t.Fatal(err)
	}

	missing, err := FindMissingOptions(configPath, example)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, opt := range missing {
		// "format" also exists under artwork; the comment must come from lyrics.
		if opt.Key == "metadata.lyrics.format" {
			found = true
			if !strings.Contains(opt.Comment, "Lyrics file format") {
				t.Fatalf("wrong comment for %s: %q", opt.Key, opt.Comment)
			}
		}
	}
	if !found {
		t.Fatalf("metadata.lyrics.format not reported missing")
	}

	changed, err := RunConfigMigrationGuide(configPath, example, "v9.9.9", true)
	if err != nil || !changed {
		t.Fatalf("migration failed: changed=%v err=%v", changed, err)
	}

	out, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(out), "\n") != strings.Count(string(out), "\r\n") {
		t.Fatalf("CRLF line endings not preserved")
	}

	gotK, err := parseYAML(normalizeNewlines(string(out)))
	if err != nil {
		t.Fatalf("migrated config does not parse: %v\n%s", err, out)
	}
	wantK, _ := parseYAML(example)
	for _, k := range wantK.Keys() {
		want := wantK.Get(k)
		if k == "general.storefront" {
			want = "jp"
		}
		if got := gotK.Get(k); !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", k, got, want)
		}
	}
	if len(gotK.Keys()) != len(wantK.Keys()) {
		t.Errorf("got %d keys, want %d", len(gotK.Keys()), len(wantK.Keys()))
	}
	if strings.Contains(string(out), "mv.audio-type") {
		t.Errorf("expected nested mv mapping, found flat dotted key:\n%s", out)
	}

	missing, err = FindMissingOptions(configPath, example)
	if err != nil || len(missing) != 0 {
		t.Fatalf("expected no missing options after migration, got %d (%v)", len(missing), err)
	}
}

func TestRunConfigMigrationGuideMissingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	changed, err := RunConfigMigrationGuide(configPath, "general:\n  proxy: \"\"\n", "v1.0.0", true)
	if err != nil || changed {
		t.Fatalf("expected no-op, got changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("migration must not create %s", configPath)
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

func TestFindBinaryAssetExactOnly(t *testing.T) {
	rel := &Release{
		TagName: "v1.2.0",
		Assets: []Asset{
			{Name: "amdl_linux_arm64"},
			{Name: "amdl_windows_arm64.exe"},
			{Name: "checksums.txt"},
		},
	}

	// 32-bit arm must not pick the arm64 build via substring match.
	if a, err := FindBinaryAsset(rel, "linux", "arm"); err == nil {
		t.Fatalf("expected no asset for linux/arm, got %s", a.Name)
	}
	if a, err := FindBinaryAsset(rel, "windows", "arm"); err == nil {
		t.Fatalf("expected no asset for windows/arm, got %s", a.Name)
	}

	// android falls back to the linux build.
	asset, err := FindBinaryAsset(rel, "android", "arm64")
	if err != nil || asset.Name != "amdl_linux_arm64" {
		t.Fatalf("expected android/arm64 to fall back to amdl_linux_arm64, got %v, %v", asset, err)
	}

	// linux/arm64 falls back to the android build.
	onlyAndroid := &Release{TagName: "v1.2.0", Assets: []Asset{{Name: "amdl_android_arm64"}}}
	asset, err = FindBinaryAsset(onlyAndroid, "linux", "arm64")
	if err != nil || asset.Name != "amdl_android_arm64" {
		t.Fatalf("expected linux/arm64 to fall back to amdl_android_arm64, got %v, %v", asset, err)
	}
}

func TestDownloadAssetStall(t *testing.T) {
	orig := downloadStallTimeout
	downloadStallTimeout = 200 * time.Millisecond
	defer func() { downloadStallTimeout = orig }()

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	client, err := NewHTTPClient("", 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = DownloadAsset(client, srv.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("expected stall error, got %v", err)
	}
}

func TestDownloadAssetSlowButProgressing(t *testing.T) {
	orig := downloadStallTimeout
	downloadStallTimeout = 200 * time.Millisecond
	defer func() { downloadStallTimeout = orig }()

	// Total time exceeds the stall timeout, but each chunk arrives well within it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 5; i++ {
			w.Write([]byte("chunk"))
			w.(http.Flusher).Flush()
			time.Sleep(80 * time.Millisecond)
		}
	}))
	defer srv.Close()

	client, err := NewHTTPClient("", 0)
	if err != nil {
		t.Fatal(err)
	}
	data, err := DownloadAsset(client, srv.URL, nil)
	if err != nil {
		t.Fatalf("expected slow download to succeed, got %v", err)
	}
	if string(data) != strings.Repeat("chunk", 5) {
		t.Fatalf("unexpected body %q", data)
	}
}
