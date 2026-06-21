package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"iconcreator/internal/iconcreator"
)

func TestCreateIconCleansIntermediatesByDefault(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "source.png")
	outputPath := filepath.Join(tempDir, "app.icns")
	if err := writeTestSource(inputPath); err != nil {
		t.Fatal(err)
	}

	out, err := iconcreator.Create(iconcreator.Config{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Radius:     220,
		Zoom:       1.8,
		PanX:       45,
		PanY:       -30,
	})
	if err != nil {
		t.Fatal(err)
	}

	if out.ICNSPath != outputPath {
		t.Fatalf("ICNSPath = %q, want %q", out.ICNSPath, outputPath)
	}
	if info, err := os.Stat(outputPath); err != nil {
		t.Fatalf("missing icns: %v", err)
	} else if info.Size() == 0 {
		t.Fatal("icns is empty")
	}
	if out.ICOPath != filepath.Join(tempDir, "app.ico") {
		t.Fatalf("ICOPath = %q, want %q", out.ICOPath, filepath.Join(tempDir, "app.ico"))
	}
	assertICO(t, out.ICOPath, 7)
	if out.PNGPath != filepath.Join(tempDir, "app.png") {
		t.Fatalf("PNGPath = %q, want %q", out.PNGPath, filepath.Join(tempDir, "app.png"))
	}
	assertPNG(t, out.PNGPath, 1024, 1024)
	if _, err := os.Stat(out.WorkingDir); !os.IsNotExist(err) {
		t.Fatalf("working dir still exists: %s", out.WorkingDir)
	}
	if entries, err := os.ReadDir(tempDir); err != nil {
		t.Fatal(err)
	} else if len(entries) != 4 {
		t.Fatalf("expected only source.png, app.icns, app.ico, and app.png, got %d entries", len(entries))
	}
}

func TestCreateIconCanKeepIntermediates(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "source.png")
	outputPath := filepath.Join(tempDir, "test-app.icns")
	if err := writeTestSource(inputPath); err != nil {
		t.Fatal(err)
	}

	out, err := iconcreator.Create(iconcreator.Config{
		InputPath:         inputPath,
		OutputPath:        outputPath,
		Radius:            220,
		Zoom:              2.2,
		PanX:              -60,
		PanY:              40,
		KeepIntermediates: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertPNG(t, out.NormalizedPNG, 1024, 1024)
	assertTransparentPixel(t, out.NormalizedPNG, 0, 0)

	for _, spec := range iconcreator.IconSpecs() {
		assertPNG(t, filepath.Join(out.IconsetDir, spec.FileName), spec.Size, spec.Size)
	}
	assertICO(t, out.ICOPath, 7)
	assertPNG(t, out.PNGPath, 1024, 1024)
}

func TestCreateIconCanTurnSolidOuterBackgroundTransparent(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "source.png")
	outputPath := filepath.Join(tempDir, "transparent.icns")
	if err := writeWhiteBackgroundSource(inputPath); err != nil {
		t.Fatal(err)
	}

	out, err := iconcreator.Create(iconcreator.Config{
		InputPath:     inputPath,
		OutputPath:    outputPath,
		Radius:        0,
		TransparentBg: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertTransparentPixel(t, out.PNGPath, 0, 0)
	assertOpaquePixel(t, out.PNGPath, 512, 512)
}

func TestSanitizeName(t *testing.T) {
	if got := iconcreator.SanitizeName("My App!.icns"); got != "My-App" {
		t.Fatalf("SanitizeName() = %q, want %q", got, "My-App")
	}
}

func writeTestSource(path string) error {
	img := image.NewNRGBA(image.Rect(0, 0, 1254, 1254))
	for y := 0; y < 1254; y++ {
		for x := 0; x < 1254; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: 180,
				A: 255,
			})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeWhiteBackgroundSource(path string) error {
	img := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			c := color.NRGBA{R: 248, G: 248, B: 246, A: 255}
			if x >= 36 && x < 92 && y >= 36 && y < 92 {
				c = color.NRGBA{R: 210, G: 54, B: 62, A: 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func assertPNG(t *testing.T, path string, width int, height int) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	b := img.Bounds()
	if b.Dx() != width || b.Dy() != height {
		t.Fatalf("%s size = %dx%d, want %dx%d", path, b.Dx(), b.Dy(), width, height)
	}
}

func assertTransparentPixel(t *testing.T, path string, x int, y int) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	_, _, _, alpha := img.At(x, y).RGBA()
	if alpha != 0 {
		t.Fatalf("%s pixel %d,%d alpha = %d, want 0", path, x, y, alpha)
	}
}

func assertOpaquePixel(t *testing.T, path string, x int, y int) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	_, _, _, alpha := img.At(x, y).RGBA()
	if alpha == 0 {
		t.Fatalf("%s pixel %d,%d alpha = %d, want opaque", path, x, y, alpha)
	}
}

func assertICO(t *testing.T, path string, count uint16) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ico: %v", err)
	}
	if len(data) < 6 {
		t.Fatalf("ico too small: %d bytes", len(data))
	}
	if data[0] != 0 || data[1] != 0 || data[2] != 1 || data[3] != 0 {
		t.Fatalf("unexpected ico header: %v", data[:4])
	}
	gotCount := uint16(data[4]) | uint16(data[5])<<8
	if gotCount != count {
		t.Fatalf("ico count = %d, want %d", gotCount, count)
	}
}
