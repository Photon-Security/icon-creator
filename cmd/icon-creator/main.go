package main

import (
	"flag"
	"fmt"
	"os"

	"iconcreator/internal/iconcreator"
)

func main() {
	cfg := iconcreator.Config{
		Radius: iconcreator.DefaultRadius,
		Zoom:   iconcreator.DefaultZoom,
		Name:   "app",
	}
	quiet := false

	flag.StringVar(&cfg.InputPath, "input", "", "source image path")
	flag.StringVar(&cfg.OutputPath, "output", "", "final .icns or .ico output path; the matching sibling file is also created")
	flag.StringVar(&cfg.OutputDir, "output-dir", "", "output directory, used with -name")
	flag.IntVar(&cfg.Radius, "radius", iconcreator.DefaultRadius, "corner radius in pixels, from 0 to 512")
	flag.Float64Var(&cfg.Zoom, "zoom", iconcreator.DefaultZoom, "center crop zoom, from 1.0 to 3.0")
	flag.Float64Var(&cfg.PanX, "pan-x", 0, "horizontal crop offset, from -100 to 100")
	flag.Float64Var(&cfg.PanY, "pan-y", 0, "vertical crop offset, from -100 to 100")
	flag.BoolVar(&cfg.TransparentBg, "transparent-background", false, "turn a solid connected outer color into transparency")
	flag.StringVar(&cfg.Name, "name", "app", "base output name when using -output-dir")
	flag.BoolVar(&cfg.KeepIntermediates, "keep-intermediates", false, "keep icon.png and .iconset beside the output icons")
	flag.BoolVar(&quiet, "quiet", false, "print only the generated icon paths")
	flag.Parse()

	out, err := iconcreator.Create(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if quiet {
		fmt.Println(out.ICNSPath)
		fmt.Println(out.ICOPath)
		fmt.Println(out.PNGPath)
		return
	}

	fmt.Printf("Created %s\n", out.ICNSPath)
	fmt.Printf("Created %s\n", out.ICOPath)
	fmt.Printf("Created %s\n", out.PNGPath)
	if cfg.KeepIntermediates {
		fmt.Printf("Working files: %s\n", out.WorkingDir)
	}
}
