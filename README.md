# Icon Creator

<p align="center">
  <img src="Icon.png" alt="Icon Creator app icon" width="220" />
</p>

Create polished app icons from a source image without leaving your Mac. Icon
Creator produces matching macOS `.icns`, Windows `.ico`, and PNG files with
live rounded-corner preview, zoom crop, drag-to-recenter positioning, optional
solid-background transparency, and automatic cleanup of temporary working files.

Developed by Florian Bidabe / Photon Security ([www.photonsec.com.au](https://www.photonsec.com.au))

[![Download](https://img.shields.io/github/v/release/Photon-Security/icon-creator?label=Download&style=for-the-badge)](https://github.com/Photon-Security/icon-creator/releases/latest)
[![Ko-fi](https://img.shields.io/badge/Support%20on-Ko--fi-ff5e5b?style=for-the-badge&logo=ko-fi&logoColor=white)](https://ko-fi.com/enelass)

![Demo](assets/Icon%20Creator%20Demo-small.gif)

## Install

1. Download the latest DMG from the [Releases page](https://github.com/Photon-Security/icon-creator/releases/latest).
2. Open the DMG and drag **Icon Creator.app** into your `/Applications` folder.
3. Launch the app and drop in a PNG, JPG, JPEG, or GIF source image.

### First launch: allow the app

The app is ad-hoc signed but not notarized with Apple, so macOS may block it the
first time. If that happens, click **Cancel**, then open **System Settings ->
Privacy & Security** and choose **Open Anyway** for Icon Creator.

If you prefer Terminal:

```bash
xattr -dr com.apple.quarantine "/Applications/Icon Creator.app"
```

## Use

Drop an image onto the preview area or click **Browse**. Use **Shape feel** to
control the rounded corners, **Zoom crop** to crop tighter, and drag the preview
to center the source image. Click **Create icons** to export matching `.icns`
`.ico`, and `.png` files beside the selected base output path.

Enable **Transparent outer color** when the source image has a solid white,
off-white, or otherwise flat background connected to the outer edge. Icon
Creator turns that connected outer color into alpha before applying the rounded
corner mask.

The normal app flow leaves only the finished `.icns`, `.ico`, and `.png` files.
Temporary `icon.png` and `.iconset` files are generated in a temp directory and
removed automatically unless **Keep working files** is enabled.

## Features

- Native macOS desktop app built with Go, Wails, and React
- Drag-and-drop source image selection
- Live rounded-corner overlay preview
- Zoom crop and drag-to-recenter controls
- Automatic `.icns`, `.ico`, and PNG export
- Optional connected solid-background removal to alpha
- Cleanup by default, with optional working-file retention
- CLI support for scripted icon generation

## Build

Requirements:

- macOS with `iconutil` and `sips`
- Go 1.22 or newer
- Node.js and npm
- Wails v2 installed at `$HOME/go/bin/wails`, or set `WAILS_BIN`

Build the macOS app and DMG:

```bash
./scripts/build_macos.sh
```

The packaged DMG is written to:

```text
dist/Icon-Creator-1.3.3-macOS-arm64.dmg
```

## CLI Usage

```bash
go run ./cmd/icon-creator -input Icon.png -output app.icns -radius 220 -zoom 1.4 -pan-x 20 -pan-y -10
```

Use `-keep-intermediates` when you need the generated `icon.png` and `.iconset`
folder for inspection.

Use `-transparent-background` to turn a solid connected outer color into alpha.

`-pan-x` and `-pan-y` accept values from `-100` to `100` and are useful after
zooming in to recenter the source image.

## Support

If this tool saved you time, consider buying me a coffee on Ko-fi:
**[ko-fi.com/enelass](https://ko-fi.com/enelass)**
