# Aura Audio Downloader — Windows

## Install

1. Download and run `AuraAudioDownloader-Setup.exe`
2. Launch **Aura Audio Downloader** from the Start Menu or desktop shortcut
3. Complete the setup wizard (Apple Music token for AAC downloads, output folders)
4. Paste an Apple Music or YouTube URL on the **Download** tab, or open a long recording on **Split mix**

## What you need

| Feature | Requirements |
|---------|----------------|
| AAC downloads | Apple Music subscription + `media-user-token` in Settings |
| YouTube audio | yt-dlp + ffmpeg + ffprobe (bundled in `tools/` or on PATH) |
| Split mix | ffmpeg + ffprobe (same bundle) |
| ALAC / Atmos | Wrapper decrypt service (see Requirements tab) |

## Files

- `AuraAudioDownloader.exe` — desktop app
- `aura.exe` — CLI (same as legacy `amd.exe`)
- `tools/` — MP4Box, ffmpeg, yt-dlp, etc.

## Config

```
%APPDATA%\AuraAudioDownloader\config.yaml
```

Settings from the old **Apple Music Downloader** folder are migrated automatically on first launch.

## Build from source

```powershell
.\scripts\build-windows.ps1
```

Outputs:

- `dist/AuraAudioDownloader.exe`
- `dist/aura.exe` and `dist/amd.exe`
- `dist/AuraAudioDownloader-Setup.exe` (if Inno Setup is installed)

## Split mix workflow

1. Download a YouTube DJ set (YouTube mode on Download tab), or open an existing `.m4a` on **Split mix**
2. Paste a timestamped tracklist, or add tracks manually and drag cut lines on the waveform
3. Use **Fit durations to master** if timings drift
4. **Export tracks** — AAC 256 kbps files with album tags, ready for Apple Music sync

## Troubleshooting

- **Missing tools** — copy binaries into `dist\tools\` or install on PATH; check the Requirements tab
- **App crashed** — open `%APPDATA%\AuraAudioDownloader\logs\app.log` (Activity tab → Open log file)

### iPhone artwork missing (album art OK, track art blank)

Apple Music on PC can show album tiles from `cover.jpg` or its artwork cache. **iPhone sync needs the same normalized JPEG embedded in every track** (`covr` atom), not just a sidecar file.

**Repair workflow (Tag Editor → expand Sync repair tools at the bottom):**

Actions are ordered **safest first**. Nothing here modifies your iPhone library directly.

1. **Check folder** — read-only; scans files in that folder only.
2. **Clear PC caches** — deletes Apple Music artwork cache on this PC; **never changes .m4a files**. Quit Apple Music first. Re-import albums afterward.
3. **Update album artwork** — embeds one shared JPEG per track in that folder; **preserves title, artist, and track numbers**; skips tracks already correct. Quit Apple Music first.
4. **Heavy repair** (collapsed) — same as step 3 across all download folders, then cache clear. Last resort only.

**iPhone wrong art:** delete the album on the phone, then re-sync that album only in Apple Devices — PC repairs cannot reset device artwork cache.

After any PC fix: remove albums from Apple Music library (keep files on disk), re-import, delete on iPhone, sync selected albums first.

For normal tag edits, use **Save tags** or **Album bulk** — those do not run the bulk artwork rewrite unless you explicitly use Sync repair tools.

The app cannot reset the iPhone music database from the PC — deleting music on the device and re-syncing is required once stale entries exist.

### Art missing everywhere (PC + iPhone) but files still have art

Apple Music keeps **three** artwork layers: embedded tags in each `.m4a`, a PC thumbnail index, and an iPhone grid cache. They drift apart — especially after cache clears without removing library entries. File properties can show correct art while both apps show blank tiles.

**Full reset (Settings → Full artwork reset):**

1. **PC** — Delete albums from Apple Music library (Keep Files) → Tag Editor: clear art cache and embed JPEG art per album folder → re-import.
2. **iPhone** — Turn off Sync Library → delete affected albums from Music → force-close app → sync **one album** from PC first → re-enable Sync Library.

Your Aura download folders are never deleted — only Apple’s caches and library references are reset.
