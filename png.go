package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/charmbracelet/freeze/font"
	"github.com/kanrichan/resvg-go"
	"github.com/tetratelabs/wazero"
)

const (
	resvgRenderTimeout    = 30 * time.Second
	resvgMemoryLimitPages = 8192 // 512 MiB of WebAssembly memory.
	resvgMaxTextElements  = 1000
)

func rsvgConvert(doc *etree.Document, _, _ float64, output string) error {
	_, err := exec.LookPath("rsvg-convert")
	if err != nil {
		return err //nolint: wrapcheck
	}

	svg, err := doc.WriteToBytes()
	if err != nil {
		return err //nolint: wrapcheck
	}

	// rsvg-convert is installed use that to convert the SVG to PNG,
	// since it is faster.
	cmd := exec.Command("rsvg-convert", "-o", output)
	cmd.Stdin = bytes.NewReader(svg)
	err = cmd.Run()
	return err //nolint: wrapcheck
}

func resvgConvert(doc *etree.Document, w, h float64, output string) error {
	if err := validateResvgFont(doc); err != nil {
		return err
	}
	if err := validateResvgComplexity(doc); err != nil {
		return err
	}

	svg, err := doc.WriteToBytes()
	if err != nil {
		return err //nolint: wrapcheck
	}

	ctx, cancel := context.WithTimeout(context.Background(), resvgRenderTimeout)
	defer cancel()
	worker, err := resvg.NewWorker(ctx, wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(resvgMemoryLimitPages))
	if err != nil {
		return fmt.Errorf("create resvg worker: %w", err)
	}
	defer worker.Close() //nolint: errcheck

	fontdb, err := worker.NewFontDBDefault()
	if err != nil {
		return fmt.Errorf("create resvg font database: %w", err)
	}
	defer fontdb.Close() //nolint: errcheck
	err = fontdb.LoadFontData(font.JetBrainsMonoTTF)
	if err != nil {
		return fmt.Errorf("load JetBrains Mono font: %w", err)
	}
	err = fontdb.LoadFontData(font.JetBrainsMonoNLTTF)
	if err != nil {
		return fmt.Errorf("load JetBrains Mono NL font: %w", err)
	}

	pixmap, err := worker.NewPixmap(uint32(w), uint32(h))
	if err != nil {
		return fmt.Errorf("create resvg pixmap: %w", err)
	}
	defer pixmap.Close() //nolint: errcheck

	tree, err := worker.NewTreeFromData(svg, &resvg.Options{
		Dpi:                192,
		ShapeRenderingMode: resvg.ShapeRenderingModeGeometricPrecision,
		TextRenderingMode:  resvg.TextRenderingModeOptimizeLegibility,
		ImageRenderingMode: resvg.ImageRenderingModeOptimizeQuality,
		DefaultSizeWidth:   float32(w),
		DefaultSizeHeight:  float32(h),
	})
	if err != nil {
		return fmt.Errorf("parse SVG with resvg: %w", err)
	}
	defer tree.Close() //nolint: errcheck

	err = tree.ConvertText(fontdb)
	if err != nil {
		return err //nolint: wrapcheck
	}
	err = tree.Render(resvg.TransformIdentity(), pixmap)
	if err != nil {
		return err //nolint: wrapcheck
	}
	png, err := pixmap.EncodePNG()
	if err != nil {
		return err //nolint: wrapcheck
	}

	err = os.WriteFile(output, png, 0o600)
	if err != nil {
		return err //nolint: wrapcheck
	}
	return err //nolint: wrapcheck
}

func validateResvgFont(doc *etree.Document) error {
	fontGroup := doc.FindElement("//g[@font-family]")
	if fontGroup == nil {
		return nil
	}

	family := fontGroup.SelectAttrValue("font-family", "")
	for candidate := range strings.SplitSeq(family, ",") {
		candidate = strings.Trim(strings.TrimSpace(candidate), `"'`)
		if candidate == "JetBrains Mono" {
			return nil
		}
	}

	return fmt.Errorf(
		"resvg supports only the bundled JetBrains Mono font, not %q; use rsvg, sips, or chromium for system fonts",
		family,
	)
}

func validateResvgComplexity(doc *etree.Document) error {
	textElements := len(doc.FindElements("//text")) + len(doc.FindElements("//tspan"))
	if textElements <= resvgMaxTextElements {
		return nil
	}

	return fmt.Errorf(
		"resvg input contains %d text elements (limit %d); use rsvg, sips, or chromium for large text-heavy captures",
		textElements,
		resvgMaxTextElements,
	)
}
