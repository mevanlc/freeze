package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/rivo/uniseg"
)

func sipsConvert(doc *etree.Document, output string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("sips rasterizer is only available on macOS")
	}

	sips, err := exec.LookPath("sips")
	if err != nil {
		return fmt.Errorf("find sips: %w", err)
	}

	svgFile, err := os.CreateTemp("", "freeze-*.svg")
	if err != nil {
		return fmt.Errorf("create temporary SVG: %w", err)
	}
	svgPath := svgFile.Name()
	defer os.Remove(svgPath) //nolint:errcheck

	sipsDoc := doc.Copy()
	prepareForSips(sipsDoc)
	if _, err := sipsDoc.WriteTo(svgFile); err != nil {
		_ = svgFile.Close()
		return fmt.Errorf("write temporary SVG: %w", err)
	}
	if err := svgFile.Close(); err != nil {
		return fmt.Errorf("close temporary SVG: %w", err)
	}

	cmd := exec.Command(sips, "-s", "format", "png", svgPath, "--out", output) //nolint:gosec
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run sips: %w", err)
	}
	return nil
}

// prepareForSips replaces styled tspan elements with independent text elements.
// CoreSVG ignores foreground fills on tspans, but honors the same attributes on
// text elements. Explicit terminal-cell positions keep each segment aligned
// without relying on CoreSVG to honor tspan positioning and colors.
func prepareForSips(doc *etree.Document) {
	root := doc.Root()
	if root == nil {
		return
	}
	textGroup := root.SelectElement("g")
	if textGroup == nil {
		return
	}
	fontSize := parseSVGLength(textGroup.SelectAttrValue("font-size", ""))
	if fontSize == 0 {
		fontSize = defaultFontSize
	}
	cellWidth := fontSize / fontHeightToWidthRatio
	prepareNestedSVGForSips(root)

	for _, line := range textGroup.SelectElements("text") {
		replacements := sipsTextElements(line, cellWidth)
		if len(replacements) == 0 {
			continue
		}

		index := line.Index()
		textGroup.RemoveChild(line)
		for i, replacement := range replacements {
			textGroup.InsertChildAt(index+i, replacement)
		}
	}
}

func sipsTextElements(line *etree.Element, cellWidth float64) []*etree.Element {
	type sourceSegment struct {
		end   int
		style *etree.Element
	}

	var text strings.Builder
	var sources []sourceSegment
	for _, token := range line.Child {
		var value string
		var style *etree.Element
		switch token := token.(type) {
		case *etree.CharData:
			value = token.Data
		case *etree.Element:
			value = token.Text()
			style = token
		default:
			continue
		}
		if value != "" {
			value = strings.ReplaceAll(value, "\u00a0", " ")
			text.WriteString(value)
			sources = append(sources, sourceSegment{end: text.Len(), style: style})
		}
	}

	var (
		column        int
		result        []*etree.Element
		sourceIndex   int
		pendingColumn int
		pendingStyle  *etree.Element
		pendingText   strings.Builder
	)
	flushPending := func() {
		if pendingText.Len() == 0 {
			return
		}
		result = append(result, newSipsTextElement(line, pendingStyle, pendingText.String(), pendingColumn, cellWidth))
		pendingText.Reset()
	}

	graphemes := uniseg.NewGraphemes(text.String())
	for graphemes.Next() {
		start, _ := graphemes.Positions()
		for sourceIndex < len(sources)-1 && start >= sources[sourceIndex].end {
			sourceIndex++
		}
		style := sources[sourceIndex].style
		grapheme := graphemes.Str()
		width := graphemes.Width()

		if width > 1 {
			flushPending()
			result = append(result, newSipsTextElement(line, style, grapheme, column, cellWidth))
		} else {
			if pendingText.Len() > 0 && pendingStyle != style {
				flushPending()
			}
			if pendingText.Len() == 0 && grapheme == " " {
				column += width
				continue
			}
			if pendingText.Len() == 0 {
				pendingColumn = column
				pendingStyle = style
			}
			pendingText.WriteString(grapheme)
		}
		column += width
	}
	flushPending()
	return result
}

func newSipsTextElement(line, style *etree.Element, text string, column int, cellWidth float64) *etree.Element {
	element := etree.NewElement("text")
	copyAttributes(element, line, nil)
	copyAttributes(element, style, map[string]bool{
		"dx":        true,
		"dy":        true,
		"x":         true,
		"xml:space": true,
		"y":         true,
	})
	element.CreateAttr("xml:space", "preserve")
	lineStart := parseSVGLength(line.SelectAttrValue("x", ""))
	element.CreateAttr("x", fmt.Sprintf("%.2fpx", lineStart+float64(column)*cellWidth))
	element.SetText(text)
	return element
}

func prepareNestedSVGForSips(root *etree.Element) {
	for _, nested := range root.SelectElements("svg") {
		group := nested.Copy()
		group.Tag = "g"
		x := parseSVGLength(group.SelectAttrValue("x", ""))
		y := parseSVGLength(group.SelectAttrValue("y", ""))
		group.RemoveAttr("x")
		group.RemoveAttr("y")
		group.CreateAttr("transform", fmt.Sprintf("translate(%.2f %.2f)", x, y))

		index := nested.Index()
		root.RemoveChild(nested)
		root.InsertChildAt(index, group)
	}
}

func parseSVGLength(value string) float64 {
	length, _ := strconv.ParseFloat(strings.TrimSuffix(value, "px"), 64)
	return length
}

func copyAttributes(dst, src *etree.Element, skip map[string]bool) {
	if src == nil {
		return
	}
	for i := range src.Attr {
		attr := &src.Attr[i]
		if !skip[attr.FullKey()] {
			dst.CreateAttr(attr.FullKey(), attr.Value)
		}
	}
}
