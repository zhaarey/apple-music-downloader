# Apple Music ALAC / Dolby Atmos Downloader

[English](./README.md) | [简体中文](./README-CN.md)

> **Original script by Sorrow.** Modified with fixes and improvements.

---

## Quick start — Windows GUI (`.exe`)

This is the fastest way to build and run the desktop app on Windows.

### 1. Prerequisites

| Requirement | Purpose | Notes |
|-------------|---------|-------|
| [Go 1.23+](https://go.dev/dl/) | Build backend | Add `go` to PATH |
| [Node.js 18+](https://nodejs.org/) | Build React UI | Used by Wails frontend |
| [Wails CLI v2](https://wails.io/docs/gettingstarted/installation) | Package GUI | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| [MP4Box](https://gpac.io/downloads/gpac-nightly-builds/) | Tagging & flattening | On PATH, or bundled in `dist/tools/` |
| Apple Music subscription | Downloads | Required for all formats |
| `media-user-token` | Lyrics, MV, AAC-LC | Paste in Settings (see below) |

**Optional**

- [Inno Setup 6](https://jrsoftware.org/isinfo.php) — builds `AppleMusicDownloader-Setup.exe`
- [wrapper](https://github.com/WorldObservationLog/wrapper) — **only** for ALAC / Dolby Atmos (see [README-WINDOWS.md](./README-WINDOWS.md))
- [mp4decrypt](https://www.bento4.com/downloads/) — music video decryption

> **AAC downloads do not require wrapper.** The GUI uses in-process Widevine for AAC-LC.

### 2. Build the executables

From the project root in PowerShell:

```powershell
.\scripts\build-windows.ps1 -SkipInstaller
```

This produces:

| Output | Description |
|--------|-------------|
| `dist\AppleMusicDownloader.exe` | Desktop GUI |
| `dist\amd.exe` | CLI (same engine) |

To also build the installer (requires Inno Setup):

```powershell
.\scripts\build-windows.ps1
```

Place optional tools in `dist\tools\` before packaging: `MP4Box.exe`, `ffmpeg.exe`, `mp4decrypt.exe`.

### 3. Run the app

```powershell
.\dist\AppleMusicDownloader.exe
```

On first launch, complete the setup wizard:

1. Choose your **storefront** (e.g. `us`)
2. Paste your **`media-user-token`** (recommended — required for AAC, lyrics, and MV)
3. Set your **download folder**

Config is saved to:

```
%APPDATA%\AppleMusicDownloader\config.yaml
```

Logs:

```
%APPDATA%\AppleMusicDownloader\logs\app.log
```

### 4. Download music

1. Open the **Download** tab
2. Paste an Apple Music album, song, or playlist URL
3. Select **AAC** quality (works without wrapper)
4. Click **Start download**

Use **Settings → Requirements** to verify MP4Box and token status. For ALAC / Atmos, see **[README-WINDOWS.md](./README-WINDOWS.md)**.

### 5. CLI alternative

The same build also outputs a command-line binary:

```powershell
.\dist\amd.exe --aac "https://music.apple.com/us/album/..."
```

---

## ⚠️ Prerequisites (CLI / Docker)

For **CLI-only** or **Docker** usage (non-GUI):

- **[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)** — tagging
- **[wrapper](https://github.com/WorldObservationLog/wrapper)** — required for ALAC / Atmos / most CLI flows
- **`media-user-token`** — required for AAC-LC, lyrics, and MV

---

## ✨ Features

1. **Inline Covers & LRC Lyrics** - Requires `media-user-token` (see instructions below)
2. **Word-by-word & Out-of-sync Lyrics** support
3. **Artist Album Download** - Automatically download all albums from an artist
   ```bash
   go run main.go https://music.apple.com/us/artist/taylor-swift/159260351 --all-album
   ```
4. **Stream Decryption** - Uses Sendy McSenderson's code for download-and-decrypt streaming, solving memory issues with large files
5. **MV Download** - Requires mp4decrypt installation
6. **Interactive Search** - Arrow-key navigation for search results
   ```bash
   go run main.go --search [song/album/artist] "search_term"
   ```
7. **Windows GUI** - Paste URLs, queue downloads, manage settings (see Quick start above)

---

## 🎵 Supported Audio Formats

| Format | Description | Requires Subscription |
|--------|-------------|----------------------|
| `alac` | audio-alac-stereo | ✅ |
| `ec3` | audio-atmos / audio-ec3 | ✅ |
| `aac` | audio-stereo | ✅ |
| `aac-lc` | audio-stereo | ✅ |
| `aac-binaural` | audio-stereo-binaural | ✅ |
| `aac-downmix` | audio-stereo-downmix | ✅ |
| `MV` | Music Video | ✅ |

> **Note:** For `aac-lc`, `MV`, and `lyrics`, you must provide a valid `media-user-token` from an active subscription.

| Format | Wrapper required? |
|--------|-------------------|
| AAC / AAC-LC (GUI) | No |
| ALAC / Atmos | Yes |
| MV | No (needs `mp4decrypt`) |

---

## Windows GUI details

See **[README-WINDOWS.md](./README-WINDOWS.md)** for installer usage, bundled tools, wrapper/WSL setup for ALAC/Atmos, and advanced build options.

---

## 🚀 Usage

### Running with Docker

1. Ensure the [wrapper](https://github.com/WorldObservationLog/wrapper) decryption program is running

2. Start the downloader:

```bash
# Show help
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --help

# Download albums
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader https://music.apple.com/ru/album/children-of-forever/1443732441

# Download single song
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --song https://music.apple.com/ru/album/bass-folk-song/1443732441?i=1443732453

# Interactive selection
docker run -it --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --select https://music.apple.com/ru/album/children-of-forever/1443732441

# Download playlists
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b

# Dolby Atmos
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538

# AAC format
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --aac https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538

# Debug/View quality
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --debug https://music.apple.com/ru/album/miles-smiles/209407331
```

**Custom Configuration:**

Mount your own `config.yaml`:

```bash
docker run --network host -v ./downloads:/downloads -v ./config.yaml:/app/config.yaml ghcr.io/zhaarey/apple-music-downloader [args]
```

> **Note:** Ensure `config.yaml` exists in your current directory before running. If it doesn't exist, Docker will create an empty directory instead of a file, causing the container to fail.

---

### Running Locally (Go)

1. Ensure the [wrapper](https://github.com/WorldObservationLog/wrapper) decryption program is running (ALAC/Atmos only; AAC-LC works without it)

2. **Download albums:**
   ```bash
   go run main.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511
   ```

3. **Download single song:**
   ```bash
   go run main.go --song https://music.apple.com/us/album/never-gonna-give-you-up-2022-remaster/1624945511?i=1624945512
   # or
   go run main.go https://music.apple.com/us/song/you-move-me-2022-remaster/1624945520
   ```

4. **Interactive selection:**
   ```bash
   go run main.go --select https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511
   ```
   Enter track numbers separated by spaces.

5. **Download playlists:**
   ```bash
   go run main.go https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b
   # or
   go run main.go https://music.apple.com/us/playlist/hi-res-lossless-24-bit-192khz/pl.u-MDAWvpjt38370N
   ```

6. **Dolby Atmos:**
   ```bash
   go run main.go --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
   ```

7. **AAC format:**
   ```bash
   go run main.go --aac https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
   ```

8. **View quality info:**
   ```bash
   go run main.go --debug https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
   ```

📖 [Chinese Tutorial (Method 3)](https://telegra.ph/Apple-Music-Alac%E9%AB%98%E8%A7%A3%E6%9E%90%E5%BA%A6%E6%97%A0%E6%8D%9F%E9%9F%B3%E4%B9%90%E4%B8%8B%E8%BD%BD%E6%95%99%E7%A8%8B-04-02-2)

---

## 📝 Getting media-user-token (For Lyrics)

1. Open [Apple Music](https://music.apple.com) and log in
2. Open Developer Tools (F12)
3. Navigate to `Application → Storage → Cookies → https://music.apple.com`
4. Find the cookie named `media-user-token` and copy its value
5. Paste the value into `config.yaml` under the `media-user-token` setting
6. Save the file and start the script

---

## 🌐 Getting Translation & Pronunciation Lyrics (Beta)

> **Note:** These features are currently in beta.

1. Open [Apple Music Beta](https://beta.music.apple.com) and log in
2. Open Developer Tools (F12) and switch to the **Network** tab
3. Search for a song that supports translation/pronunciation lyrics (K-Pop songs recommended)
4. Press **Ctrl+R** to refresh and let DevTools capture network traffic
5. Play the song and click the lyrics button - look for a request named `syllable-lyrics`
6. Stop recording (click the red circle button), then select the **Fetch/XHR** tab
7. Click on the `syllable-lyrics` request to view details
8. Find the URL containing: `.../syllable-lyrics?l=<language_code>&extend=ttmlLocalizations`
9. Copy the language value and paste it into `config.yaml`
10. **Optional:** To disable pronunciation, remove the corresponding value in config.yaml: `...%5D=<remove_this_value>&extend...`
11. Save and run the script as usual

---

## 👏 Special Thanks

- **chocomint** - Created `agent-arm64.js`

---