package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

func TestFallbackRasterizer(t *testing.T) {
	primaryErr := errors.New("primary failed")
	var calls []string
	primary := pngRasterizerFunc(func(*etree.Document, float64, float64, string) error {
		calls = append(calls, "primary")
		return primaryErr
	})
	fallback := pngRasterizerFunc(func(*etree.Document, float64, float64, string) error {
		calls = append(calls, "fallback")
		return nil
	})

	rasterizer := fallbackRasterizer{primary: primary, fallback: fallback}
	if err := rasterizer.Rasterize(etree.NewDocument(), 1, 1, "out.png"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "primary" || calls[1] != "fallback" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestUnknownRasterizer(t *testing.T) {
	if _, err := newPNGRasterizer("unknown"); err == nil {
		t.Fatal("expected an unknown rasterizer error")
	}
}

func TestAutoRasterizerPrefersRSVGPDFWhenAvailable(t *testing.T) {
	lookPath := func(name string) (string, error) {
		return "/commands/" + name, nil
	}

	rasterizer, err := newPNGRasterizerWithLookPath(rasterizerAuto, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	fallback, ok := rasterizer.(fallbackRasterizer)
	if !ok {
		t.Fatalf("auto rasterizer is %T, want fallbackRasterizer", rasterizer)
	}
	if _, ok := fallback.primary.(rsvgPDFRasterizer); !ok {
		t.Fatalf("auto primary is %T, want rsvgPDFRasterizer", fallback.primary)
	}
}

func TestAutoRasterizerSkipsRSVGPDFWhenDependencyIsMissing(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "pdftocairo" {
			return "", errors.New("not found")
		}
		return "/commands/" + name, nil
	}

	rasterizer, err := newPNGRasterizerWithLookPath(rasterizerAuto, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	fallback, ok := rasterizer.(fallbackRasterizer)
	if !ok {
		t.Fatalf("auto rasterizer is %T, want fallbackRasterizer", rasterizer)
	}
	if _, ok := fallback.primary.(rsvgPDFRasterizer); ok {
		t.Fatal("auto selected rsvg-pdf without pdftocairo")
	}
}

func TestRSVGPDFRasterizerReportsMissingDependency(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "pdftocairo" {
			return "", errors.New("not found")
		}
		return "/commands/" + name, nil
	}

	_, err := newPNGRasterizerWithLookPath(rasterizerRSVGPDF, lookPath)
	if err == nil || !strings.Contains(err.Error(), "rsvg-pdf requires pdftocairo on PATH") {
		t.Fatalf("got error %v, want missing pdftocairo error", err)
	}
}
