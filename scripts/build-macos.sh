#!/usr/bin/env bash
# Build Aura Audio Downloader for macOS (Apple Silicon arm64).
# Produces dist/AuraAudioDownloader.app and dist/AuraAudioDownloader-arm64.dmg
#
# Optional tools in dist/tools/ before packaging:
#   MP4Box, ffmpeg, ffprobe, yt-dlp

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
TOOLS="$DIST/tools"
GUI="$ROOT/gui"
SKIP_FRONTEND=0
SKIP_DMG=0

for arg in "$@"; do
  case "$arg" in
    --skip-frontend) SKIP_FRONTEND=1 ;;
    --skip-dmg) SKIP_DMG=1 ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  echo "Go not found. Install from https://go.dev/dl/" >&2
  exit 1
fi

if ! command -v wails >/dev/null 2>&1; then
  GOPATH="$(go env GOPATH)"
  if [[ -x "$GOPATH/bin/wails" ]]; then
    export PATH="$GOPATH/bin:$PATH"
  else
    echo "Wails CLI not found. Run: go install github.com/wailsapp/wails/v2/cmd/wails@latest" >&2
    exit 1
  fi
fi

mkdir -p "$DIST" "$TOOLS"

if [[ "$SKIP_FRONTEND" -eq 0 ]]; then
  echo "Building frontend..."
  pushd "$GUI/frontend" >/dev/null
  if [[ ! -d node_modules ]]; then npm install; fi
  npm run build
  popd >/dev/null
fi

echo "Building CLI (aura)..."
go build -ldflags="-s -w" -o "$DIST/aura" "$ROOT/cmd/amd"

echo "Building GUI (Wails darwin/arm64)..."
pushd "$GUI" >/dev/null
WAILS_ARGS=(build -platform darwin/arm64 -clean)
if [[ "$SKIP_FRONTEND" -eq 0 ]]; then
  WAILS_ARGS+=(-s)
fi
wails "${WAILS_ARGS[@]}"
popd >/dev/null

APP_SRC=""
for candidate in \
  "$GUI/build/bin/AuraAudioDownloader.app" \
  "$ROOT/build/bin/AuraAudioDownloader.app"; do
  if [[ -d "$candidate" ]]; then
    APP_SRC="$candidate"
    break
  fi
done

if [[ -z "$APP_SRC" ]]; then
  echo "Wails build did not produce AuraAudioDownloader.app" >&2
  exit 1
fi

APP_DST="$DIST/AuraAudioDownloader.app"
rm -rf "$APP_DST"
cp -R "$APP_SRC" "$APP_DST"

MACOS_BIN="$APP_DST/Contents/MacOS"
TOOLS_DST="$MACOS_BIN/tools"
mkdir -p "$TOOLS_DST"

for tool in MP4Box ffmpeg ffprobe yt-dlp; do
  if [[ -f "$TOOLS/$tool" ]]; then
    cp "$TOOLS/$tool" "$TOOLS_DST/$tool"
    chmod +x "$TOOLS_DST/$tool"
  fi
done

if [[ ! -f "$TOOLS_DST/MP4Box" ]] && ! command -v MP4Box >/dev/null 2>&1; then
  echo "Warning: MP4Box not bundled in dist/tools/ and not on PATH — tagging may fail until gpac is installed."
fi

for tool in ffmpeg ffprobe yt-dlp; do
  if [[ ! -f "$TOOLS_DST/$tool" ]] && ! command -v "$tool" >/dev/null 2>&1; then
    echo "Warning: $tool not bundled and not on PATH — YouTube downloads need Homebrew: brew install ffmpeg yt-dlp"
  fi
done

if [[ "$SKIP_DMG" -eq 0 ]]; then
  bash "$ROOT/installer/macos/create-dmg.sh" "$APP_DST" "$DIST/AuraAudioDownloader-arm64.dmg"
fi

echo ""
echo "Build complete:"
echo "  App: $APP_DST"
echo "  CLI: $DIST/aura"
if [[ "$SKIP_DMG" -eq 0 ]]; then
  echo "  DMG: $DIST/AuraAudioDownloader-arm64.dmg"
fi
