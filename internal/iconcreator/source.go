package iconcreator

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/fyne-io/image/ico"
	"github.com/jackmordaunt/icns/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/riff"
	"golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

const maxSVGRasterDimension = 8192

// DecodeSourceConfig reads a source image's dimensions without fully decoding
// raster files. SVG dimensions reflect the high-resolution raster used during
// icon creation.
func DecodeSourceConfig(path string) (image.Config, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, "", fmt.Errorf("open source image: %w", err)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".svg" {
		icon, err := readSVG(f)
		if err != nil {
			return image.Config{}, "", fmt.Errorf("decode source SVG: %w", err)
		}
		width, height := svgRasterSize(icon.ViewBox.W, icon.ViewBox.H)
		return image.Config{Width: width, Height: height}, "svg", nil
	}
	if ext == ".ico" || ext == ".icns" {
		img, format, err := decodeContainer(f, ext)
		if err != nil {
			return image.Config{}, "", fmt.Errorf("decode source %s: %w", strings.ToUpper(strings.TrimPrefix(ext, ".")), err)
		}
		size := img.Bounds().Size()
		return image.Config{ColorModel: img.ColorModel(), Width: size.X, Height: size.Y}, format, nil
	}

	config, format, err := image.DecodeConfig(f)
	if err != nil {
		return image.Config{}, "", fmt.Errorf("decode source image: %w", err)
	}
	return config, format, nil
}

// DecodeSource loads a supported source image. SVG files are rasterized with
// enough detail for the 1024px icon output while preserving their aspect ratio.
func DecodeSource(path string) (image.Image, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("open source image: %w", err)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".svg" {
		img, err := decodeSVG(f)
		if err != nil {
			return nil, "", fmt.Errorf("decode source SVG: %w", err)
		}
		return img, "svg", nil
	}
	if ext == ".ico" || ext == ".icns" {
		img, format, err := decodeContainer(f, ext)
		if err != nil {
			return nil, "", fmt.Errorf("decode source %s: %w", strings.ToUpper(strings.TrimPrefix(ext, ".")), err)
		}
		return img, format, nil
	}

	img, format, err := image.Decode(f)
	if err != nil {
		// x/image/webp versions compatible with Go 1.22 reject valid extended
		// lossless WebP files whose VP8X header advertises alpha. Retry that
		// specific container layout by decoding its VP8L payload directly.
		if _, seekErr := f.Seek(0, io.SeekStart); seekErr == nil {
			if webpImage, fallbackErr := decodeExtendedLosslessWebP(f); fallbackErr == nil {
				return webpImage, "webp", nil
			}
		}
		return nil, "", fmt.Errorf("decode source image: %w", err)
	}
	return img, format, nil
}

var (
	webpFormType = riff.FourCC{'W', 'E', 'B', 'P'}
	webpVP8X     = riff.FourCC{'V', 'P', '8', 'X'}
	webpVP8L     = riff.FourCC{'V', 'P', '8', 'L'}
)

// decodeExtendedLosslessWebP handles VP8X containers with a top-level VP8L
// image. This is valid WebP, but x/image/webp rejected it before v0.38.0.
func decodeExtendedLosslessWebP(r io.Reader) (image.Image, error) {
	formType, chunks, err := riff.NewReader(r)
	if err != nil || formType != webpFormType {
		return nil, fmt.Errorf("invalid extended lossless WebP")
	}

	var canvas image.Point
	seenVP8X := false
	for {
		chunkID, chunkLen, chunkData, err := chunks.Next()
		if err != nil {
			return nil, err
		}

		switch chunkID {
		case webpVP8X:
			if seenVP8X || chunkLen != 10 {
				return nil, fmt.Errorf("invalid WebP VP8X header")
			}
			seenVP8X = true
			var header [10]byte
			if _, err := io.ReadFull(chunkData, header[:]); err != nil {
				return nil, err
			}
			const animationBit = 1 << 1
			if header[0]&animationBit != 0 {
				return nil, fmt.Errorf("animated WebP is not supported")
			}
			canvas = image.Pt(
				1+int(header[4])+(int(header[5])<<8)+(int(header[6])<<16),
				1+int(header[7])+(int(header[8])<<8)+(int(header[9])<<16),
			)

		case webpVP8L:
			if !seenVP8X {
				return nil, fmt.Errorf("WebP does not use an extended lossless container")
			}

			buffered := bufio.NewReader(chunkData)
			var header [5]byte
			if _, err := io.ReadFull(buffered, header[:]); err != nil {
				return nil, err
			}
			config, err := vp8l.DecodeConfig(bytes.NewReader(header[:]))
			if err != nil {
				return nil, err
			}
			if config.Width != canvas.X || config.Height != canvas.Y {
				return nil, fmt.Errorf("WebP canvas and image dimensions do not match")
			}

			return vp8l.Decode(io.MultiReader(bytes.NewReader(header[:]), buffered))
		}
	}
}

func decodeContainer(f *os.File, ext string) (img image.Image, format string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			img = nil
			format = ""
			err = fmt.Errorf("invalid or corrupt %s file", strings.ToUpper(strings.TrimPrefix(ext, ".")))
		}
	}()

	switch ext {
	case ".ico":
		img, err = ico.Decode(f)
		if err != nil {
			return nil, "", err
		}
		return img, "ico", nil
	case ".icns":
		img, err = icns.Decode(f)
		if err != nil {
			if strings.Contains(err.Error(), "no icons found") {
				return nil, "", fmt.Errorf("no supported PNG or JPEG icon frames found (JPEG2000-only ICNS files are not supported)")
			}
			return nil, "", err
		}
		return img, "icns", nil
	default:
		return nil, "", fmt.Errorf("unsupported icon container %q", ext)
	}
}

func decodeSVG(f *os.File) (image.Image, error) {
	icon, err := readSVG(f)
	if err != nil {
		return nil, err
	}

	width, height := svgRasterSize(icon.ViewBox.W, icon.ViewBox.H)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	icon.SetTarget(0, 0, float64(width), float64(height))
	scanner := rasterx.NewScannerGV(width, height, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(width, height, scanner), 1)
	return img, nil
}

func readSVG(f *os.File) (*oksvg.SvgIcon, error) {
	icon, err := oksvg.ReadIconStream(f)
	if err != nil {
		return nil, err
	}
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 ||
		math.IsNaN(icon.ViewBox.W) || math.IsNaN(icon.ViewBox.H) ||
		math.IsInf(icon.ViewBox.W, 0) || math.IsInf(icon.ViewBox.H, 0) {
		return nil, fmt.Errorf("SVG must define a positive width and height or viewBox")
	}
	return icon, nil
}

func svgRasterSize(width, height float64) (int, int) {
	if width >= height {
		rasterHeight := CanvasSize
		rasterWidth := int(math.Round(float64(rasterHeight) * width / height))
		if rasterWidth > maxSVGRasterDimension {
			rasterWidth = maxSVGRasterDimension
			rasterHeight = maxInt(1, int(math.Round(float64(rasterWidth)*height/width)))
		}
		return rasterWidth, rasterHeight
	}

	rasterWidth := CanvasSize
	rasterHeight := int(math.Round(float64(rasterWidth) * height / width))
	if rasterHeight > maxSVGRasterDimension {
		rasterHeight = maxSVGRasterDimension
		rasterWidth = maxInt(1, int(math.Round(float64(rasterHeight)*width/height)))
	}
	return rasterWidth, rasterHeight
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
