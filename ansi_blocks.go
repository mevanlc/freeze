package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/beevik/etree"
)

// ANSIBlocks controls whether block-like ANSI glyphs are rendered by the font
// or as terminal-cell graphics.
type ANSIBlocks string

const (
	ansiBlocksFont     ANSIBlocks = "font"
	ansiBlocksTerminal ANSIBlocks = "terminal"
)

type unitRect struct {
	x      float64
	y      float64
	width  float64
	height float64
}

type cellGraphic struct {
	rects []unitRect
	shade int
}

func rect(x, y, width, height float64) unitRect {
	return unitRect{x: x, y: y, width: width, height: height}
}

func (p *dispatcher) renderANSIBlock(r rune, style *etree.Element) bool {
	if p.config.ANSIBlocks != ansiBlocksTerminal {
		return false
	}

	graphic, ok := ansiCellGraphic(r)
	if !ok {
		return false
	}

	p.row = clamp(p.row, 0, len(p.lines)-1)
	line := p.lines[p.row]
	cellWidth := p.config.Font.Size * p.scale / fontHeightToWidthRatio
	cellHeight := p.config.Font.Size * p.config.LineHeight
	x := parseSVGLength(line.SelectAttrValue("x", "")) + float64(p.col)*cellWidth
	if p.config.ShowLineNumbers {
		x += p.config.Font.Size * 3 * p.scale
	}

	y := float64(p.row)*cellHeight + p.config.Padding[top] + p.config.Margin[top]
	if baseline := line.SelectAttrValue("y", ""); baseline != "" {
		y = parseSVGLength(baseline) - cellHeight
	}

	fill := style.SelectAttrValue("fill", "")
	if fill == "" {
		fill = p.svg.SelectAttrValue("fill", "black")
	}

	if graphic.shade != 0 {
		patternID := p.ansiShadePattern(fill, graphic.shade, cellWidth, cellHeight)
		p.addANSIBlockRect(x, y, cellWidth, cellHeight, "url(#"+patternID+")")
		return true
	}

	for _, piece := range graphic.rects {
		p.addANSIBlockRect(
			x+piece.x*cellWidth,
			y+piece.y*cellHeight,
			piece.width*cellWidth,
			piece.height*cellHeight,
			fill,
		)
	}
	return true
}

func (p *dispatcher) addANSIBlockRect(x, y, width, height float64, fill string) {
	if strings.HasSuffix(p.config.Output, ".png") {
		right := math.Round(x + width)
		bottom := math.Round(y + height)
		x = math.Round(x)
		y = math.Round(y)
		width = right - x
		height = bottom - y
	}
	if width <= 0 || height <= 0 {
		return
	}
	rectangle := etree.NewElement("rect")
	rectangle.CreateAttr("x", fmt.Sprintf("%.5fpx", x))
	rectangle.CreateAttr("y", fmt.Sprintf("%.5fpx", y))
	rectangle.CreateAttr("width", fmt.Sprintf("%.5fpx", width))
	rectangle.CreateAttr("height", fmt.Sprintf("%.5fpx", height))
	rectangle.CreateAttr("fill", fill)
	p.ansiBlockGroup().AddChild(rectangle)
}

func (p *dispatcher) ansiBlockGroup() *etree.Element {
	if p.blockGroup == nil {
		p.blockGroup = etree.NewElement("g")
		p.blockGroup.CreateAttr("class", "ansi-blocks")
		p.blockGroup.CreateAttr("shape-rendering", "crispEdges")
		p.svg.InsertChildAt(0, p.blockGroup)
	}
	return p.blockGroup
}

func (p *dispatcher) ansiShadePattern(fill string, shade int, cellWidth, cellHeight float64) string {
	if p.shadePatterns == nil {
		p.shadePatterns = make(map[string]string)
	}
	key := fmt.Sprintf("%d:%s", shade, fill)
	if id, ok := p.shadePatterns[key]; ok {
		return id
	}

	if p.shadeDefs == nil {
		p.shadeDefs = etree.NewElement("defs")
		p.ansiBlockGroup().InsertChildAt(0, p.shadeDefs)
	}

	p.nextShadePattern++
	id := fmt.Sprintf("ansiShade%d", p.nextShadePattern)
	p.shadePatterns[key] = id

	patternWidth := cellWidth / 2
	patternHeight := cellHeight / 4
	pixelWidth := patternWidth / 2
	pixelHeight := patternHeight / 2
	pattern := etree.NewElement("pattern")
	pattern.CreateAttr("id", id)
	pattern.CreateAttr("patternUnits", "userSpaceOnUse")
	pattern.CreateAttr("width", fmt.Sprintf("%.5f", patternWidth))
	pattern.CreateAttr("height", fmt.Sprintf("%.5f", patternHeight))
	pattern.CreateAttr("shape-rendering", "crispEdges")

	addPixel := func(x, y float64) {
		pixel := etree.NewElement("rect")
		pixel.CreateAttr("x", fmt.Sprintf("%.5f", x))
		pixel.CreateAttr("y", fmt.Sprintf("%.5f", y))
		pixel.CreateAttr("width", fmt.Sprintf("%.5f", pixelWidth))
		pixel.CreateAttr("height", fmt.Sprintf("%.5f", pixelHeight))
		pixel.CreateAttr("fill", fill)
		pattern.AddChild(pixel)
	}

	addPixel(0, 0)
	if shade >= 2 {
		addPixel(pixelWidth, pixelHeight)
	}
	if shade >= 3 {
		addPixel(pixelWidth, 0)
	}
	p.shadeDefs.AddChild(pattern)
	return id
}

