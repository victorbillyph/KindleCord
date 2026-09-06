#!/bin/sh
# KindleCord one-line installer for Kindle (run via SSH on the Kindle)
# Usage: curl -fsSL https://raw.githubusercontent.com/victorbillyph/KindleCord/main/scripts/install_kindle.sh | sh

set -e

REPO="victorbillyph/KindleCord"
EXT_DIR="/mnt/us/extensions/KindleCord"
TMP_DIR="/var/tmp/kc-install"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"

log() { echo "[KindleCord] $*"; }

log "Fetching latest release info..."
RELEASE_JSON=$(curl -fsSL "$GITHUB_API")
ZIP_URL=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url": "[^"]*KindleCord\.zip"' | cut -d'"' -f4)
VERSION=$(echo "$RELEASE_JSON" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)

if [ -z "$ZIP_URL" ]; then
    log "ERROR: Could not find KindleCord.zip in latest release"
    exit 1
fi

log "Latest version: $VERSION"
log "Downloading: $ZIP_URL"

mkdir -p "$TMP_DIR"
cd "$TMP_DIR"

curl -fL -o KindleCord.zip "$ZIP_URL"

log "Extracting..."
rm -rf "$EXT_DIR"
mkdir -p "$EXT_DIR"
unzip -q -o KindleCord.zip -d /mnt/us/extensions/

log "Setting permissions..."
chmod +x "$EXT_DIR/kindlecord"
chmod +x "$EXT_DIR/bin/"*.sh 2>/dev/null || true
chmod +x "$EXT_DIR/bin/fbink" 2>/dev/null || true

log "Cleaning up..."
rm -rf "$TMP_DIR"

log "Done! KindleCord $VERSION installed to $EXT_DIR"
log "Open KUAL → KindleCord to launch, or run:"
log "  $EXT_DIR/bin/start.sh"