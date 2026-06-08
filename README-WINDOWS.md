# Apple Music Downloader — Windows

## Quick start (installer)

1. Download and run `AppleMusicDownloader-Setup.exe`
2. Launch **Apple Music Downloader** from the Start Menu or desktop shortcut
3. Complete the first-run wizard (storefront, optional `media-user-token`, output folder)
4. Paste an Apple Music URL, choose **AAC**, and click **Start download**

No Go toolchain, MP4Box PATH setup, or manual `config.yaml` editing required.

## What works out of the box

| Feature | Requires |
|---------|----------|
| AAC downloads | Apple Music subscription |
| Lyrics (LRC) | `media-user-token` in Settings |
| Music videos | `media-user-token` + bundled `mp4decrypt` |
| ALAC / Dolby Atmos | Manual **wrapper** setup (see below) |

## Bundled tools

The installer includes:

- `AppleMusicDownloader.exe` — GUI application
- `amd.exe` — CLI (same engine, for power users)
- `tools/MP4Box.exe` — tagging (GPAC)
- `tools/ffmpeg.exe` — optional conversion / animated artwork
- `tools/mp4decrypt.exe` — music video decryption

Configuration is stored in:

```
%APPDATA%\AppleMusicDownloader\config.yaml
```

## ALAC / Dolby Atmos (wrapper)

The decrypt **wrapper** service is Linux-only. On Windows it typically runs under **WSL2**.

1. Install [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install) (Ubuntu recommended)
2. Download [WorldObservationLog/wrapper](https://github.com/WorldObservationLog/wrapper/releases) for Linux x86_64
3. Start wrapper inside WSL:
   ```bash
   ./wrapper -H 0.0.0.0 -D 10020 -M 20020
   ```
4. In the app **Settings → Advanced**, ensure ports match:
   - `decrypt-m3u8-port`: `127.0.0.1:10020`
   - `get-m3u8-port`: `127.0.0.1:20020`
5. Use **Settings → Dependencies → Test** — wrapper rows should show **OK**
6. Select **Lossless (ALAC)** or **Dolby Atmos** on the Download tab

Until wrapper is running, use **AAC** quality — it works without wrapper via in-process Widevine.

## Building from source

Requirements:

- Go 1.23+
- Node.js 18+ (frontend)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)
- Optional: [Inno Setup 6](https://jrsoftware.org/isinfo.php) for the installer

```powershell
.\scripts\build-windows.ps1
```

Output:

- `dist/AppleMusicDownloader.exe`
- `dist/amd.exe`
- `dist/tools/` (place MP4Box, ffmpeg, mp4decrypt binaries before packaging)
- `dist/AppleMusicDownloader-Setup.exe` (if Inno Setup is installed)

### Third-party binaries

Download and place in `dist/tools/` before running the build script:

| File | Source |
|------|--------|
| `MP4Box.exe` | [GPAC nightly builds](https://gpac.io/downloads/gpac-nightly-builds/) |
| `ffmpeg.exe` | [ffmpeg.org](https://ffmpeg.org/download.html) (essentials build) |
| `mp4decrypt.exe` | [Bento4](https://www.bento4.com/downloads/) |

## CLI fallback

```powershell
& "$env:ProgramFiles\AppleMusicDownloader\amd.exe" --aac "https://music.apple.com/us/album/..."
```

Or from a dev build:

```powershell
go run ./cmd/amd --aac "https://music.apple.com/..."
```

## Troubleshooting

- **App crashed during download** — open the log file at `%APPDATA%\AppleMusicDownloader\logs\app.log` (Queue tab → **Open log file**)
- **Failed to get token** — check internet; optionally set `authorization-token` in Settings → Advanced
- **mp4decrypt not found** — reinstall; verify `tools\mp4decrypt.exe` exists next to the app
- **Lyrics failed** — verify `media-user-token` and matching **storefront** (e.g. `jp` for Japan accounts)
- **ALAC unavailable** — wrapper not running; fall back to AAC or complete WSL wrapper setup above