func ansiCellGraphic(r rune) (cellGraphic, bool) {
	if graphic, ok := standardBlockGraphic(r); ok {
		return graphic, true
	}
	if r >= '\u2800' && r <= '\u28ff' {
		return cellGraphic{rects: brailleRects(uint8(r - '\u2800'))}, true
	}
	if mask, ok := sextantMask(r); ok {
		return cellGraphic{rects: gridRects(2, 3, mask, false)}, true
	}
	if mask, ok := octantMask(r); ok {
		return cellGraphic{rects: gridRects(2, 4, mask, false)}, true
	}

	switch {
	case r >= '\U0001fb70' && r <= '\U0001fb75':
		position := float64(r-'\U0001fb6f') / 8
		return cellGraphic{rects: []unitRect{rect(position, 0, 1.0/8, 1)}}, true
	case r >= '\U0001fb76' && r <= '\U0001fb7b':
		position := float64(r-'\U0001fb75') / 8
		return cellGraphic{rects: []unitRect{rect(0, position, 1, 1.0/8)}}, true
	case r == '\U0001fbce':
		return cellGraphic{rects: []unitRect{rect(0, 0, 2.0/3, 1)}}, true
	case r == '\U0001fbcf':
		return cellGraphic{rects: []unitRect{rect(0, 0, 1.0/3, 1)}}, true
	case r == '\U0001fbe4':
		return cellGraphic{rects: []unitRect{rect(1.0/4, 0, 1.0/2, 1.0/2)}}, true
	case r == '\U0001fbe5':
		return cellGraphic{rects: []unitRect{rect(1.0/4, 1.0/2, 1.0/2, 1.0/2)}}, true
	case r >= '\U0001cc21' && r <= '\U0001cc2f':
		return cellGraphic{rects: gridRects(2, 2, uint8(r-'\U0001cc21'+1), true)}, true
	case r >= '\U0001ce51' && r <= '\U0001ce8f':
		return cellGraphic{rects: gridRects(2, 3, uint8(r-'\U0001ce51'+1), true)}, true
	case r >= '\U0001ce90' && r <= '\U0001ce9f':
		position := int(r - '\U0001ce90')
		return cellGraphic{rects: []unitRect{rect(float64(position%4)/4, float64(position/4)/4, 1.0/4, 1.0/4)}}, true
	case r >= '\U0001cea0' && r <= '\U0001ceaf':
		return partialQuarterGraphic(r), true
	default:
		return cellGraphic{}, false
	}
}

func standardBlockGraphic(r rune) (cellGraphic, bool) {
	switch {
	case r == '\u2580':
		return cellGraphic{rects: []unitRect{rect(0, 0, 1, 1.0/2)}}, true
	case r >= '\u2581' && r <= '\u2587':
		height := float64(r-'\u2580') / 8
		return cellGraphic{rects: []unitRect{rect(0, 1-height, 1, height)}}, true
	case r == '\u2588':
		return cellGraphic{rects: []unitRect{rect(0, 0, 1, 1)}}, true
	case r >= '\u2589' && r <= '\u258f':
		width := float64('\u2590'-r) / 8
		return cellGraphic{rects: []unitRect{rect(0, 0, width, 1)}}, true
	case r == '\u2590':
		return cellGraphic{rects: []unitRect{rect(1.0/2, 0, 1.0/2, 1)}}, true
	case r >= '\u2591' && r <= '\u2593':
		return cellGraphic{shade: int(r - '\u2590')}, true
	case r == '\u2594':
		return cellGraphic{rects: []unitRect{rect(0, 0, 1, 1.0/8)}}, true
	case r == '\u2595':
		return cellGraphic{rects: []unitRect{rect(7.0/8, 0, 1.0/8, 1)}}, true
	}

	mask, ok := quadrantMask(r)
	if !ok {
		return cellGraphic{}, false
	}
	return cellGraphic{rects: gridRects(2, 2, mask, false)}, true
}

