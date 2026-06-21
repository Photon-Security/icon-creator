#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="Icon Creator"
VERSION="1.3.4"
BUILD_DIR="$ROOT_DIR/build"
DIST_DIR="$ROOT_DIR/dist"
APP_ICON_ICNS="$BUILD_DIR/appicon.icns"
APP_ICON_PNG="$BUILD_DIR/appicon.png"
WAILS_BIN="${WAILS_BIN:-$HOME/go/bin/wails}"
DMG_ROOT=""

cleanup() {
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

if [[ ! -f "$APP_ICON_ICNS" ]]; then
  echo "Missing $APP_ICON_ICNS" >&2
  exit 1
fi

if [[ ! -f "$APP_ICON_PNG" ]]; then
  echo "Missing $APP_ICON_PNG" >&2
  exit 1
fi

mkdir -p "$BUILD_DIR" "$DIST_DIR"

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

cp "$APP_ICON_ICNS" "$APP_FROM/Contents/Resources/iconfile.icns"

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
