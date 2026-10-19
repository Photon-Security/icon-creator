# Changelog

All notable changes to Icon Creator are documented here.

## v1.3.11 - 2026-10-19

- Added clipboard image paste with **⌘V** on macOS and **Ctrl+V** on Windows.
- Pasted images use the same preview and icon export controls as browsed or dropped images.

## v1.3.10 - 2026-08-10

- Fixed valid extended lossless WebP sources with transparency being rejected as an invalid format.

## v1.3.9 - 2026-08-08

- Added aspect-ratio-aware zoom-out for non-square source images.
- Added an exact-fit zoom floor with transparent letterboxing so the entire source can remain visible.
- Fixed the live preview to match exported crop, scale, and pan behavior for wide and tall images.
- Enabled panning on any overflowing axis, including wide images at 100% zoom.

## v1.3.8 - 2026-08-08

- Added an in-app **About** dialog with the author, version, Photon Security website, and donation link.
- Added ICO and ICNS source imports so existing icons can be edited and re-exported.
- Automatically selects the largest supported icon frame and preserves imported transparency.
- Uses a protected `<name>-edited` output name by default when opening an ICO or ICNS file.

## v1.3.7 - 2026-08-08

- Added a compiled Windows application to releases alongside the macOS app.
- Replaced the macOS-only ICNS conversion step with a portable pure-Go writer.
- Fixed **Reveal** so generated files open correctly in Windows Explorer.

## v1.3.6 - 2026-08-06

- Added WebP and SVG source image support to the desktop app and CLI.

## v1.3.5 - 2026-06-29

- Fixed the packaged macOS app icon rendering with an unwanted grey outer layer.
- Regenerated the app icon source assets so Finder and Launch Services display the intended artwork.

## v1.3.4 - 2026-06-21

- Changed the packaged app icon to the provided `app.icns` artwork.
- Added the source `.icns` app icon asset to the repository.

## v1.3.3 - 2026-06-21

- Added final PNG export beside every `.icns` and `.ico` output.
- Added **Transparent outer color** to turn a solid connected edge background into alpha.
- Added CLI `-transparent-background` support.
- Updated result reporting to show all three generated files.

## v1.3.2 - 2026-06-21

- Added the first public macOS DMG release.
- Added a Wails desktop UI for creating icons from source images.
- Added live rounded-corner preview with shape, zoom, and drag-to-recenter controls.
- Added matching macOS `.icns` and Windows `.ico` export.
- Added automatic cleanup of temporary `icon.png` and `.iconset` working files.
- Added optional **Keep working files** mode for inspection and debugging.
- Added CLI support for scripted icon generation.