func quadrantMask(r rune) (uint8, bool) {
	switch r {
	case '\u2596':
		return 4, true
	case '\u2597':
		return 8, true
	case '\u2598':
		return 1, true
	case '\u2599':
		return 13, true
	case '\u259a':
		return 9, true
	case '\u259b':
		return 7, true
	case '\u259c':
		return 11, true
	case '\u259d':
		return 2, true
	case '\u259e':
		return 6, true
	case '\u259f':
		return 14, true
	default:
		return 0, false
	}
}

func gridRects(columns, rows int, mask uint8, separated bool) []unitRect {
	gapX, gapY := 0.0, 0.0
	if separated {
		gapX, gapY = 1.0/8, 1.0/8
	}
	width := (1 - float64(columns+1)*gapX) / float64(columns)
	height := (1 - float64(rows+1)*gapY) / float64(rows)
	result := make([]unitRect, 0, columns*rows)
	for row := 0; row < rows; row++ {
		for column := 0; column < columns; column++ {
			bit := uint(row*columns + column)
			if mask&(1<<bit) == 0 {
				continue
			}
			x := gapX + float64(column)*(width+gapX)
			y := gapY + float64(row)*(height+gapY)
			result = append(result, rect(x, y, width, height))
		}
	}
	return result
}

func brailleRects(mask uint8) []unitRect {
	positions := [...]struct{ column, row int }{
		{0, 0}, {0, 1}, {0, 2}, {1, 0},
		{1, 1}, {1, 2}, {0, 3}, {1, 3},
	}
	result := make([]unitRect, 0, 8)
	for bit, position := range positions {
		if mask&(1<<bit) == 0 {
			continue
		}
		x := 1.0/8 + float64(position.column)/2
		y := 1.0/16 + float64(position.row)/4
		result = append(result, rect(x, y, 1.0/4, 1.0/8))
	}
	return result
}

func sextantMask(r rune) (uint8, bool) {
	switch {
	case r >= '\U0001fb00' && r <= '\U0001fb13':
		return uint8(r - '\U0001fb00' + 1), true
	case r >= '\U0001fb14' && r <= '\U0001fb27':
		return uint8(r - '\U0001fb00' + 2), true
	case r >= '\U0001fb28' && r <= '\U0001fb3b':
		return uint8(r - '\U0001fb00' + 3), true
	default:
		return 0, false
	}
}

func octantMask(r rune) (uint8, bool) {
	switch r {
	case '\U0001fbe6':
		return 20, true
	case '\U0001fbe7':
		return 40, true
	}
	if r < '\U0001cd00' || r > '\U0001cde5' {
		return 0, false
	}

	target := int(r - '\U0001cd00')
	index := 0
	for mask := 1; mask < 255; mask++ {
		if excludedOctantMask(mask) {
			continue
		}
		if index == target {
			return uint8(mask), true
		}
		index++
	}
	return 0, false
}

// Unicode assigns octant code points in mask order, omitting shapes already
// available as older Block Elements or legacy-computing characters.
func excludedOctantMask(mask int) bool {
	switch mask {
	case 1, 2, 3, 5, 10, 15, 20, 40, 63, 64, 80, 85, 90, 95,
		128, 160, 165, 170, 175, 192, 240, 245, 250, 252:
		return true
	default:
		return false
	}
}

func partialQuarterGraphic(r rune) cellGraphic {
	shapes := [...]unitRect{
		rect(1.0/2, 3.0/4, 1.0/2, 1.0/4),
		rect(1.0/4, 3.0/4, 3.0/4, 1.0/4),
		rect(0, 3.0/4, 3.0/4, 1.0/4),
		rect(0, 3.0/4, 1.0/2, 1.0/4),
		rect(0, 1.0/2, 1.0/4, 1.0/2),
		rect(0, 1.0/4, 1.0/4, 3.0/4),
		rect(0, 0, 1.0/4, 3.0/4),
		rect(0, 0, 1.0/4, 1.0/2),
		rect(0, 0, 1.0/2, 1.0/4),
		rect(0, 0, 3.0/4, 1.0/4),
		rect(1.0/4, 0, 3.0/4, 1.0/4),
		rect(1.0/2, 0, 1.0/2, 1.0/4),
		rect(3.0/4, 0, 1.0/4, 1.0/2),
		rect(3.0/4, 0, 1.0/4, 3.0/4),
		rect(3.0/4, 1.0/4, 1.0/4, 3.0/4),
		rect(3.0/4, 1.0/2, 1.0/4, 1.0/2),
	}
	return cellGraphic{rects: []unitRect{shapes[int(r-'\U0001cea0')]}}
}
