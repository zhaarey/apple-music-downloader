package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
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

const (
	// apiTimeout bounds a full GitHub API request.
	apiTimeout = 30 * time.Second
	// startupCheckTimeout keeps the startup update notice from delaying the program when GitHub is unreachable.
	startupCheckTimeout = 5 * time.Second
)

// downloadStallTimeout aborts an asset download when no data arrives for this long.
// A var so tests can shorten it.
var downloadStallTimeout = 60 * time.Second

// NewHTTPClient creates an http.Client configured with optional proxy.
// timeout bounds the whole request including reading the body; 0 disables it,
// which large downloads rely on (DownloadAsset applies its own stall timeout).
func NewHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
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
		Timeout:   timeout,
	}, nil
}

// FetchLatestRelease fetches the latest release info from GitHub API.
func FetchLatestRelease(proxyURL string) (*Release, error) {
	client, err := NewHTTPClient(proxyURL, apiTimeout)
	if err != nil {
		return nil, err
	}
	return fetchLatestRelease(client)
}

func fetchLatestRelease(client *http.Client) (*Release, error) {
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
// Only exact release file names are accepted: a loose substring match would let
// e.g. linux/arm pick amdl_linux_arm64, whose checksum still verifies.
func FindBinaryAsset(rel *Release, goos, goarch string) (*Asset, error) {
	candidates := []string{binaryAssetName(goos, goarch)}
	// Android/Termux and linux/arm64 releases are the same GOOS=linux build.
	switch {
	case goos == "android":
		candidates = append(candidates, binaryAssetName("linux", goarch))
	case goos == "linux" && goarch == "arm64":
		candidates = append(candidates, binaryAssetName("android", goarch))
	}

	for _, name := range candidates {
		for i := range rel.Assets {
			if strings.EqualFold(rel.Assets[i].Name, name) {
				return &rel.Assets[i], nil
			}
		}
	}

	return nil, fmt.Errorf("no matching binary asset found for %s/%s in release %s", goos, goarch, rel.TagName)
}

// binaryAssetName returns the release file name, e.g. amdl_windows_amd64.exe or amdl_linux_arm64.
func binaryAssetName(goos, goarch string) string {
	name := fmt.Sprintf("amdl_%s_%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
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
// The download is aborted if no data arrives for downloadStallTimeout, so slow
// but progressing connections can finish while dead ones don't hang forever.
func DownloadAsset(client *http.Client, downloadURL string, onProgress func(downloaded, total int64)) ([]byte, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stall := time.AfterFunc(downloadStallTimeout, cancel)
	defer stall.Stop()

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "amdl-updater")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("download asset failed: no response within %s", downloadStallTimeout)
		}
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
			stall.Reset(downloadStallTimeout)
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
			if ctx.Err() != nil {
				return nil, fmt.Errorf("download stalled: no data received for %s", downloadStallTimeout)
			}
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
