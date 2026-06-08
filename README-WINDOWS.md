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
2. Load a track template (e.g. `secret_sky_2020`) or add tracks manually
3. Drag cut lines on the waveform (or use **Fit durations to master**)
4. **Export tracks** — AAC 256 kbps files with album tags, ready for Apple Music sync

## Troubleshooting

- **Missing tools** — copy binaries into `dist\tools\` or install on PATH; check the Requirements tab
- **App crashed** — open `%APPDATA%\AuraAudioDownloader\logs\app.log` (Activity tab → Open log file)
