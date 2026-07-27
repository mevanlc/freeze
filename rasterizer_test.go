package main

import (
	"errors"
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
