package iconcreator

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
)

const (
	CanvasSize    = 1024
	DefaultRadius = 220
	DefaultZoom   = 1.0
	MaxZoom       = 3.0
)

type Config struct {
	InputPath         string
	OutputPath        string
	OutputDir         string
	Name              string
	Radius            int
	Zoom              float64
	PanX              float64
	PanY              float64
	KeepIntermediates bool
}

type Output struct {
	NormalizedPNG string
	IconsetDir    string
	ICNSPath      string
	ICOPath       string
	WorkingDir    string
}

type IconSpec struct {
	FileName string
	Size     int
}

func Create(cfg Config) (Output, error) {
	if cfg.InputPath == "" {
		return Output{}, errors.New("missing input image")
	}

	outputPaths, err := finalOutputPaths(cfg)
	if err != nil {
		return Output{}, err
	}

	radius := cfg.Radius
	if radius < 0 {
		radius = 0
	}
	if radius > CanvasSize/2 {
		radius = CanvasSize / 2
	}
	zoom := normalizeZoom(cfg.Zoom)
	panX := normalizePan(cfg.PanX)
	panY := normalizePan(cfg.PanY)

	source, err := loadImage(cfg.InputPath)
	if err != nil {
		return Output{}, err
	}

	parentDir := filepath.Dir(outputPaths.ICNSPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return Output{}, fmt.Errorf("create output directory: %w", err)
	}

	workDir, cleanup, err := workingDirectory(outputPaths.ICNSPath, cfg.KeepIntermediates)
	if err != nil {
		return Output{}, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	icon1024 := RoundedIcon(source, CanvasSize, radius, zoom, panX, panY)
	normalizedPath := filepath.Join(workDir, "icon.png")
	if err := writePNG(normalizedPath, icon1024); err != nil {
		return Output{}, err
	}

	name := SanitizeName(strings.TrimSuffix(filepath.Base(outputPaths.ICNSPath), filepath.Ext(outputPaths.ICNSPath)))
	if name == "" {
		name = "app"
	}
	iconsetDir := filepath.Join(workDir, name+".iconset")
	if err := os.RemoveAll(iconsetDir); err != nil {
		return Output{}, fmt.Errorf("remove old iconset: %w", err)
	}
	if err := os.MkdirAll(iconsetDir, 0755); err != nil {
		return Output{}, fmt.Errorf("create iconset directory: %w", err)
	}

	for _, spec := range IconSpecs() {
		resized := resizeNRGBA(icon1024, spec.Size, spec.Size)
		if err := writePNG(filepath.Join(iconsetDir, spec.FileName), resized); err != nil {
			return Output{}, err
		}
	}

	if err := os.Remove(outputPaths.ICNSPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Output{}, fmt.Errorf("remove old icns: %w", err)
	}
	if err := os.Remove(outputPaths.ICOPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Output{}, fmt.Errorf("remove old ico: %w", err)
	}

	cmd := exec.Command("/usr/bin/iconutil", "-c", "icns", "-o", outputPaths.ICNSPath, iconsetDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Output{}, fmt.Errorf("run iconutil: %w%s", err, formatToolError(stderr.String()))
	}
	if err := writeICO(outputPaths.ICOPath, icon1024); err != nil {
		return Output{}, err
	}

	out := Output{
		ICNSPath:   outputPaths.ICNSPath,
		ICOPath:    outputPaths.ICOPath,
		WorkingDir: workDir,
	}
	if cfg.KeepIntermediates {
		out.NormalizedPNG = normalizedPath
		out.IconsetDir = iconsetDir
	}
	return out, nil
}

func RoundedIcon(src image.Image, size int, radius int, zoom float64, panX float64, panY float64) *image.NRGBA {
	cropped := centerSquare(src, normalizeZoom(zoom), normalizePan(panX), normalizePan(panY))
	resized := resizeImage(cropped, size, size)
	applyRoundedMask(resized, radius)
	return resized
}

func IconSpecs() []IconSpec {
	return []IconSpec{
		{"icon_16x16.png", 16},
		{"icon_16x16@2x.png", 32},
		{"icon_32x32.png", 32},
		{"icon_32x32@2x.png", 64},
		{"icon_128x128.png", 128},
		{"icon_128x128@2x.png", 256},
		{"icon_256x256.png", 256},
		{"icon_256x256@2x.png", 512},
		{"icon_512x512.png", 512},
		{"icon_512x512@2x.png", 1024},
	}
}

func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".icns")
	name = strings.TrimSuffix(name, ".ico")
	name = strings.TrimSuffix(name, ".iconset")
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	name = strings.Trim(name, "-_")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return name
}

