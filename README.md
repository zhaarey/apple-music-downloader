# Apple Music ALAC / Dolby Atmos Downloader

[English](./README.md) | [简体中文](./README-CN.md)

> **Original script by Sorrow.** Modified with fixes and improvements.

---

## ⚠️ Prerequisites

**Must be installed first:**

- **[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)** - Ensure it's correctly added to your environment variables
- **[wrapper](https://github.com/WorldObservationLog/wrapper)** - Decryption program must be running before use

**Optional (for MV download):**

- **[mp4decrypt](https://www.bento4.com/downloads/)**

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

1. Ensure the [wrapper](https://github.com/WorldObservationLog/wrapper) decryption program is running

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
