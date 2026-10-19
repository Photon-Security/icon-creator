package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"iconcreator/internal/iconcreator"
)

func TestPreviewDataURLUsesInputMIMEType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preview")
	if err := os.WriteFile(path, []byte("preview-data"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		format string
		prefix string
	}{
		{format: "webp", prefix: "data:image/webp;base64,"},
		{format: "svg", prefix: "data:image/svg+xml;base64,"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			got, err := previewDataURL(path, test.format)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(got, test.prefix) {
				t.Fatalf("preview data URL = %q, want prefix %q", got, test.prefix)
			}
		})
	}
}

func TestIconContainerPreviewUsesPNGDataURL(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	f, err := os.Create(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 30, G: 160, B: 220, A: uint8(100 + x)})
		}
	}
	if err := png.Encode(f, source); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	out, err := iconcreator.Create(iconcreator.Config{InputPath: sourcePath, OutputPath: filepath.Join(dir, "made.icns")})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct{ path, format string }{{out.ICOPath, "ico"}, {out.ICNSPath, "icns"}} {
		t.Run(test.format, func(t *testing.T) {
			got, err := previewDataURL(test.path, test.format)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(got, "data:image/png;base64,") {
				t.Fatalf("preview data URL uses wrong MIME type: %q", got[:min(len(got), 40)])
			}
		})
	}
}

func TestDefaultOutputPathProtectsImportedContainers(t *testing.T) {
	tests := []struct{ input, want string }{
		{input: filepath.Join("images", "logo.ico"), want: filepath.Join("images", "logo-edited.icns")},
		{input: filepath.Join("images", "Logo.ICNS"), want: filepath.Join("images", "Logo-edited.icns")},
		{input: filepath.Join("images", "logo.webp"), want: filepath.Join("images", "app.icns")},
	}
	for _, test := range tests {
		if got := defaultOutputPath(test.input); got != test.want {
			t.Errorf("defaultOutputPath(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestPasteImageImportsClipboardPNG(t *testing.T) {
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image.NewNRGBA(image.Rect(0, 0, 32, 24))); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	t.Cleanup(func() { app.shutdown(nil) })
	info, err := app.PasteImage("data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "Pasted image" || info.Width != 32 || info.Height != 24 {
		t.Fatalf("unexpected pasted image info: %+v", info)
	}
	if info.DefaultOutputPath == filepath.Join(filepath.Dir(info.Path), "app.icns") {
		t.Fatal("export defaults to temporary clipboard directory")
	}
	if _, err := os.Stat(info.Path); err != nil {
		t.Fatalf("pasted source unavailable for export: %v", err)
	}
	path := info.Path
	app.shutdown(nil)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("pasted source was not cleaned up: %v", err)
	}
}

func TestPasteImageRejectsInvalidData(t *testing.T) {
	app := NewApp()
	t.Cleanup(func() { app.shutdown(nil) })
	for _, data := range []string{
		"data:text/plain;base64,dGV4dA==",
		"data:image/png;base64,not base64",
		"data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not an image")),
	} {
		if _, err := app.PasteImage(data); err == nil {
			t.Fatalf("accepted invalid image: %q", data)
		}
	}
}
