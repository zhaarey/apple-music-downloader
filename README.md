# Apple Music ALAC / Dolby Atmos Downloader

[English](./README.md) | [简体中文](./README-CN.md) | [Cloud Server / Proxy Setup](./PROXY-SETUP.md)

> **Original script by Sorrow.** Modified with fixes and improvements.

This command-line tool downloads albums, songs, playlists, stations and music videos from Apple Music, preserves or embeds metadata and lyrics, and supports ALAC, AAC and Dolby Atmos. Use it only with content you are entitled to access and in accordance with Apple's terms and applicable law.

## Contents

- [Features](#features)
- [Supported formats](#supported-formats)
- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage](#usage)
- [Upgrade](#upgrade)
- [For Developers](#for-developers)
- [Get media-user-token](#get-media-user-token)
- [Lyrics options](#lyrics-options)
- [Credits](#credits)

## Features

1. Inline cover art and LRC lyrics.
2. Word-by-word and unsynchronized lyrics.
3. Artist album downloads.
4. Streaming download and decryption for large files.
5. Music-video downloads using in-process mp4ff decryption.
6. Interactive search and track selection.

## Supported formats

| Format | Description | Requires subscription |
|---|---|---|
| `alac` | `audio-alac-stereo` | Yes |
| `ec3` | `audio-atmos` / `audio-ec3` | Yes |
| `aac` | `audio-stereo` | Yes |
| `aac-lc` | `audio-stereo` | Yes |
| `aac-binaural` | `audio-stereo-binaural` | Yes |
| `aac-downmix` | `audio-stereo-downmix` | Yes |
| `MV` | Music video | Yes |

Stations require a valid `media-user-token` from an active subscription.

## Requirements

Install and prepare these before running the downloader:

1. **wrapper-lite**: [github.com/WorldObservationLog/wrapper/tree/lite](https://github.com/WorldObservationLog/wrapper/tree/lite). Required backend decryption service. Start it before using this downloader and set its HTTP endpoint in `lite-server`, for example `http://127.0.0.1:12340`.
2. **ffmpeg** (Optional): Required only for post-download conversion, animated artwork, or `ffmpeg`-dependent features. See [ffmpeg.org](https://ffmpeg.org/).

> **Note**: If you are using the precompiled release binaries, **Go is NOT required**. Go (1.23.1+) is only needed if you build from source (see [For Developers](#for-developers)).

## Quick Start

Download the precompiled binary for your operating system and architecture from the [latest GitHub Releases](https://github.com/itouakirai/apple-music-downloader/releases/latest):

| Platform | Architecture | Precompiled Binary |
|---|---|---|
| **Windows** | x86_64 (64-bit) | `amdl_windows_amd64.exe` |
| **Windows** | ARM64 | `amdl_windows_arm64.exe` |
| **macOS** | Apple Silicon (M1/M2/M3/M4) | `amdl_darwin_arm64` |
| **macOS** | Intel (x86_64) | `amdl_darwin_amd64` |
| **Linux** | x86_64 (amd64) | `amdl_linux_amd64` |
| **Linux** | ARM64 (aarch64) | `amdl_linux_arm64` |
| **Android (Termux)** | ARM64 (aarch64) | `amdl_android_arm64` |

---

### Windows (PowerShell)

1. Open PowerShell and run the following command to download the executable:

```powershell
# Download precompiled binary (example for 64-bit Windows)
Invoke-WebRequest -Uri "https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_windows_amd64.exe" -OutFile "amdl.exe"
```

2. Run `amdl.exe` once to automatically generate the default `config.yaml`:

```powershell
.\amdl.exe
```

3. Open and edit `config.yaml` to set your `lite-server` address and destination paths.
4. Test running the downloader:

```powershell
.\amdl.exe --help
```

Example:

```powershell
.\amdl.exe "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

---

### macOS

1. Download the precompiled binary:

```bash
# For Apple Silicon (M-series):
curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_darwin_arm64

# For Intel Macs:
# curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_darwin_amd64

# Grant execution permission
chmod +x amdl
```

> **Tip (macOS Gatekeeper)**: If macOS blocks running `amdl` because it is from an unidentified developer, remove the quarantine attribute:
> ```bash
> xattr -d com.apple.quarantine amdl
> ```

2. Run `amdl` once to automatically generate the default `config.yaml`:

```bash
./amdl
```

3. Edit `config.yaml` with your settings.
4. Run the downloader:

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

---

### Linux

1. Download the precompiled binary:

```bash
# For x86_64 / amd64:
curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_linux_amd64

# For ARM64:
# curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_linux_arm64

# Grant execution permission
chmod +x amdl
```

2. Run `amdl` once to automatically generate the default `config.yaml`:

```bash
./amdl
```

3. Edit `config.yaml` with your settings.
4. Run the downloader:

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

---

### Android (Termux)

Install Termux from [F-Droid](https://f-droid.org/en/packages/com.termux/) or the [official GitHub releases](https://github.com/termux/termux-app/releases) (do not use the obsolete Play Store version).

1. Update packages and install curl (and optionally ffmpeg):

```bash
pkg update && pkg upgrade
pkg install -y curl ffmpeg
```

2. Grant access to shared Android storage (creates `~/storage/shared`):

```bash
termux-setup-storage
```

3. Download precompiled binary:

```bash
curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_android_arm64
chmod +x amdl
```

4. Run `amdl` once to automatically generate the default `config.yaml`:

```bash
./amdl
```

5. Edit `config.yaml`. To save downloads into Android's shared Music directory, point the paths in `config.yaml` to the shared mount:

```yaml
paths:
  alac: "/sdcard/Music/amdl"
  atmos: "/sdcard/Music/amdl-atmos"
  aac: "/sdcard/Music/amdl-aac"
  mv: "/sdcard/Music/amdl-mv"
```

6. Run:

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

> **Tips for Termux**:
> - For long downloads, prevent Android from sleeping with `termux-wake-lock` (release with `termux-wake-unlock` when done).
> - If wrapper-lite runs on another device on your network, point `lite-server` to that device's LAN IP, not `127.0.0.1`.
> - These instructions target current Android arm64 Termux environments; 32-bit Android is not a documented target.
> - If self-update on an older version fails with `lookup api.github.com on [::1]:53 ... connection refused`, re-download the binary once as in step 3; later versions update normally.

## Configuration

When `amdl` is run for the first time without an existing configuration file, it automatically creates `config.yaml` in the current directory from an embedded default template.

At minimum, review and set:

```yaml
general:
  # wrapper-lite HTTP API endpoint.
  lite-server: "http://127.0.0.1:12340"

  # Required for stations. See "Get media-user-token" below.
  media-user-token: "your-media-user-token"

# Destination folders. Relative paths are resolved from the working directory.
paths:
  alac: "AM-Lossless"
  atmos: "AM-Atmos"
  aac: "AM-AAC"
  mv: "AM-MV"
```

If wrapper-lite runs on another machine or container, replace `127.0.0.1` with that host's reachable LAN or public address.

## Usage

Before running any command, make sure:

1. wrapper-lite is running.
2. `config.yaml` exists and has the correct `lite-server` value.

### Album

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

On Windows:

```powershell
.\amdl.exe "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

### Single song

```bash
./amdl "https://music.apple.com/us/album/never-gonna-give-you-up-2022-remaster/1624945511?i=1624945512"
./amdl "https://music.apple.com/us/song/you-move-me-2022-remaster/1624945520"
```

### Artist albums

```bash
./amdl --all-album "https://music.apple.com/us/artist/taylor-swift/159260351"
```

### Playlist

```bash
./amdl "https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b"
```

### Interactive selection

```bash
./amdl --select "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

Enter track numbers separated by spaces.

### Interactive search

```bash
./amdl --search album "never gonna give you up"
./amdl --search song "you move me"
./amdl --search artist "taylor swift"
```

### Dolby Atmos

```bash
./amdl --atmos "https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538"
```

### AAC

```bash
./amdl --aac "https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538"
```

### Show quality information

```bash
./amdl --debug "https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538"
```

### Common options

```text
--alac-max <sample-rate>
--atmos-max <bitrate>
--aac-type <aac|aac-lc|aac-binaural|aac-downmix>
--mv-max <resolution>
--mv-audio-type <atmos|ac3|aac>
--lite-server <wrapper-lite-url>
```

## Get media-user-token

`media-user-token` is required for stations.

1. Open [Apple Music](https://music.apple.com) and sign in.
2. Open browser developer tools with `F12`.
3. Go to `Application` > `Storage` > `Cookies` > `https://music.apple.com`.
4. Find the cookie named `media-user-token` and copy its value.
5. Paste it into `general.media-user-token` in `config.yaml`.
6. Restart the downloader.

## Lyrics options

Configure lyrics settings under `metadata.lyrics` in `config.yaml`:

```yaml
metadata:
  lyrics:
    save-file: false                      # Save .lrc or .ttml file to disk
    embed: true                           # Embed lyrics directly into audio container tags
    type: "lyrics"                        # Lyrics type: "lyrics" (standard line-by-line) or "syllable-lyrics" (word-by-word)
    format: "lrc"                         # Lyrics file format: "lrc" or "ttml"
    extra: ""                             # Additional options: "" (default), "translation", or "pronunciation"
```

> **Tips**:
> - Set `general.language` in `config.yaml` (e.g. `"en-US"`, `"zh-Hans-CN"`, `"ja"`) to control the target language for translated lyrics.
> - `extra`: set to `"translation"` for translated lyrics or `"pronunciation"` for phonetic/romanized lyrics (when provided by Apple Music).

## Upgrade

### Built-in Self Update (Recommended)

Upgrade to the latest official release with automatic checksum verification and interactive configuration migration:

```bash
# Check and perform self-update with interactive config migration
./amdl --update

# Shorthand flag
./amdl -U

# Check for updates without downloading
./amdl --check-update

# Automated / non-interactive mode (automatically accept new option defaults)
./amdl -U -y
```

On Windows PowerShell:

```powershell
# Perform self-update
.\amdl.exe --update

# Shorthand flag
.\amdl.exe -U
```

> **Note**: Self-update automatically creates a timestamped `config.yaml.bak_...` backup and safely guides the migration of new settings without overwriting your credentials, custom paths, or comments. If a proxy is configured in `config.yaml`, the updater automatically routes through it.

### Manual Binary Update

Download the latest precompiled executable from [GitHub Releases](https://github.com/itouakirai/apple-music-downloader/releases/latest) and replace your current `amdl` / `amdl.exe` binary.

> For developers building from source, see [For Developers](#for-developers).

## For Developers

If you want to contribute, modify the code, or build the downloader from source:

### Prerequisites

1. **Go 1.23.1 or newer**: [go.dev/dl](https://go.dev/dl/).
2. **Git**: [git-scm.com](https://git-scm.com/).
3. **ffmpeg**: [ffmpeg.org](https://ffmpeg.org/) (optional for audio conversion or animated artwork).

### Clone and Build

**macOS / Linux**:

```bash
git clone https://github.com/itouakirai/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
./amdl --help
```

**Windows (PowerShell)**:

```powershell
git clone https://github.com/itouakirai/apple-music-downloader.git
cd apple-music-downloader
copy config.yaml.example config.yaml
go build -o amdl.exe .
.\amdl.exe --help
```

### Run in Development

```bash
go run . "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

### Update Source Code

```bash
git pull
go build -o amdl .
```

On Windows (PowerShell):

```powershell
git pull
go build -o amdl.exe .
```

## Credits

- **Sorrow** created the original script.
- **WorldObservationLog** created [wrapper / wrapper-lite](https://github.com/WorldObservationLog/wrapper), used as the backend decryption service.
- **Sendy McSenderson** contributed the streaming download-and-decrypt implementation.
- [go-mp4tag](https://github.com/itouakirai/go-mp4tag) writes station MP4 metadata.
- [FFmpeg](https://ffmpeg.org/) supports optional conversion and animated-artwork features.
