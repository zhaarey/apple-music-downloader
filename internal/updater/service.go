package updater

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"amdl/internal/version"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

// PrintStartupUpdateNotice performs a quick, cached check and prints a subtle banner if an update exists.
func PrintStartupUpdateNotice(proxyURL string) {
	if version.IsDev() {
		return
	}

	// Throttle checks to once every 24 hours
	res, err := CheckUpdateCached(version.Version, proxyURL, 24*time.Hour)
	if err == nil && res != nil && res.HasUpdate {
		fmt.Printf("\033[1;33m🔔 [Update] A new version %s is available! (Current: %s). Run 'amdl --update' to upgrade.\033[0m\n\n",
			res.LatestVersion, res.CurrentVersion)
	}
}

// CheckUpdateOnly fetches and displays the latest version status without downloading.
func CheckUpdateOnly(proxyURL string) error {
	fmt.Printf("Current version: %s\n", version.Info())
	fmt.Printf("Checking for updates from https://github.com/%s/%s...\n", RepoOwner, RepoName)

	rel, err := FetchLatestRelease(proxyURL)
	if err != nil {
		return fmt.Errorf("check update failed: %w", err)
	}

	cmp := version.Compare(version.Version, rel.TagName)
	if cmp >= 0 && !version.IsDev() {
		fmt.Printf("✔ You are already on the latest version (%s).\n", version.Version)
		return nil
	}

	fmt.Printf("\n✨ Latest version available: %s\n", rel.TagName)
	fmt.Printf("Published at: %s\n", rel.PublishedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Release URL:  %s\n", rel.HTMLURL)
	if rel.Body != "" {
		fmt.Println("\nRelease Notes:")
		fmt.Println(strings.TrimSpace(rel.Body))
	}
	fmt.Println("\nRun 'amdl --update' to perform the upgrade.")
	return nil
}

// ExecuteSelfUpdate coordinates the complete update flow: check -> download -> verify -> swap -> migrate config.
// force reinstalls the latest release even when already up to date.
func ExecuteSelfUpdate(configFile, proxyURL string, autoYes, force bool) error {
	fmt.Println("==================================================================")
	fmt.Println("               AMDL Self-Update Assistant                         ")
	fmt.Println("==================================================================")
	fmt.Printf("Current: %s\n", version.Info())

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	if version.IsDev() {
		fmt.Println("\n\033[1;33mNotice: You are running a development or locally built version (dev).\033[0m")
		fmt.Println("Self-update installs official precompiled release binaries from GitHub.")
		fmt.Println("If you cloned this repository, consider running: 'git pull && go build' instead.")
		if !autoYes {
			if !isTTY {
				// Never overwrite a local build without explicit consent.
				fmt.Println("Update canceled: non-interactive session. Re-run with --yes to overwrite this build.")
				return nil
			}
			fmt.Print("Do you want to proceed and overwrite with official release? [y/N]: ")
			ans, _ := reader.ReadString('\n')
			if !strings.EqualFold(strings.TrimSpace(ans), "y") {
				fmt.Println("Update canceled.")
				return nil
			}
		}
	}

	fmt.Printf("Querying GitHub for latest release (%s/%s)...\n", RepoOwner, RepoName)
	rel, err := FetchLatestRelease(proxyURL)
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	cmp := version.Compare(version.Version, rel.TagName)
	if cmp >= 0 && !version.IsDev() && !force {
		fmt.Printf("✔ amdl is already up to date (%s).\n", version.Version)
		// --yes only accepts prompts; reinstalling needs an explicit prompt answer or --force-update.
		if autoYes || !isTTY {
			return nil
		}
		fmt.Print("Do you want to reinstall/force update anyway? [y/N]: ")
		ans, _ := reader.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(ans), "y") {
			return nil
		}
	}

	// No overall timeout: DownloadAsset aborts on stalls instead, so large binaries
	// over slow links can still complete.
	client, err := NewHTTPClient(proxyURL, 0)
	if err != nil {
		return err
	}

	fmt.Printf("\nTarget version: \033[1;32m%s\033[0m\n", rel.TagName)
	if rel.Name != "" && rel.Name != rel.TagName {
		fmt.Printf("Release:        %s\n", rel.Name)
	}
	if rel.Body != "" {
		fmt.Println("\nRelease Notes:")
		fmt.Println(strings.TrimSpace(rel.Body))
	}
	fmt.Println()

	binAsset, err := FindBinaryAsset(rel, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return fmt.Errorf("locate binary asset: %w", err)
	}

	sumsAsset, err := FindChecksumsAsset(rel)
	if err != nil {
		return fmt.Errorf("locate checksums asset: %w", err)
	}

	// 1. Download checksums.txt
	fmt.Printf("Downloading checksums (%s)...\n", sumsAsset.Name)
	sumsBytes, err := DownloadAsset(client, sumsAsset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	// 2. Download binary with progress bar
	fmt.Printf("Downloading %s (%0.2f MB)...\n", binAsset.Name, float64(binAsset.Size)/(1024*1024))
	bar := progressbar.NewOptions64(
		binAsset.Size,
		progressbar.OptionSetDescription("Progress"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(30),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)

	binBytes, err := DownloadAsset(client, binAsset.BrowserDownloadURL, func(downloaded, total int64) {
		_ = bar.Set64(downloaded)
	})
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	// 3. Verify SHA256 checksum
	fmt.Printf("Verifying SHA256 integrity for %s...\n", binAsset.Name)
	if err := VerifyChecksum(binBytes, binAsset.Name, string(sumsBytes)); err != nil {
		return fmt.Errorf("integrity verification failed: %w", err)
	}
	fmt.Println("✔ Checksum verified successfully.")

	// 4. Atomically apply binary update
	fmt.Println("Applying binary replacement...")
	exePath, err := ApplyBinaryUpdate(binBytes)
	if err != nil {
		return fmt.Errorf("binary replacement failed: %w", err)
	}
	fmt.Println("✔ Binary executable updated successfully.")

	// 5. Config migration runs in the new binary: this process only embeds the
	// old config.yaml.example and would never see options added by the release.
	if err := runMigrationInBinary(exePath, configFile, autoYes); err != nil {
		fmt.Printf("\033[1;33mWarning: config migration had an issue: %v\033[0m\n", err)
		fmt.Println("Run 'amdl --migrate-config' to retry.")
	}

	fmt.Println("\n==================================================================")
	fmt.Printf("✨ Update to %s completed successfully!\n", rel.TagName)
	fmt.Println("==================================================================")
	return nil
}

// runMigrationInBinary runs `<exePath> --migrate-config` so the migration
// compares against the config.yaml.example embedded in that binary.
func runMigrationInBinary(exePath, configFile string, autoYes bool) error {
	args := []string{"--migrate-config"}
	if configFile != "" {
		args = append(args, "--config", configFile)
	}
	if autoYes {
		args = append(args, "--yes")
	}

	cmd := exec.Command(exePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunConfigMigration adds options missing from the user config, using the
// config.yaml.example embedded in the running binary. It backs --migrate-config.
func RunConfigMigration(configFile string, autoYes bool) error {
	if configFile == "" {
		configFile = "config.yaml"
	}
	// A config.yaml.example in the working directory is deliberately not used:
	// it may be left over from an older release.
	if DefaultConfigExample == "" {
		return fmt.Errorf("this build has no embedded config.yaml.example to compare against")
	}

	_, err := RunConfigMigrationGuide(configFile, DefaultConfigExample, version.Version, autoYes)
	return err
}
