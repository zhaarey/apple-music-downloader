# Apple Music ALAC / Dolby Atmos Downloader

[English](./README.md) | [简体中文](./README-CN.md) | [Cloud Server / Proxy Setup](./PROXY-SETUP.md)

> **Original script by Sorrow.** Modified with fixes and improvements.

This command-line tool downloads albums, songs, playlists, stations and music videos from Apple Music, preserves or embeds metadata and lyrics, and supports ALAC, AAC and Dolby Atmos. Use it only with content you are entitled to access and in accordance with Apple's terms and applicable law.

## Contents

- [Features](#features)
- [Supported formats](#supported-formats)
- [Requirements](#requirements)
- [Configuration](#configuration)
- [Install on Windows](#install-on-windows)
- [Install on macOS](#install-on-macos)
- [Install on Linux](#install-on-linux)
- [Install on Android with Termux](#install-on-android-with-termux)
- [Usage](#usage)
- [Get media-user-token](#get-media-user-token)
- [Lyrics options](#lyrics-options)
- [Upgrade](#upgrade)
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

Install these before running the downloader:

1. **Go 1.23.1 or newer**: [go.dev/dl](https://go.dev/dl/).
2. **wrapper-lite**: [github.com/WorldObservationLog/wrapper/tree/lite](https://github.com/WorldObservationLog/wrapper/tree/lite). Start it before using this downloader and set its HTTP endpoint in `lite-server`, for example `http://127.0.0.1:12340`.
3. **ffmpeg**: required only for post-download conversion, animated artwork, or `ffmpeg`-dependent features. See [ffmpeg.org](https://ffmpeg.org/).

## Configuration

Copy the example config to `config.yaml` in the project root:

```bash
cp config.yaml.example config.yaml
```

On Windows PowerShell, use:

```powershell
copy config.yaml.example config.yaml
```

At minimum, review and set:

```yaml
# wrapper-lite HTTP API endpoint.
lite-server: "http://127.0.0.1:12340"

# Required for stations. See "Get media-user-token" below.
media-user-token: "your-media-user-token"

# Destination folders. Relative paths are resolved from the working directory.
alac-save-folder: "AM-DL downloads"
atmos-save-folder: "AM-DL-Atmos downloads"
aac-save-folder: "AM-DL-AAC downloads"
mv-save-folder: "AM-DL-MV downloads"

# Required for ffmpeg-based conversion or animated artwork.
convert-after-download: false
save-animated-artwork: false
```

If wrapper-lite runs on another machine or container, replace `127.0.0.1` with that host's reachable LAN or public address.

## Install on Windows

1. Install **Git**: [git-scm.com/download/win](https://git-scm.com/download/win).
2. Install **Go 1.23.1 or newer**: [go.dev/dl](https://go.dev/dl/).
3. Install **ffmpeg** if you plan to convert files or save animated artwork: [ffmpeg.org/download.html](https://ffmpeg.org/download.html).

From PowerShell:

```powershell
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
copy config.yaml.example config.yaml
go build -o amdl.exe .
.\amdl.exe --help
```

Example:

```powershell
.\amdl.exe "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

## Install on macOS

1. Install Homebrew: [brew.sh](https://brew.sh/).
2. Install the runtime and media tools:

```bash
brew install go git ffmpeg
```

Then clone, configure and build:

```bash
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
./amdl --help
```

Example:

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

## Install on Linux

Choose the commands for your distribution. Package names may differ on other distributions.

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y git build-essential ffmpeg
```

If your repository's Go package is older than `1.23.1`, install Go from [go.dev/dl](https://go.dev/dl/) instead of using the distro package.

### Fedora

```bash
sudo dnf install -y git gcc make ffmpeg
```

If needed, install Go manually from the official site.

### Arch Linux

```bash
sudo pacman -S --needed git base-devel ffmpeg
```

If needed, install Go manually from the official site.

Then clone, configure and build:

```bash
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
./amdl --help
```

Example:

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

## Install on Android with Termux

Install Termux from [F-Droid](https://f-droid.org/en/packages/com.termux/) or the [official GitHub releases](https://github.com/termux/termux-app/releases). Do not use the outdated Play Store build.

1. Update the package index and installed packages:

```bash
pkg update && pkg upgrade
```

2. Install the toolchain and media tools:

```bash
pkg install golang git ffmpeg
```

3. Optionally grant access to shared Android storage. Termux creates the `~/storage/shared` tree after you approve the prompt:

```bash
termux-setup-storage
```

4. Clone, configure and build:

```bash
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
```

5. To save into Android shared music storage, point the save folders at the shared storage mount:

```yaml
alac-save-folder: "/sdcard/Music/amdl"
atmos-save-folder: "/sdcard/Music/amdl-atmos"
aac-save-folder: "/sdcard/Music/amdl-aac"
mv-save-folder: "/sdcard/Music/amdl-mv"
```

Run normally:

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

For long downloads, keep Android from suspending Termux:

```bash
termux-wake-lock
```

Release the lock when finished:

```bash
termux-wake-unlock
```

If wrapper-lite runs on another device, set `lite-server` to that device's LAN or public address, not `127.0.0.1`. These instructions target current Android arm64 Termux environments; 32-bit Android is not a documented target.

## Usage

Before running any command, make sure:

1. wrapper-lite is running.
2. `config.yaml` exists and has the correct `lite-server` value.

### Album

```bash
go run . "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

Or use the built binary:

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
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
5. Paste it into `media-user-token` in `config.yaml`.
6. Restart the downloader.

## Lyrics options

Key settings in `config.yaml`:

```yaml
lrc-type: "lyrics"          # lyrics or syllable-lyrics
lrc-format: "lrc"           # lrc or ttml
lrc-extra: ""               # translation or pronunciation
embed-lrc: true             # embed lyrics in the media file
save-lrc-file: false        # also save an external .lrc file
```

Set `lrc-extra` according to the service's language or feature code when you want translated or phonetic lyrics.

## Upgrade

Pull the latest source and rebuild:

```bash
git pull
go build -o amdl .
```

On Windows:

```powershell
git pull
go build -o amdl.exe .
```

If `config.yaml.example` gains new options, compare it with your `config.yaml` before copying anything. Preserve your existing credentials and save folders.

## Credits

- **Sorrow** created the original script.
- **WorldObservationLog** created [wrapper / wrapper-lite](https://github.com/WorldObservationLog/wrapper), used as the backend decryption service.
- **Sendy McSenderson** contributed the streaming download-and-decrypt implementation.
- [go-mp4tag](https://github.com/itouakirai/go-mp4tag) writes station MP4 metadata.
- [FFmpeg](https://ffmpeg.org/) supports optional conversion and animated-artwork features.
