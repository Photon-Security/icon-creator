#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="Icon Creator"
VERSION="1.3.2"
BUILD_DIR="$ROOT_DIR/build"
DIST_DIR="$ROOT_DIR/dist"
SOURCE_ICON="$ROOT_DIR/Icon.png"
WAILS_BIN="${WAILS_BIN:-$HOME/go/bin/wails}"
TMP_ICON_DIR=""
DMG_ROOT=""

cleanup() {
  if [[ -n "$TMP_ICON_DIR" ]]; then
    rm -rf "$TMP_ICON_DIR"
  fi
  if [[ -n "$DMG_ROOT" ]]; then
    rm -rf "$DMG_ROOT"
  fi
  rm -rf "$BUILD_DIR/bin" "$BUILD_DIR/app-icon" "$BUILD_DIR/dmg-root" "$BUILD_DIR/test-output"
  rm -f "$BUILD_DIR/darwin/iconfile.icns"
}
trap cleanup EXIT

if [[ ! -x "$WAILS_BIN" ]]; then
  echo "wails is required but was not found at $WAILS_BIN" >&2
  exit 1
fi

if [[ ! -f "$SOURCE_ICON" ]]; then
  echo "Missing $SOURCE_ICON" >&2
  exit 1
fi

mkdir -p "$BUILD_DIR" "$DIST_DIR"

TMP_ICON_DIR="$(mktemp -d "${TMPDIR:-/tmp}/icon-creator-appicon.XXXXXX")"
(cd "$ROOT_DIR" && go run ./cmd/icon-creator -input "$SOURCE_ICON" -output "$TMP_ICON_DIR/AppIcon.icns" -keep-intermediates -quiet >/dev/null)
cp "$TMP_ICON_DIR/AppIcon-icon-work/icon.png" "$BUILD_DIR/appicon.png"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

(cd "$ROOT_DIR" && "$WAILS_BIN" build -clean -platform darwin/arm64 -trimpath -ldflags="-s -w" -o "$APP_NAME")

APP_FROM="$BUILD_DIR/bin/$APP_NAME.app"
if [[ ! -d "$APP_FROM" && -d "$BUILD_DIR/bin/IconCreator.app" ]]; then
  APP_FROM="$BUILD_DIR/bin/IconCreator.app"
fi
APP_DIR="$DIST_DIR/$APP_NAME.app"
if [[ ! -d "$APP_FROM" ]]; then
  echo "Wails did not produce $APP_FROM" >&2
  exit 1
fi

cp -R "$APP_FROM" "$APP_DIR"

if command -v codesign >/dev/null 2>&1; then
  codesign --force --deep --sign - "$APP_DIR" >/dev/null
fi

DMG_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/icon-creator-dmg.XXXXXX")"
cp -R "$APP_DIR" "$DMG_ROOT/"
ln -s /Applications "$DMG_ROOT/Applications"

hdiutil create \
  -volname "$APP_NAME" \
  -srcfolder "$DMG_ROOT" \
  -ov \
  -format UDZO \
  "$DIST_DIR/Icon-Creator-$VERSION-macOS-arm64.dmg" >/dev/null

echo "$DIST_DIR/Icon-Creator-$VERSION-macOS-arm64.dmg"
