#!/usr/bin/env bash
# Create an unsigned DMG from AuraAudioDownloader.app
# Usage: create-dmg.sh /path/to/AuraAudioDownloader.app /path/to/output.dmg

set -euo pipefail

APP_PATH="${1:?app path required}"
DMG_PATH="${2:?output dmg path required}"
STAGING="$(mktemp -d)"
trap 'rm -rf "$STAGING"' EXIT

mkdir -p "$STAGING/dmg-root"
cp -R "$APP_PATH" "$STAGING/dmg-root/"
ln -s /Applications "$STAGING/dmg-root/Applications"

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
if [[ -f "$ROOT_DIR/README-macOS.md" ]]; then
  cp "$ROOT_DIR/README-macOS.md" "$STAGING/dmg-root/README-macOS.txt"
fi

rm -f "$DMG_PATH"
hdiutil create \
  -volname "Aura Audio Downloader" \
  -srcfolder "$STAGING/dmg-root" \
  -ov \
  -format UDZO \
  "$DMG_PATH"

echo "Created $DMG_PATH"