type outputPaths struct {
	ICNSPath string
	ICOPath  string
}

func finalOutputPaths(cfg Config) (outputPaths, error) {
	if cfg.OutputPath != "" {
		outputPath := filepath.Clean(cfg.OutputPath)
		if filepath.Ext(outputPath) == "" {
			outputPath += ".icns"
		}
		ext := strings.ToLower(filepath.Ext(outputPath))
		if ext != ".icns" && ext != ".ico" {
			return outputPaths{}, errors.New("output file must use the .icns or .ico extension")
		}
		base := strings.TrimSuffix(outputPath, filepath.Ext(outputPath))
		return outputPaths{
			ICNSPath: base + ".icns",
			ICOPath:  base + ".ico",
		}, nil
	}

	if cfg.OutputDir == "" {
		return outputPaths{}, errors.New("missing output path")
	}

	name := SanitizeName(cfg.Name)
	if name == "" {
		name = "app"
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return outputPaths{
		ICNSPath: filepath.Join(cfg.OutputDir, base+".icns"),
		ICOPath:  filepath.Join(cfg.OutputDir, base+".ico"),
	}, nil
}

func workingDirectory(outputPath string, keep bool) (string, func(), error) {
	if keep {
		base := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))
		workDir := filepath.Join(filepath.Dir(outputPath), base+"-icon-work")
		if err := os.RemoveAll(workDir); err != nil {
			return "", nil, fmt.Errorf("remove old working directory: %w", err)
		}
		if err := os.MkdirAll(workDir, 0755); err != nil {
			return "", nil, fmt.Errorf("create working directory: %w", err)
		}
		return workDir, nil, nil
	}

	workDir, err := os.MkdirTemp("", "icon-creator-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary working directory: %w", err)
	}
	return workDir, func() {
		_ = os.RemoveAll(workDir)
	}, nil
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode source image: %w", err)
	}
	return img, nil
}

func centerSquare(src image.Image, zoom float64, panX float64, panY float64) image.Image {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	side := w
	if h < side {
		side = h
	}
	cropSide := int(math.Round(float64(side) / zoom))
	cropSide = clampInt(cropSide, 1, side)

	maxShift := float64(side-cropSide) / 2
	xShift := int(math.Round(-(panX / 100) * maxShift))
	yShift := int(math.Round(-(panY / 100) * maxShift))
	x0 := b.Min.X + (w-cropSide)/2 + xShift
	y0 := b.Min.Y + (h-cropSide)/2 + yShift
	x0 = clampInt(x0, b.Min.X, b.Max.X-cropSide)
	y0 = clampInt(y0, b.Min.Y, b.Max.Y-cropSide)
	return cropImage(src, image.Rect(x0, y0, x0+cropSide, y0+cropSide))
}

func cropImage(src image.Image, rect image.Rectangle) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			dst.Set(x, y, src.At(rect.Min.X+x, rect.Min.Y+y))
		}
	}
	return dst
}

func resizeImage(src image.Image, width int, height int) *image.NRGBA {
	return resizeNRGBA(toNRGBA(src), width, height)
}

