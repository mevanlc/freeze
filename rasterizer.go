package main

import (
	"fmt"
	"os/exec"

	"github.com/beevik/etree"
)

// Rasterizer identifies the implementation used to convert SVG to PNG.
type Rasterizer string

const (
	rasterizerAuto     Rasterizer = "auto"
	rasterizerRSVG     Rasterizer = "rsvg"
	rasterizerRSVGPDF  Rasterizer = "rsvg-pdf"
	rasterizerResvg    Rasterizer = "resvg"
	rasterizerSips     Rasterizer = "sips"
	rasterizerChromium Rasterizer = "chromium"
)

type pngRasterizer interface {
	Rasterize(doc *etree.Document, width, height float64, output string) error
}

type pngRasterizerFunc func(doc *etree.Document, width, height float64, output string) error

func (f pngRasterizerFunc) Rasterize(doc *etree.Document, width, height float64, output string) error {
	return f(doc, width, height, output)
}

type fallbackRasterizer struct {
	primary  pngRasterizer
	fallback pngRasterizer
}

func (r fallbackRasterizer) Rasterize(doc *etree.Document, width, height float64, output string) error {
	if err := r.primary.Rasterize(doc, width, height, output); err == nil {
		return nil
	}
	return r.fallback.Rasterize(doc, width, height, output)
}

func newPNGRasterizer(name Rasterizer) (pngRasterizer, error) {
	return newPNGRasterizerWithLookPath(name, exec.LookPath)
}

func newPNGRasterizerWithLookPath(name Rasterizer, lookPath commandLookPath) (pngRasterizer, error) {
	rsvg := pngRasterizerFunc(rsvgConvert)
	resvg := pngRasterizerFunc(resvgConvert)
	rsvgFallback := fallbackRasterizer{primary: rsvg, fallback: resvg}

	switch name {
	case rasterizerAuto:
		rsvgPDF, err := newRSVGPDFRasterizer(lookPath)
		if err != nil {
			// An unavailable optional backend is not an error for automatic selection.
			return rsvgFallback, nil //nolint:nilerr
		}
		return fallbackRasterizer{primary: rsvgPDF, fallback: rsvgFallback}, nil
	case rasterizerRSVG:
		return rsvg, nil
	case rasterizerRSVGPDF:
		return newRSVGPDFRasterizer(lookPath)
	case rasterizerResvg:
		return resvg, nil
	case rasterizerSips:
		return pngRasterizerFunc(func(doc *etree.Document, _, _ float64, output string) error {
			return sipsConvert(doc, output)
		}), nil
	case rasterizerChromium:
		return pngRasterizerFunc(chromiumConvert), nil
	default:
		return nil, fmt.Errorf("unknown rasterizer %q", name)
	}
}

func rasterizePNG(name Rasterizer, doc *etree.Document, width, height float64, output string) error {
	rasterizer, err := newPNGRasterizer(name)
	if err != nil {
		return err
	}
	return rasterizer.Rasterize(doc, width, height, output)
}
