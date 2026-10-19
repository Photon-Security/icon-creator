package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"

	"iconcreator/internal/iconcreator"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx       context.Context
	pastedDir string
	pastedMu  sync.Mutex
}

type ImageInfo struct {
	Path              string `json:"path"`
	Name              string `json:"name"`
	Directory         string `json:"directory"`
	DefaultOutputPath string `json:"defaultOutputPath"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	SizeBytes         int64  `json:"sizeBytes"`
	PreviewDataURL    string `json:"previewDataURL"`
}

type CreateIconRequest struct {
	InputPath         string  `json:"inputPath"`
	OutputPath        string  `json:"outputPath"`
	Radius            int     `json:"radius"`
	Zoom              float64 `json:"zoom"`
	PanX              float64 `json:"panX"`
	PanY              float64 `json:"panY"`
	TransparentBg     bool    `json:"transparentBg"`
	KeepIntermediates bool    `json:"keepIntermediates"`
}

type CreateIconResponse struct {
	ICNSPath      string `json:"icnsPath"`
	ICOPath       string `json:"icoPath"`
	PNGPath       string `json:"pngPath"`
	Directory     string `json:"directory"`
	FileName      string `json:"fileName"`
	ICNSFileName  string `json:"icnsFileName"`
	ICOFileName   string `json:"icoFileName"`
	PNGFileName   string `json:"pngFileName"`
	WorkingDir    string `json:"workingDir,omitempty"`
	CleanedUp     bool   `json:"cleanedUp"`
	ReplacedFile  bool   `json:"replacedFile"`
	OutputSize    int64  `json:"outputSize"`
	ICNSSize      int64  `json:"icnsSize"`
	ICOSize       int64  `json:"icoSize"`
	PNGSize       int64  `json:"pngSize"`
	StatusMessage string `json:"statusMessage"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(context.Context) {
	a.pastedMu.Lock()
	defer a.pastedMu.Unlock()
	if a.pastedDir != "" {
		_ = os.RemoveAll(a.pastedDir)
		a.pastedDir = ""
	}
}

// PasteImage imports image bytes supplied by a clipboard paste event. The
// temporary source stays available until exit so exports can use it later.
func (a *App) PasteImage(dataURL string) (ImageInfo, error) {
	const maxImageBytes = 50 << 20
	header, encoded, found := strings.Cut(dataURL, ",")
	extensions := map[string]string{
		"data:image/png;base64":                ".png",
		"data:image/jpeg;base64":               ".jpg",
		"data:image/gif;base64":                ".gif",
		"data:image/webp;base64":               ".webp",
		"data:image/svg+xml;base64":            ".svg",
		"data:image/x-icon;base64":             ".ico",
		"data:image/vnd.microsoft.icon;base64": ".ico",
		"data:image/icns;base64":               ".icns",
	}
	extension, supported := extensions[header]
	if !found || !supported || len(encoded) > base64.StdEncoding.EncodedLen(maxImageBytes) {
		return ImageInfo{}, fmt.Errorf("invalid or oversized clipboard image (50 MB maximum)")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) == 0 || len(data) > maxImageBytes {
		return ImageInfo{}, fmt.Errorf("invalid or oversized clipboard image (50 MB maximum)")
	}

	a.pastedMu.Lock()
	defer a.pastedMu.Unlock()
	if a.pastedDir == "" {
		a.pastedDir, err = os.MkdirTemp("", "icon-creator-paste-*")
		if err != nil {
			return ImageInfo{}, fmt.Errorf("prepare clipboard image: %w", err)
		}
	}
	file, err := os.CreateTemp(a.pastedDir, "pasted-*"+extension)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("save clipboard image: %w", err)
	}
	path := file.Name()
	if _, err = file.Write(data); err != nil {
		file.Close()
		os.Remove(path)
		return ImageInfo{}, fmt.Errorf("save clipboard image: %w", err)
	}
	if err = file.Close(); err != nil {
		os.Remove(path)
		return ImageInfo{}, fmt.Errorf("save clipboard image: %w", err)
	}
	info, err := a.InspectImage(path)
	if err != nil {
		os.Remove(path)
		return ImageInfo{}, err
	}
	// A pasted source has no meaningful source directory for export.
	home, err := os.UserHomeDir()
	if err != nil {
		os.Remove(path)
		return ImageInfo{}, fmt.Errorf("find default export directory: %w", err)
	}
	info.Name = "Pasted image"
	info.DefaultOutputPath = filepath.Join(home, "app.icns")
	return info, nil
}

func (a *App) SelectImage() (ImageInfo, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select source image",
		Filters: []runtime.FileFilter{
			{DisplayName: "Image Files (*.png, *.jpg, *.jpeg, *.gif, *.webp, *.svg, *.ico, *.icns)", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.svg;*.ico;*.icns"},
			{DisplayName: "PNG Files (*.png)", Pattern: "*.png"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || path == "" {
		return ImageInfo{}, err
	}
	return a.InspectImage(path)
}

func (a *App) SelectOutput(defaultPath string) (string, error) {
	defaultDir := ""
	defaultName := "app.icns"
	if defaultPath != "" {
		defaultDir = filepath.Dir(defaultPath)
		defaultName = filepath.Base(defaultPath)
	}

	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Save icon export",
		DefaultDirectory:     defaultDir,
		DefaultFilename:      defaultName,
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "Icon Export (*.icns, *.ico, *.png)", Pattern: "*.icns;*.ico;*.png"},
		},
	})
}

