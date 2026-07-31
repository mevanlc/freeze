package main

import (
	imagepng "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

func TestValidateResvgFont(t *testing.T) {
	tests := []struct {
		name    string
		family  string
		wantErr string
	}{
		{name: "bundled font", family: "JetBrains Mono"},
		{name: "bundled fallback", family: "Meslo, 'JetBrains Mono'"},
		{
			name:    "system fonts only",
			family:  "MesloLGSDZ Nerd Font, Menlo, Apple Color Emoji",
			wantErr: "resvg supports only the bundled JetBrains Mono font",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := etree.NewDocument()
			root := doc.CreateElement("svg")
			group := root.CreateElement("g")
			group.CreateAttr("font-family", tc.family)

			err := validateResvgFont(doc)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got error %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateResvgComplexity(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("svg")
	for range resvgMaxTextElements + 1 {
		root.CreateElement("tspan")
	}

	err := validateResvgComplexity(doc)
	if err == nil || !strings.Contains(err.Error(), "use rsvg-pdf, rsvg, sips, or chromium") {
		t.Fatalf("got error %v, want a rasterizer recommendation", err)
	}
}

func TestRSVGPDFRasterizerDimensionsAndTransparency(t *testing.T) {
	rsvgConvertPath, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("rsvg-convert is not installed")
	}
	pdftocairoPath, err := exec.LookPath("pdftocairo")
	if err != nil {
		t.Skip("pdftocairo is not installed")
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><rect x="5" y="2" width="10" height="6" fill="#ff0000"/></svg>`); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(t.TempDir(), "rsvg-pdf.png")
	rasterizer := rsvgPDFRasterizer{
		rsvgConvertPath: rsvgConvertPath,
		pdftocairoPath:  pdftocairoPath,
	}
	if err := rasterizer.Rasterize(doc, 20, 10, output); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint: errcheck
	image, err := imagepng.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if got := image.Bounds().Size(); got.X != 20 || got.Y != 10 {
		t.Fatalf("image size is %dx%d, want 20x10", got.X, got.Y)
	}
	if _, _, _, alpha := image.At(0, 0).RGBA(); alpha != 0 {
		t.Fatalf("background alpha is %#x, want transparent", alpha)
	}
	red, green, blue, alpha := image.At(10, 5).RGBA()
	if red != 0xffff || green != 0 || blue != 0 || alpha != 0xffff {
		t.Fatalf("center pixel is rgba(%#x, %#x, %#x, %#x), want opaque red", red, green, blue, alpha)
	}
}
