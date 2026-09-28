package updater

// RepoOwner and RepoName identify the GitHub repository self-update pulls
// releases from. Release builds override them via -ldflags so a fork's
// binaries update from the fork and upstream's from upstream.
var (
	RepoOwner = "zhaarey"
	RepoName  = "apple-music-downloader"
)

// DefaultConfigExample holds the embedded content of config.yaml.example,
// populated at runtime from package main via init().
var DefaultConfigExample string