func (a *App) InspectImage(path string) (ImageInfo, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return ImageInfo{}, fmt.Errorf("missing image path")
	}

	config, format, err := iconcreator.DecodeSourceConfig(path)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("read image: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("read image metadata: %w", err)
	}

	preview, err := previewDataURL(path, format)
	if err != nil {
		return ImageInfo{}, err
	}

	return ImageInfo{
		Path:              path,
		Name:              filepath.Base(path),
		Directory:         filepath.Dir(path),
		DefaultOutputPath: defaultOutputPath(path),
		Width:             config.Width,
		Height:            config.Height,
		SizeBytes:         info.Size(),
		PreviewDataURL:    preview,
	}, nil
}

func (a *App) CreateIcon(req CreateIconRequest) (CreateIconResponse, error) {
	if strings.TrimSpace(req.OutputPath) == "" && strings.TrimSpace(req.InputPath) != "" {
		req.OutputPath = defaultOutputPath(req.InputPath)
	}

	replaced := false
	for _, path := range siblingOutputPaths(req.OutputPath) {
		if _, err := os.Stat(path); err == nil {
			replaced = true
			break
		}
	}

	out, err := iconcreator.Create(iconcreator.Config{
		InputPath:         req.InputPath,
		OutputPath:        req.OutputPath,
		Radius:            req.Radius,
		Zoom:              req.Zoom,
		PanX:              req.PanX,
		PanY:              req.PanY,
		TransparentBg:     req.TransparentBg,
		KeepIntermediates: req.KeepIntermediates,
	})
	if err != nil {
		return CreateIconResponse{}, err
	}

	icnsInfo, err := os.Stat(out.ICNSPath)
	if err != nil {
		return CreateIconResponse{}, fmt.Errorf("read generated icon: %w", err)
	}
	icoInfo, err := os.Stat(out.ICOPath)
	if err != nil {
		return CreateIconResponse{}, fmt.Errorf("read generated ico: %w", err)
	}
	pngInfo, err := os.Stat(out.PNGPath)
	if err != nil {
		return CreateIconResponse{}, fmt.Errorf("read generated png: %w", err)
	}

	response := CreateIconResponse{
		ICNSPath:      out.ICNSPath,
		ICOPath:       out.ICOPath,
		PNGPath:       out.PNGPath,
		Directory:     filepath.Dir(out.ICNSPath),
		FileName:      filepath.Base(out.ICNSPath) + " + " + filepath.Base(out.ICOPath) + " + " + filepath.Base(out.PNGPath),
		ICNSFileName:  filepath.Base(out.ICNSPath),
		ICOFileName:   filepath.Base(out.ICOPath),
		PNGFileName:   filepath.Base(out.PNGPath),
		WorkingDir:    out.WorkingDir,
		CleanedUp:     !req.KeepIntermediates,
		ReplacedFile:  replaced,
		OutputSize:    icnsInfo.Size() + icoInfo.Size() + pngInfo.Size(),
		ICNSSize:      icnsInfo.Size(),
		ICOSize:       icoInfo.Size(),
		PNGSize:       pngInfo.Size(),
		StatusMessage: "Created " + filepath.Base(out.ICNSPath) + ", " + filepath.Base(out.ICOPath) + ", and " + filepath.Base(out.PNGPath),
	}
	return response, nil
}

func (a *App) Reveal(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("missing path")
	}
	switch goruntime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,"+filepath.FromSlash(path)).Run()
	default:
		return exec.Command("/usr/bin/open", "-R", path).Run()
	}
}

func previewDataURL(path string, format string) (string, error) {
	if format == "ico" || format == "icns" {
		img, _, err := iconcreator.DecodeSource(path)
		if err != nil {
			return "", fmt.Errorf("decode preview image: %w", err)
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, img); err != nil {
			return "", fmt.Errorf("encode preview image: %w", err)
		}
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes()), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read preview image: %w", err)
	}

	mime := "image/" + format
	if format == "jpg" || format == "jpeg" {
		mime = "image/jpeg"
	}
	if format == "svg" {
		mime = "image/svg+xml"
	}
	if format == "" {
		mime = "application/octet-stream"
	}

	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func defaultOutputPath(inputPath string) string {
	directory := filepath.Dir(inputPath)
	ext := strings.ToLower(filepath.Ext(inputPath))
	if ext == ".ico" || ext == ".icns" {
		name := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		return filepath.Join(directory, name+"-edited.icns")
	}
	return filepath.Join(directory, "app.icns")
}

func siblingOutputPaths(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	clean := filepath.Clean(path)
	if filepath.Ext(clean) == "" {
		clean += ".icns"
	}
	ext := strings.ToLower(filepath.Ext(clean))
	base := strings.TrimSuffix(clean, filepath.Ext(clean))
	if ext != ".icns" && ext != ".ico" && ext != ".png" {
		return []string{clean}
	}
	return []string{base + ".icns", base + ".ico", base + ".png"}
}
