# Aura Audio Downloader — Windows guide

This guide covers everyday use on Windows: installing the app, working through each tab, and fixing common sync and artwork issues.

For a feature overview across platforms, see **[README.md](./README.md)**.

---

## Install

1. Run **`AuraAudioDownloader-Setup.exe`** (or build with `.\scripts\build-windows.ps1`).
2. Launch **Aura Audio Downloader** from the Start Menu or desktop shortcut.
3. Complete the setup wizard:
   - Apple Music **storefront**
   - **`media-user-token`** (required for AAC downloads)
   - Default **download folder**
4. Open **Requirements** and confirm tools show **Detected** (MP4Box, ffmpeg, yt-dlp are bundled with the installer when present in `dist\tools\`).

### Files after install

| Path | Purpose |
|------|---------|
| `AuraAudioDownloader.exe` | Desktop app |
| `aura.exe` / `amd.exe` | CLI (same engine) |
| `tools\` | MP4Box, ffmpeg, ffprobe, yt-dlp, etc. |

### Config

```
%APPDATA%\AuraAudioDownloader\config.yaml
%APPDATA%\AuraAudioDownloader\logs\app.log
```

Older settings under `%APPDATA%\AppleMusicDownloader\` migrate on first launch.

---

## Using the app

### Apple Music tab

1. Paste an album, song, or playlist URL.
2. Choose **AAC** (recommended) or lossless / Atmos if wrapper is configured.
3. **Fetch** → review tracks → **Start download**.

**Catalog lookup:** search Apple Music, paste URLs, or paste a **Spotify track link** (`open.spotify.com/track/…`) to find the Apple Music equivalent. Playlists and Spotify albums are not supported — use individual track links.

**Bulk queue:** add multiple URLs or search hits. For playlists, expand the preview and check only the tracks you want.

### YouTube tab

1. Paste a YouTube video or playlist URL.
2. Pick delivery: **Audio only**, **Audio + video**, or **Video only**.
3. Adjust metadata (title, artist, album, cover) before downloading.
4. Choose output location if you use separate YouTube folders in Settings.

When the job finishes:

- **Split into tracks** → **Split mix** with the master file loaded.
- **Trim dead space** → **Trim** with the file loaded.

### Split mix tab

For long recordings split into many songs:

1. Open a master `.m4a` (from YouTube or disk).
2. Paste a timestamped tracklist, e.g.  
   `0:00 Artist - Title`  
   `5:42 Next Artist - Next Song`
3. Drag boundaries on the waveform; preview segments.
4. **Fit durations to master** if the last tracks run long or short.
5. Set album name, artist, artwork, output folder.
6. **Export tracks** — AAC 256 kbps files with full tags.

Save projects to resume later.

### Trim tab

For a **single** cut (remove silence at start/end):

1. **Open file** or arrive via **Trim dead space** / Tag Editor **Open in Trim**.
2. Preview `.m4a` (audio player) or `.mp4` (video player).
3. Drag start/end on the waveform; nudge with time fields.
4. **Save as new** — choose output path.
5. **Save** — backs up the original as `.bak`, then replaces it.

Minimum trimmed length: 1 second. Tags can be preserved on export.

### Tag Editor tab

- Open one file or use **Album bulk** for every track in a folder.
- Edit metadata and artwork; drag files onto the tab to open.
- **Open in Trim** when you only need to shorten the file.
- **Sync repair tools** (bottom) — for iPhone artwork issues (below).

### Activity, Requirements, Settings

- **Activity** — job status, failed tracks, cancel downloads, open log.
- **Requirements** — tool detection and per-feature notes.
- **Settings** — token, folders, lyrics/cover options, YouTube paths, Reset Apple sync, full artwork reset guide.

---

## Requirements by feature

| Feature | What you need |
|---------|----------------|
| AAC downloads | Subscription + `media-user-token` + MP4Box |
| YouTube | yt-dlp + ffmpeg (in `tools\` or PATH) |
| Split mix | ffmpeg + ffprobe |
| Trim | ffmpeg |
| ALAC / Atmos | Wrapper decrypt service (see Requirements tab) |
| Tag editor | MP4Box for reliable embeds |

Refresh **Requirements** after copying tools into `dist\tools\`.

---

## Troubleshooting

### Missing tools

Copy binaries into `dist\tools\` or add them to PATH, then **Requirements → Refresh**:

- `MP4Box.exe` — tagging AAC
- `ffmpeg.exe`, `ffprobe.exe` — YouTube, Split mix, Trim
- `yt-dlp.exe` — YouTube
- `mp4decrypt.exe` — optional, music video decryption

### App crashed or download failed

Open the log:

```
%APPDATA%\AuraAudioDownloader\logs\app.log
```

Or **Activity → Open log file**.

**AAC license / auth errors:** refresh `media-user-token` from music.apple.com cookies.

### iPhone artwork missing (album art OK, track art blank)

Apple Music on PC can show tiles from `cover.jpg` or cache. **iPhone sync needs the same JPEG embedded in every track** (`covr`), not just a sidecar file.

**Tag Editor → Sync repair tools:**

1. **Check folder** — read-only scan.
2. **Reset Apple sync** — stops Apple Music and sync agents, restarts USB service (like canceling a restart). Does not delete caches.
3. **Update album artwork** — embeds one shared JPEG per track; preserves titles and track numbers.

**If the iPhone still shows wrong art:** delete the album on the phone, then re-sync that album only in Apple Devices. PC tools cannot reset device artwork cache.

**Artwork only appears after restarting Windows:** use **Settings → Reset Apple sync** (or Tag Editor step 2) instead of rebooting.

After any PC fix: remove albums from Apple Music library (keep files on disk), re-import, delete on iPhone, sync selected albums first.

### Art missing everywhere (PC + iPhone)

**Settings → Full artwork reset** walks through resetting PC library references and iPhone caches without deleting your Aura download folders.

---

## Build from source

```powershell
.\scripts\build-windows.ps1
```

| Output | Description |
|--------|-------------|
| `dist\AuraAudioDownloader.exe` | GUI |
| `dist\aura.exe` | CLI |
| `dist\AuraAudioDownloader-Setup.exe` | Installer (needs Inno Setup 6) |

Skip installer only:

```powershell
.\scripts\build-windows.ps1 -SkipInstaller
```

**ALAC / Atmos:** see the Requirements tab and wrapper setup notes in the project for the decrypt service — AAC does not need wrapper in the GUI.
