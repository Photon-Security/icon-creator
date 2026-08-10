package iconcreator

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeGeneratedIconContainersSelectsLargestFrame(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	source := image.NewNRGBA(image.Rect(0, 0, CanvasSize, CanvasSize))
	for y := 0; y < CanvasSize; y++ {
		for x := 0; x < CanvasSize; x++ {
			alpha := uint8(255)
			if x >= 400 && x < 624 && y >= 400 && y < 624 {
				alpha = 96
			}
			source.SetNRGBA(x, y, color.NRGBA{R: 28, G: 180, B: 120, A: alpha})
		}
	}
	writeTestPNG(t, sourcePath, source)

	out, err := Create(Config{InputPath: sourcePath, OutputPath: filepath.Join(dir, "roundtrip.icns")})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		format   string
		wantSize int
		upperExt string
	}{
		{path: out.ICOPath, format: "ico", wantSize: 256, upperExt: ".ICO"},
		{path: out.ICNSPath, format: "icns", wantSize: 1024, upperExt: ".ICNS"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			upperPath := strings.TrimSuffix(test.path, filepath.Ext(test.path)) + test.upperExt
			if err := os.Rename(test.path, upperPath); err != nil {
				t.Fatal(err)
			}
			img, format, err := DecodeSource(upperPath)
			if err != nil {
				t.Fatal(err)
			}
			if format != test.format {
				t.Fatalf("format = %q, want %q", format, test.format)
			}
			if got := img.Bounds().Size(); got.X != test.wantSize || got.Y != test.wantSize {
				t.Fatalf("size = %dx%d, want %dx%d", got.X, got.Y, test.wantSize, test.wantSize)
			}
			config, configFormat, err := DecodeSourceConfig(upperPath)
			if err != nil {
				t.Fatal(err)
			}
			if configFormat != test.format || config.Width != test.wantSize || config.Height != test.wantSize {
				t.Fatalf("config = %dx%d %q, want %dx%d %q", config.Width, config.Height, configFormat, test.wantSize, test.wantSize, test.format)
			}
			_, _, _, alpha := img.At(test.wantSize/2, test.wantSize/2).RGBA()
			if alpha == 0 || alpha == 0xffff {
				t.Fatalf("alpha = %d, want preserved partial transparency", alpha)
			}
		})
	}
}

func TestDecodeSourceRejectsCorruptIconContainers(t *testing.T) {
	for _, ext := range []string{".ico", ".icns"} {
		t.Run(ext, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "corrupt"+ext)
			if err := os.WriteFile(path, []byte("not-an-icon"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := DecodeSource(path); err == nil {
				t.Fatal("expected corrupt container to fail")
			}
		})
	}
}

func TestDecodeSourceBMPBackedICO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bitmap.ico")
	if err := os.WriteFile(path, bmpBackedICO(), 0600); err != nil {
		t.Fatal(err)
	}
	img, format, err := DecodeSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if format != "ico" || img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Fatalf("decoded = %dx%d %q, want 2x2 ico", img.Bounds().Dx(), img.Bounds().Dy(), format)
	}
	_, _, _, alpha := img.At(0, 1).RGBA()
	if alpha != 0 {
		t.Fatalf("masked pixel alpha = %d, want 0", alpha)
	}
}

func TestDecodeSourceExplainsJPEG2000OnlyICNS(t *testing.T) {
	var body bytes.Buffer
	body.WriteString("ic10")
	data := []byte{0x00, 0x00, 0x00, 0x0c, 0x6a, 0x50, 0x20, 0x20, 0, 0, 0, 0}
	if err := binary.Write(&body, binary.BigEndian, uint32(8+len(data))); err != nil {
		t.Fatal(err)
	}
	body.Write(data)
	var file bytes.Buffer
	file.WriteString("icns")
	if err := binary.Write(&file, binary.BigEndian, uint32(8+body.Len())); err != nil {
		t.Fatal(err)
	}
	file.Write(body.Bytes())
	path := filepath.Join(t.TempDir(), "legacy.icns")
	if err := os.WriteFile(path, file.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	_, _, err := DecodeSource(path)
	if err == nil || !strings.Contains(err.Error(), "JPEG2000-only ICNS files are not supported") {
		t.Fatalf("error = %v, want clear JPEG2000-only explanation", err)
	}
}

func writeTestPNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func bmpBackedICO() []byte {
	const (
		width       = 2
		height      = 2
		rowBytes    = 8
		andRowBytes = 4
	)
	var dib bytes.Buffer
	binary.Write(&dib, binary.LittleEndian, uint32(40))
	binary.Write(&dib, binary.LittleEndian, int32(width))
	binary.Write(&dib, binary.LittleEndian, int32(height*2))
	binary.Write(&dib, binary.LittleEndian, uint16(1))
	binary.Write(&dib, binary.LittleEndian, uint16(24))
	binary.Write(&dib, binary.LittleEndian, uint32(0))
	binary.Write(&dib, binary.LittleEndian, uint32(rowBytes*height))
	binary.Write(&dib, binary.LittleEndian, int32(0))
	binary.Write(&dib, binary.LittleEndian, int32(0))
	binary.Write(&dib, binary.LittleEndian, uint32(0))
	binary.Write(&dib, binary.LittleEndian, uint32(0))
	dib.Write([]byte{0, 0, 255, 0, 255, 0, 0, 0})
	dib.Write([]byte{255, 0, 0, 255, 255, 255, 0, 0})
	dib.Write([]byte{0x80, 0, 0, 0})
	dib.Write(make([]byte, andRowBytes))

	var icoFile bytes.Buffer
	binary.Write(&icoFile, binary.LittleEndian, uint16(0))
	binary.Write(&icoFile, binary.LittleEndian, uint16(1))
	binary.Write(&icoFile, binary.LittleEndian, uint16(1))
	icoFile.Write([]byte{width, height, 0, 0})
	binary.Write(&icoFile, binary.LittleEndian, uint16(1))
	binary.Write(&icoFile, binary.LittleEndian, uint16(24))
	binary.Write(&icoFile, binary.LittleEndian, uint32(dib.Len()))
	binary.Write(&icoFile, binary.LittleEndian, uint32(22))
	icoFile.Write(dib.Bytes())
	return icoFile.Bytes()
}

func TestDecodeSourceWebP(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString("UklGRjIAAABXRUJQVlA4ICYAAACQAQCdASoCAAEAAgA0JYgCdLoAA5gA/us2/4oBSOHO8rH1KoAAAA==")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "source.webp")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	img, format, err := DecodeSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if format != "webp" {
		t.Fatalf("format = %q, want webp", format)
	}
	if got := img.Bounds().Size(); got.X != 2 || got.Y != 1 {
		t.Fatalf("size = %dx%d, want 2x1", got.X, got.Y)
	}
	config, configFormat, err := DecodeSourceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if configFormat != "webp" || config.Width != 2 || config.Height != 1 {
		t.Fatalf("config = %dx%d %q, want 2x1 webp", config.Width, config.Height, configFormat)
	}
}

