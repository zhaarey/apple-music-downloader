package updater

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// NewHTTPClient creates an http.Client configured with optional proxy.
func NewHTTPClient(proxyURL string) (*http.Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxyURL, err)
		}
		transport.Proxy = http.ProxyURL(parsed)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}, nil
}

// FetchLatestRelease fetches the latest release info from GitHub API.
func FetchLatestRelease(proxyURL string) (*Release, error) {
	client, err := NewHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "amdl-updater")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to GitHub API failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no release found on repository %s/%s", RepoOwner, RepoName)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release json: %w", err)
	}

	return &rel, nil
}

// FindBinaryAsset locates the single binary asset matching target OS and architecture.
func FindBinaryAsset(rel *Release, goos, goarch string) (*Asset, error) {
	// Expected filename formats:
	// amdl_windows_amd64.exe, amdl_linux_amd64, amdl_darwin_arm64, etc.
	expectedBase := fmt.Sprintf("amdl_%s_%s", goos, goarch)
	if goos == "windows" {
		expectedBase += ".exe"
	}

	for _, a := range rel.Assets {
		if strings.EqualFold(a.Name, expectedBase) {
			return &a, nil
		}
	}

	// Fallback loose match: contains both os and arch, not checksum or archive
	for _, a := range rel.Assets {
		lower := strings.ToLower(a.Name)
		if strings.Contains(lower, goos) && strings.Contains(lower, goarch) {
			if !strings.HasSuffix(lower, ".txt") && !strings.HasSuffix(lower, ".zip") && !strings.HasSuffix(lower, ".tar.gz") {
				return &a, nil
			}
		}
	}

	return nil, fmt.Errorf("no matching binary asset found for %s/%s in release %s", goos, goarch, rel.TagName)
}

// FindChecksumsAsset finds checksums.txt in the release assets.
func FindChecksumsAsset(rel *Release) (*Asset, error) {
	for _, a := range rel.Assets {
		lower := strings.ToLower(a.Name)
		if lower == "checksums.txt" || lower == "sha256sums" || lower == "sha256sums.txt" {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("checksums.txt not found in release %s", rel.TagName)
}

// DownloadAsset downloads the content of an asset with progress callback.
func DownloadAsset(client *http.Client, downloadURL string, onProgress func(downloaded, total int64)) ([]byte, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "amdl-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download asset failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned HTTP status %d", resp.StatusCode)
	}

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	var result []byte

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read download stream: %w", err)
		}
	}

	return result, nil
}

// VerifyChecksum checks the SHA256 of data against the checksums file content.
func VerifyChecksum(data []byte, filename string, checksumsContent string) error {
	sum := sha256.Sum256(data)
	actualHash := hex.EncodeToString(sum[:])

	baseName := filepath.Base(filename)
	scanner := bufio.NewScanner(strings.NewReader(checksumsContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			expectedHash := strings.ToLower(fields[0])
			entryFile := strings.TrimPrefix(fields[1], "*")
			if strings.EqualFold(filepath.Base(entryFile), baseName) {
				if strings.EqualFold(actualHash, expectedHash) {
					return nil
				}
				return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", baseName, expectedHash, actualHash)
			}
		}
	}

	return fmt.Errorf("file %s not found in checksums list", baseName)
}
