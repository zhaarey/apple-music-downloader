# Aura Audio Downloader

Download Apple Music, pull YouTube DJ sets, trim dead space, split long recordings into tracks, and fix tags — all in one **desktop** app (Windows and macOS) built for syncing to Apple Music on iPhone and Mac. A native **iPhone app is in design** — not shipped yet; see [iPhone app (Aura iOS)](#iphone-app-aura-ios).

---

## Install and run

### Windows

1. Run `AuraAudioDownloader-Setup.exe`, or build from source (see [Build from source](#build-from-source)).
2. Launch **Aura Audio Downloader** from the Start Menu.
3. Complete the first-run wizard: storefront, `media-user-token`, and download folder.

```powershell
# Build from source (project root)
.\scripts\build-windows.ps1 -SkipInstaller

# Run
.\dist\AuraAudioDownloader.exe
```

Windows-specific troubleshooting (artwork sync, bundled tools): **[README-WINDOWS.md](./README-WINDOWS.md)**

### macOS

```bash
chmod +x scripts/build-macos.sh
./scripts/build-macos.sh
open dist/AuraAudioDownloader.app
```

On macOS the app includes **Apple Music**, **YouTube**, **Trim**, and **Tag Editor** tabs. Split mix is Windows-only today.

### Config and logs

| Platform | Config | Logs |
|----------|--------|------|
| Windows | `%APPDATA%\AuraAudioDownloader\config.yaml` | `%APPDATA%\AuraAudioDownloader\logs\app.log` |
| macOS | `~/Library/Application Support/AuraAudioDownloader/config.yaml` | `~/Library/Application Support/AuraAudioDownloader/logs/app.log` |

Settings from the older **Apple Music Downloader** folder are migrated automatically on first launch.

---

## First-time setup

On first launch, the wizard walks you through:

1. **Storefront** — your Apple Music region (e.g. `us`, `gb`, `jp`).
2. **`media-user-token`** — required for Apple Music AAC downloads, lyrics, and some metadata.  
   Open [music.apple.com](https://music.apple.com) while logged in → DevTools (F12) → **Application → Cookies** → copy `media-user-token` → paste in Settings or the wizard.
3. **Output folder** — where AAC and other downloads are saved.
4. **Dependencies** — MP4Box, ffmpeg, yt-dlp, etc. The installer bundles most tools; use the **Requirements** tab to verify.

You can reopen setup anytime from **Settings**.

---

## App overview

Aura is organized around a few workflows. While a download runs, you can switch to **Trim**, **Tag Editor**, **Split mix**, or **Settings** without stopping the job.

| Tab | What it does |
|-----|----------------|
| **Apple Music** | Download albums, songs, and playlists from Apple Music URLs |
| **YouTube** | Download DJ sets and mixes as tagged AAC (and optional MP4 video) |
| **Split mix** *(Windows)* | Cut one long recording into many tagged tracks using a tracklist and waveform |
| **Trim** | Remove dead space at the start or end of a single `.m4a` or `.mp4` |
| **Tag Editor** | Edit metadata and artwork on local files; bulk-edit an album folder |
| **Activity** | Job progress, failed tracks, and log access |
| **Requirements** | Live tool detection and feature availability |
| **Settings** | Token, folders, YouTube options, sync repair helpers |

---

## Key features and how to use them

### Apple Music downloads

1. Open the **Apple Music** tab.
2. Paste an album, song, or playlist URL from [music.apple.com](https://music.apple.com).
3. Choose quality:
   - **AAC** — works out of the box with your subscription and token.
   - **Lossless (ALAC)** / **Dolby Atmos** — require the wrapper decrypt service (see **Requirements** and [README-WINDOWS.md](./README-WINDOWS.md)).
4. Click **Fetch**, review the track list, then **Start download**.

**Bulk queue:** search the catalog, paste multiple URLs, or add tracks from search results. For Apple Music **playlists**, you can expand the preview and pick individual tracks before queuing.

**Output:** tagged `.m4a` files (AAC 256 kbps by default), with optional embedded lyrics and cover art when enabled in Settings.

### Spotify track → Apple Music

Paste a **single Spotify track link** (`open.spotify.com/track/…`) in the lookup panel on the Apple Music tab. Aura finds the best Apple Music match so you can queue it — no Spotify API keys needed.

Playlists and albums are not supported for matching; use individual track links only.

### YouTube downloads

1. Open the **YouTube** tab.
2. Paste a video or playlist URL.
3. Choose delivery mode:
   - **Audio only** — AAC 256 kbps, tagged for Apple Music.
   - **Audio + video** — AAC plus an H.264 `.mp4` that plays in the iPhone Music app.
   - **Video only** — H.264 `.mp4` with embedded AAC stereo.
4. Edit title, artist, album, and artwork in the metadata panel if needed.
5. Start the download.

Requires **yt-dlp** and **ffmpeg** (bundled in `tools/` or on PATH). No Apple account needed for YouTube mode.

**After a download finishes**, the completion banner offers:

- **Split into tracks** — opens **Split mix** with the file ready (Windows).
- **Trim dead space** — opens **Trim** with the file loaded.

### Split mix *(Windows)*

Use this for long DJ sets where you need **many tracks**, not a single cut.

1. Download a YouTube set, or open **Split mix** and pick an existing `.m4a`.
2. Paste a timestamped tracklist (`0:00 Artist - Title` per line), or add tracks manually.
3. Drag cut lines on the waveform; preview each segment.
4. Use **Fit durations to master** if timings drift slightly.
5. Set album metadata and artwork, then **Export tracks**.

Each export is a separate AAC file with track numbers and tags, ready to import into Apple Music.

### Trim

Use this to remove intro/outro silence from **one file** — common after YouTube downloads.

1. Open **Trim**, or use **Trim dead space** / **Open in Trim** from a download or the Tag Editor.
2. Preview audio (`.m4a`) or video (`.mp4`); drag **start** and **end** handles on the waveform.
3. Fine-tune with time fields or ±1 s / ±100 ms nudges (minimum selection: 1 second).
4. **Save as new** — pick a destination path.
5. **Save** — replaces the original after creating a one-time `.bak` backup.

Tags can be copied from the source on export. Supports `.m4a` and `.mp4` only in v1.

### Tag Editor

Open any local `.m4a`, `.mp4`, or other supported audio file:

- Edit title, artist, album, track numbers, genre, year.
- Replace or optimize artwork for Apple Music / iPhone sync.
- **Album bulk** — edit every track in the same folder at once.
- **Open in Trim** — jump to Trim with the current file.
- **Sync repair tools** — fix missing iPhone artwork when PC tiles look fine (see [README-WINDOWS.md](./README-WINDOWS.md)).

Drag and drop files onto the tab to open them quickly.

### Activity and Requirements

- **Activity** — current and recent jobs, per-track failures, cancel in-progress downloads, open output folder or log file.
- **Requirements** — see which tools are detected, what each feature needs, and AAC troubleshooting tips.

---

## Typical workflows

### YouTube DJ set → library-ready tracks

```
YouTube tab → download set → Split into tracks → adjust cuts → Export tracks → import to Apple Music
```

### Quick cleanup after YouTube download

```
YouTube tab → download → Trim dead space → adjust in/out → Save or Save as new
```

### Fix tags before syncing to iPhone

```
Tag Editor → open file or folder → edit tags/artwork → Save tags → Sync repair if art missing on phone
```

### Find an Apple Music version of a Spotify song

```
Apple Music tab → paste Spotify track URL → pick match → add to queue → download
```

---

## What you need

| Feature | Requirements |
|---------|----------------|
| Apple Music AAC | Active subscription + `media-user-token` |
| Lyrics (LRC) | Valid `media-user-token` |
| YouTube audio / video | yt-dlp + ffmpeg |
| Split mix | ffmpeg + ffprobe |
| Trim | ffmpeg |
| Tag editor | MP4Box recommended for embed operations |
| ALAC / Dolby Atmos | Wrapper decrypt service + subscription |
| Spotify track match | Internet only (single track links) |

Check the **Requirements** tab inside the app for live status.

---

## CLI

The same engine ships as a command-line tool:

```powershell
.\dist\aura.exe --help
```

Legacy name `amd.exe` is also included on Windows. Useful for scripts and headless use; the GUI is the recommended experience for tagging, trim, and split workflows.

Example:

```powershell
.\dist\aura.exe --aac "https://music.apple.com/us/album/..."
```

---

## iPhone app (Aura iOS)

There is **no installable IPA in this repo yet**. Aura on iPhone is **designed but not built** — specs and shared export rules live in **[docs/ios/](./docs/ios/)**. The desktop app (Windows/macOS) is what you use today; iPhone is the playback target after sync, or the future on-device download source.

### What works today (without the IPA)

| Goal | How |
|------|-----|
| Listen offline in the Music app | Download on **PC** → import into Apple Music → sync to iPhone |
| Fix tags / trim / split | **Tag Editor**, **Trim**, and **Split mix** on desktop only |
| Music videos (`.mp4`) on iPhone | YouTube **Audio + video** on desktop → import/sync, or edit with desktop Tag Editor |

See [README-WINDOWS.md](./README-WINDOWS.md) for iPhone artwork sync troubleshooting.

### What the IPA is planned to do

On-device downloads with the **same file layout** as desktop (`01. Title.m4a`, `cover.jpg`, `01. Title [video].mp4`) so copies picked up on a PC open in **Tag Editor** and **Trim** without re-encoding. Spec: [cross-device-export.md](./docs/ios/cross-device-export.md).

| Feature | Planned | Notes |
|---------|---------|--------|
| **YouTube download** | Yes (Phase A) | AAC 256k, optional H.264 music-video MP4; metadata + artwork before download |
| **YouTube delivery modes** | Yes | Audio only · Audio + video · Video only (same as desktop) |
| **Add to Music library** | Guided manual import | No silent auto-import — iOS has no public API; in-app steps for **Music → Add from Files** |
| **Keep files for PC** | Yes | `Documents/Downloads/{Album}/` visible in Files app; optional iCloud export |
| **Pre-import validation** | Yes | Same checks as desktop **Validate iPhone sync** before import |
| **Apple Music download** | Later (Phase D) | Requires mobile token UX + on-device decrypt; likely sideload/TestFlight first |
| **Tag Editor** | PC preferred | Light pre-download metadata on phone; bulk edit stays on desktop |
| **Trim / Split mix** | Desktop only | Copy downloads to PC for these workflows |
| **Share extension** | Phase C | Send YouTube / Apple Music URLs from Safari into Aura |

### Planned iOS app structure

Four tabs (see [ui-navigation.md](./docs/ios/ui-navigation.md)):

- **Apple** — paste URL, preview, download (when Apple mode is enabled)
- **YouTube** — URL, metadata/artwork editor, delivery mode, download
- **Downloads** — progress, history, **Add to Music**, **Show in Files**, import status
- **Settings** — token, storage, iCloud export, onboarding

PC-only UI (sync repair, cache purge, Requirements tool scan) will **not** appear on iOS.

### Rollout phases

| Phase | Delivers |
|-------|----------|
| **A** | YouTube download + tagged album folders + PC pickup |
| **B** | Guided Music import + pre-import validation + import state tracking |
| **C** | Share extension, background downloads, notifications |
| **D** | Apple Music AAC download on device |

### Distribution (expected)

- **App Store** — may limit YouTube download and Apple Music decrypt; details in [apple-music-ios-spike.md](./docs/ios/apple-music-ios-spike.md)
- **TestFlight / sideload** — target for full feature parity during development

### For developers

| Resource | Purpose |
|----------|---------|
| [docs/ios/](./docs/ios/) | Full design pack (pipeline, import UX, UI map) |
| [export-contract.json](./docs/ios/export-contract.json) | Machine-readable album layout v1 |
| [internal/crossdevice/contract.go](./internal/crossdevice/contract.go) | Shared naming/encoding constants |

**Status:** specification complete, **native iOS project not started** in this repository.

---

## Build from source

**Windows**

```powershell
.\scripts\build-windows.ps1 -SkipInstaller
```

Outputs: `dist\AuraAudioDownloader.exe`, `dist\aura.exe`, optional `dist\AuraAudioDownloader-Setup.exe` if Inno Setup is installed.

Place optional tools in `dist\tools\` before packaging: `MP4Box.exe`, `ffmpeg.exe`, `ffprobe.exe`, `yt-dlp.exe`, `mp4decrypt.exe`.

**Prerequisites:** Go 1.23+, Node.js 18+, [Wails CLI v2](https://wails.io/docs/gettingstarted/installation).

---

## Credits

Based on work by **Sorrow**, with fixes and improvements by contributors. See [README-CN.md](./README-CN.md) for the Chinese readme.