func toNRGBA(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func resizeNRGBA(src *image.NRGBA, width int, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()
	if srcW == 0 || srcH == 0 || width == 0 || height == 0 {
		return dst
	}

	scaleX := float64(srcW) / float64(width)
	scaleY := float64(srcH) / float64(height)

	for y := 0; y < height; y++ {
		sy := (float64(y)+0.5)*scaleY - 0.5
		y0 := clampInt(int(math.Floor(sy)), 0, srcH-1)
		y1 := clampInt(y0+1, 0, srcH-1)
		fy := sy - math.Floor(sy)
		if sy < 0 {
			fy = 0
		}

		for x := 0; x < width; x++ {
			sx := (float64(x)+0.5)*scaleX - 0.5
			x0 := clampInt(int(math.Floor(sx)), 0, srcW-1)
			x1 := clampInt(x0+1, 0, srcW-1)
			fx := sx - math.Floor(sx)
			if sx < 0 {
				fx = 0
			}

			c00 := src.NRGBAAt(x0, y0)
			c10 := src.NRGBAAt(x1, y0)
			c01 := src.NRGBAAt(x0, y1)
			c11 := src.NRGBAAt(x1, y1)

			dst.SetNRGBA(x, y, color.NRGBA{
				R: bilinear(c00.R, c10.R, c01.R, c11.R, fx, fy),
				G: bilinear(c00.G, c10.G, c01.G, c11.G, fx, fy),
				B: bilinear(c00.B, c10.B, c01.B, c11.B, fx, fy),
				A: bilinear(c00.A, c10.A, c01.A, c11.A, fx, fy),
			})
		}
	}

	return dst
}

func bilinear(c00, c10, c01, c11 uint8, fx, fy float64) uint8 {
	top := float64(c00)*(1-fx) + float64(c10)*fx
	bottom := float64(c01)*(1-fx) + float64(c11)*fx
	return uint8(math.Round(top*(1-fy) + bottom*fy))
}

func applyRoundedMask(img *image.NRGBA, radius int) {
	if radius <= 0 {
		return
	}

	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	r := float64(radius)
	samples := []float64{0.125, 0.375, 0.625, 0.875}
	sampleCount := float64(len(samples) * len(samples))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			coverage := 0.0
			for _, oy := range samples {
				for _, ox := range samples {
					if pointInRoundedRect(float64(x)+ox, float64(y)+oy, float64(w), float64(h), r) {
						coverage++
					}
				}
			}
			coverage /= sampleCount
			if coverage >= 1 {
				continue
			}

			i := img.PixOffset(x, y)
			img.Pix[i+3] = uint8(math.Round(float64(img.Pix[i+3]) * coverage))
			if img.Pix[i+3] == 0 {
				img.Pix[i+0] = 0
				img.Pix[i+1] = 0
				img.Pix[i+2] = 0
			}
		}
	}
}

func pointInRoundedRect(x, y, w, h, r float64) bool {
	if x >= r && x <= w-r {
		return true
	}
	if y >= r && y <= h-r {
		return true
	}

	cx := r
	if x > w-r {
		cx = w - r
	}
	cy := r
	if y > h-r {
		cy = h - r
	}

	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= r*r
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create png %s: %w", path, err)
	}
	defer f.Close()

	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(f, img); err != nil {
		return fmt.Errorf("write png %s: %w", path, err)
	}
	return nil
}

func writeICO(path string, src *image.NRGBA) error {
	type icoImage struct {
		size int
		data []byte
	}

	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	images := make([]icoImage, 0, len(sizes))
	for _, size := range sizes {
		var buf bytes.Buffer
		resized := resizeNRGBA(src, size, size)
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&buf, resized); err != nil {
			return fmt.Errorf("encode ico png %dx%d: %w", size, size, err)
		}
		images = append(images, icoImage{size: size, data: buf.Bytes()})
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create ico %s: %w", path, err)
	}
	defer f.Close()

	if err := binary.Write(f, binary.LittleEndian, uint16(0)); err != nil {
		return fmt.Errorf("write ico header: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil {
		return fmt.Errorf("write ico type: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(len(images))); err != nil {
		return fmt.Errorf("write ico count: %w", err)
	}

	offset := uint32(6 + 16*len(images))
	for _, img := range images {
		width := byte(img.size)
		height := byte(img.size)
		if img.size == 256 {
			width = 0
			height = 0
		}

		entry := []byte{width, height, 0, 0}
		if _, err := f.Write(entry); err != nil {
			return fmt.Errorf("write ico directory entry: %w", err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil {
			return fmt.Errorf("write ico planes: %w", err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint16(32)); err != nil {
			return fmt.Errorf("write ico bit depth: %w", err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint32(len(img.data))); err != nil {
			return fmt.Errorf("write ico image size: %w", err)
		}
		if err := binary.Write(f, binary.LittleEndian, offset); err != nil {
			return fmt.Errorf("write ico image offset: %w", err)
		}
		offset += uint32(len(img.data))
	}

	for _, img := range images {
		if _, err := f.Write(img.data); err != nil {
			return fmt.Errorf("write ico image data: %w", err)
		}
	}

	return nil
}

func normalizeZoom(zoom float64) float64 {
	if zoom <= 0 {
		return DefaultZoom
	}
	if zoom < DefaultZoom {
		return DefaultZoom
	}
	if zoom > MaxZoom {
		return MaxZoom
	}
	return zoom
}

func normalizePan(pan float64) float64 {
	if pan < -100 {
		return -100
	}
	if pan > 100 {
		return 100
	}
	return pan
}

func clampInt(v, minValue, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func formatToolError(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}
	return ": " + stderr
}
