package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"iconcreator/internal/iconcreator"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
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
	KeepIntermediates bool    `json:"keepIntermediates"`
}

type CreateIconResponse struct {
	ICNSPath      string `json:"icnsPath"`
	ICOPath       string `json:"icoPath"`
	Directory     string `json:"directory"`
	FileName      string `json:"fileName"`
	ICNSFileName  string `json:"icnsFileName"`
	ICOFileName   string `json:"icoFileName"`
	WorkingDir    string `json:"workingDir,omitempty"`
	CleanedUp     bool   `json:"cleanedUp"`
	ReplacedFile  bool   `json:"replacedFile"`
	OutputSize    int64  `json:"outputSize"`
	ICNSSize      int64  `json:"icnsSize"`
	ICOSize       int64  `json:"icoSize"`
	StatusMessage string `json:"statusMessage"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) SelectImage() (ImageInfo, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select source image",
		Filters: []runtime.FileFilter{
			{DisplayName: "Image Files (*.png, *.jpg, *.jpeg, *.gif)", Pattern: "*.png;*.jpg;*.jpeg;*.gif"},
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
			{DisplayName: "Icon Export (*.icns, *.ico)", Pattern: "*.icns;*.ico"},
		},
	})
}

func (a *App) InspectImage(path string) (ImageInfo, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return ImageInfo{}, fmt.Errorf("missing image path")
	}

	file, err := os.Open(path)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("open image: %w", err)
	}
	config, format, err := image.DecodeConfig(file)
	_ = file.Close()
	if err != nil {
		return ImageInfo{}, fmt.Errorf("read image dimensions: %w", err)
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
		DefaultOutputPath: filepath.Join(filepath.Dir(path), "app.icns"),
		Width:             config.Width,
		Height:            config.Height,
		SizeBytes:         info.Size(),
		PreviewDataURL:    preview,
	}, nil
}

func (a *App) CreateIcon(req CreateIconRequest) (CreateIconResponse, error) {
	if strings.TrimSpace(req.OutputPath) == "" && strings.TrimSpace(req.InputPath) != "" {
		req.OutputPath = filepath.Join(filepath.Dir(req.InputPath), "app.icns")
	}

	replaced := false
	if _, err := os.Stat(req.OutputPath); err == nil {
		replaced = true
	}
	if _, err := os.Stat(siblingIconPath(req.OutputPath)); err == nil {
		replaced = true
	}

	out, err := iconcreator.Create(iconcreator.Config{
		InputPath:         req.InputPath,
		OutputPath:        req.OutputPath,
		Radius:            req.Radius,
		Zoom:              req.Zoom,
		PanX:              req.PanX,
		PanY:              req.PanY,
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

	response := CreateIconResponse{
		ICNSPath:      out.ICNSPath,
		ICOPath:       out.ICOPath,
		Directory:     filepath.Dir(out.ICNSPath),
		FileName:      filepath.Base(out.ICNSPath) + " + " + filepath.Base(out.ICOPath),
		ICNSFileName:  filepath.Base(out.ICNSPath),
		ICOFileName:   filepath.Base(out.ICOPath),
		WorkingDir:    out.WorkingDir,
		CleanedUp:     !req.KeepIntermediates,
		ReplacedFile:  replaced,
		OutputSize:    icnsInfo.Size() + icoInfo.Size(),
		ICNSSize:      icnsInfo.Size(),
		ICOSize:       icoInfo.Size(),
		StatusMessage: "Created " + filepath.Base(out.ICNSPath) + " and " + filepath.Base(out.ICOPath),
	}
	return response, nil
}

func (a *App) Reveal(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("missing path")
	}
	return exec.Command("/usr/bin/open", "-R", path).Run()
}

func previewDataURL(path string, format string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read preview image: %w", err)
	}

	mime := "image/" + format
	if format == "jpg" {
		mime = "image/jpeg"
	}
	if format == "" {
		mime = "application/octet-stream"
	}

	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func siblingIconPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	clean := filepath.Clean(path)
	ext := strings.ToLower(filepath.Ext(clean))
	if ext == "" {
		return strings.TrimSuffix(clean, filepath.Ext(clean)) + ".ico"
	}
	base := strings.TrimSuffix(clean, filepath.Ext(clean))
	if ext == ".ico" {
		return base + ".icns"
	}
	return base + ".ico"
}