func TestDecodeSourceExtendedLosslessWebPWithAlpha(t *testing.T) {
	// A 2x2 lossless image with alpha, wrapped in a VP8X container with XMP.
	// This is the same layout used by Pastebin_Ukraine.webp.
	bare, err := base64.StdEncoding.DecodeString("UklGRkgAAABXRUJQVlA4TDwAAAAvAUAAEB8gFED8cSIJph+HQATxV5lilzkuNW0bsCEY+cpiKmQbARLZPZpneyiB33NxS4BsgDJV2ET0Pz4=")
	if err != nil {
		t.Fatal(err)
	}
	if len(bare) < 12 || string(bare[12:16]) != "VP8L" {
		t.Fatal("invalid bare WebP test fixture")
	}

	var body bytes.Buffer
	body.WriteString("VP8X")
	binary.Write(&body, binary.LittleEndian, uint32(10))
	body.Write([]byte{0x14, 0, 0, 0, 1, 0, 0, 1, 0, 0})
	body.Write(bare[12:])
	body.WriteString("XMP ")
	binary.Write(&body, binary.LittleEndian, uint32(4))
	body.WriteString("test")

	var extended bytes.Buffer
	extended.WriteString("RIFF")
	binary.Write(&extended, binary.LittleEndian, uint32(4+body.Len()))
	extended.WriteString("WEBP")
	extended.Write(body.Bytes())

	path := filepath.Join(t.TempDir(), "extended-lossless.webp")
	if err := os.WriteFile(path, extended.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	img, format, err := DecodeSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if format != "webp" {
		t.Fatalf("format = %q, want webp", format)
	}
	if got := img.Bounds().Size(); got.X != 2 || got.Y != 2 {
		t.Fatalf("size = %dx%d, want 2x2", got.X, got.Y)
	}
	hasTransparency := false
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha < 0xffff {
				hasTransparency = true
			}
		}
	}
	if !hasTransparency {
		t.Fatal("expected WebP alpha to be preserved")
	}

	config, configFormat, err := DecodeSourceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if configFormat != "webp" || config.Width != 2 || config.Height != 2 {
		t.Fatalf("config = %dx%d %q, want 2x2 webp", config.Width, config.Height, configFormat)
	}
}

func TestDecodeSourceSVG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.svg")
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 100"><rect width="200" height="100" fill="#e3364a"/></svg>`
	if err := os.WriteFile(path, []byte(svg), 0600); err != nil {
		t.Fatal(err)
	}

	img, format, err := DecodeSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if format != "svg" {
		t.Fatalf("format = %q, want svg", format)
	}
	if got := img.Bounds().Size(); got.X != 2048 || got.Y != 1024 {
		t.Fatalf("raster size = %dx%d, want 2048x1024", got.X, got.Y)
	}
	_, _, _, alpha := img.At(1024, 512).RGBA()
	if alpha == 0 {
		t.Fatal("expected the SVG rectangle to be rendered")
	}
	config, configFormat, err := DecodeSourceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if configFormat != "svg" || config.Width != 2048 || config.Height != 1024 {
		t.Fatalf("config = %dx%d %q, want 2048x1024 svg", config.Width, config.Height, configFormat)
	}
}

func TestDecodeSourceSVGRequiresDimensions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.svg")
	if err := os.WriteFile(path, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`), 0600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := DecodeSource(path); err == nil {
		t.Fatal("expected an SVG without dimensions to fail")
	}
}
