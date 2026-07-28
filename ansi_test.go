package main

import (
	"math"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/charmbracelet/x/ansi"
)

func TestDispatcherGraphemeLayout(t *testing.T) {
	line := etree.NewElement("text")
	line.CreateAttr("x", "10.00px")
	group := etree.NewElement("g")
	group.AddChild(line)
	config := Config{
		ANSILayout: ansiLayoutGrapheme,
		Font: Font{
			Size: 14,
		},
	}
	d := dispatcher{
		lines:  []*etree.Element{line},
		svg:    group,
		config: &config,
		scale:  1,
	}

	for _, r := range "A🧑‍💻B" {
		d.Print(r)
	}
	d.Execute(ansi.LF)

	children := line.ChildElements()
	if len(children) != 4 {
		t.Fatalf("expected a base span and three grapheme spans, got %d", len(children))
	}
	want := []struct {
		text string
		x    string
	}{
		{text: "A", x: "10.00px"},
		{text: "🧑‍💻", x: "18.33px"},
		{text: "B", x: "35.00px"},
	}
	for i, expected := range want {
		child := children[i+1]
		if got := child.Text(); got != expected.text {
			t.Errorf("span %d text: got %q, want %q", i, got, expected.text)
		}
		if got := child.SelectAttrValue("x", ""); got != expected.x {
			t.Errorf("span %d x: got %q, want %q", i, got, expected.x)
		}
		if child.SelectAttr("dx") != nil {
			t.Errorf("span %d unexpectedly has a dx adjustment", i)
		}
	}
}

func TestDispatcherRuneLayoutRemainsDefault(t *testing.T) {
	line := etree.NewElement("text")
	group := etree.NewElement("g")
	group.AddChild(line)
	config := Config{Font: Font{Size: 14}}
	d := dispatcher{
		lines:  []*etree.Element{line},
		svg:    group,
		config: &config,
		scale:  1,
	}

	for _, r := range "🧑‍💻" {
		d.Print(r)
	}
	d.Execute(ansi.LF)

	children := line.ChildElements()
	if len(children) != 3 {
		t.Fatalf("expected legacy rune layout to produce three spans, got %d", len(children))
	}
	if got := children[1].Text(); got != "🧑‍" {
		t.Errorf("unexpected first legacy span: %q", got)
	}
	if got := children[2].Text(); got != "💻" {
		t.Errorf("unexpected second legacy span: %q", got)
	}
}

func TestDispatcherBackgroundsOccupyExactCells(t *testing.T) {
	line := etree.NewElement("text")
	group := etree.NewElement("g")
	group.AddChild(line)
	config := Config{
		ANSILayout: ansiLayoutGrapheme,
		Font:       Font{Size: 14},
		LineHeight: 1.2,
		Margin:     []float64{0, 0, 0, 0},
		Padding:    []float64{0, 0, 0, 0},
	}
	d := dispatcher{
		lines:  []*etree.Element{line},
		svg:    group,
		config: &config,
		scale:  1,
	}
	parser := ansi.NewParser()
	parser.SetHandler(ansi.Handler{
		Print:     d.Print,
		HandleCsi: d.CsiDispatch,
		Execute:   d.Execute,
	})
	parser.Parse([]byte("\x1b[48;2;255;0;0m \x1b[48;2;0;255;0m \x1b[49m|"))
	d.Execute(ansi.LF)

	cellWidth := config.Font.Size / fontHeightToWidthRatio
	for _, fill := range []string{"#ff0000", "#00ff00"} {
		rect := group.FindElement("rect[@fill='" + fill + "']")
		if rect == nil {
			t.Fatalf("missing background rectangle %s", fill)
		}
		width := parseSVGLength(rect.SelectAttrValue("width", ""))
		if math.Abs(width-cellWidth) > 0.00001 {
			t.Errorf("background %s width: got %f, want one cell (%f)", fill, width, cellWidth)
		}
	}

	green := group.FindElement("rect[@fill='#00ff00']")
	greenX := parseSVGLength(green.SelectAttrValue("x", ""))
	if math.Abs(greenX-cellWidth) > 0.01 {
		t.Errorf("second background x: got %f, want %f", greenX, cellWidth)
	}
}

func TestDispatcherTerminalBlocksFillCellsWithoutChangingLineHeight(t *testing.T) {
	first := etree.NewElement("text")
	first.CreateAttr("x", "20.00px")
	first.CreateAttr("y", "36.80px")
	second := etree.NewElement("text")
	second.CreateAttr("x", "20.00px")
	second.CreateAttr("y", "53.60px")
	group := etree.NewElement("g")
	group.CreateAttr("fill", "#c4c4c4")
	group.AddChild(first)
	group.AddChild(second)
	config := Config{
		ANSILayout: ansiLayoutGrapheme,
		ANSIBlocks: ansiBlocksTerminal,
		Font:       Font{Size: 14},
		LineHeight: 1.2,
		Margin:     []float64{0, 0, 0, 0},
		Padding:    []float64{20, 0, 0, 20},
	}
	d := dispatcher{
		lines:  []*etree.Element{first, second},
		svg:    group,
		config: &config,
		scale:  1,
	}
	parser := ansi.NewParser()
	parser.SetHandler(ansi.Handler{
		Print:     d.Print,
		HandleCsi: d.CsiDispatch,
		Execute:   d.Execute,
	})
	parser.Parse([]byte("\x1b[38;2;255;0;0m██"))
	d.Execute(ansi.LF)
	parser.Parse([]byte("\x1b[38;2;255;0;0m█"))
	d.Execute(ansi.LF)

	blocks := group.FindElement("g[@class='ansi-blocks']").SelectElements("rect")
	if len(blocks) != 3 {
		t.Fatalf("expected three full-cell rectangles, got %d", len(blocks))
	}
	cellWidth := config.Font.Size / fontHeightToWidthRatio
	cellHeight := config.Font.Size * config.LineHeight
	rows := map[float64]int{}
	for _, block := range blocks {
		if got := block.SelectAttrValue("fill", ""); got != "#ff0000" {
			t.Errorf("block fill: got %q, want #ff0000", got)
		}
		if got := parseSVGLength(block.SelectAttrValue("width", "")); math.Abs(got-cellWidth) > 0.00001 {
			t.Errorf("block width: got %f, want %f", got, cellWidth)
		}
		if got := parseSVGLength(block.SelectAttrValue("height", "")); math.Abs(got-cellHeight) > 0.00001 {
			t.Errorf("block height: got %f, want %f", got, cellHeight)
		}
		rows[parseSVGLength(block.SelectAttrValue("y", ""))]++
	}
	if rows[20] != 2 || rows[36.8] != 1 {
		t.Errorf("block rows: got %#v, want two at 20 and one at 36.8", rows)
	}
	if math.Abs((20+cellHeight)-36.8) > 0.00001 {
		t.Fatal("test setup does not describe touching terminal rows")
	}
	for _, line := range []*etree.Element{first, second} {
		for _, span := range line.ChildElements() {
			if strings.ContainsRune(span.Text(), '█') {
				t.Error("terminal block should be replaced by a cell-advancing space in SVG text")
			}
		}
	}
}

func TestANSICellGraphicFamilies(t *testing.T) {
	tests := []struct {
		name      string
		r         rune
		wantRects int
		wantShade int
	}{
		{name: "full block", r: '█', wantRects: 1},
		{name: "light shade", r: '░', wantShade: 1},
		{name: "braille", r: '⣿', wantRects: 8},
		{name: "first sextant", r: '\U0001fb00', wantRects: 1},
		{name: "last sextant", r: '\U0001fb3b', wantRects: 5},
		{name: "first octant", r: '\U0001cd00', wantRects: 1},
		{name: "last octant", r: '\U0001cde5', wantRects: 7},
		{name: "middle left quarter", r: '\U0001fbe6', wantRects: 2},
		{name: "separated sextant", r: '\U0001ce51', wantRects: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			graphic, ok := ansiCellGraphic(tc.r)
			if !ok {
				t.Fatal("expected terminal-cell graphic")
			}
			if len(graphic.rects) != tc.wantRects {
				t.Errorf("rect count: got %d, want %d", len(graphic.rects), tc.wantRects)
			}
			if graphic.shade != tc.wantShade {
				t.Errorf("shade: got %d, want %d", graphic.shade, tc.wantShade)
			}
		})
	}

	for _, r := range []rune{'A', '─'} {
		if _, ok := ansiCellGraphic(r); ok {
			t.Errorf("%U should remain font-rendered", r)
		}
	}
}

func TestANSICellGraphicRangeCoverage(t *testing.T) {
	ranges := [][2]rune{
		{'\u2580', '\u259f'},
		{'\u2800', '\u28ff'},
		{'\U0001fb00', '\U0001fb3b'},
		{'\U0001fb70', '\U0001fb7b'},
		{'\U0001fbce', '\U0001fbcf'},
		{'\U0001fbe4', '\U0001fbe7'},
		{'\U0001cc21', '\U0001cc2f'},
		{'\U0001cd00', '\U0001cde5'},
		{'\U0001ce51', '\U0001ce8f'},
		{'\U0001ce90', '\U0001ceaf'},
	}
	for _, bounds := range ranges {
		for r := bounds[0]; r <= bounds[1]; r++ {
			if _, ok := ansiCellGraphic(r); !ok {
				t.Errorf("missing terminal-cell graphic for %U", r)
			}
		}
	}
}

func TestDispatcherTerminalBlocksShareSnappedPNGEdges(t *testing.T) {
	line := etree.NewElement("text")
	line.CreateAttr("x", "0.00px")
	line.CreateAttr("y", "67.20px")
	group := etree.NewElement("g")
	group.AddChild(line)
	config := Config{
		ANSIBlocks: ansiBlocksTerminal,
		Output:     "blocks.png",
		Font:       Font{Size: 14},
		LineHeight: 4.8,
		Margin:     []float64{0, 0, 0, 0},
		Padding:    []float64{0, 0, 0, 0},
	}
	d := dispatcher{
		lines:  []*etree.Element{line},
		svg:    group,
		config: &config,
		scale:  4,
	}
	d.Print('█')
	d.Print('█')

	blocks := group.FindElement("g[@class='ansi-blocks']").SelectElements("rect")
	if len(blocks) != 2 {
		t.Fatalf("expected two full-cell rectangles, got %d", len(blocks))
	}
	left := blocks[0]
	right := blocks[1]
	leftEdge := parseSVGLength(left.SelectAttrValue("x", "")) + parseSVGLength(left.SelectAttrValue("width", ""))
	rightEdge := parseSVGLength(right.SelectAttrValue("x", ""))
	if leftEdge != rightEdge {
		t.Errorf("adjacent PNG blocks do not share an edge: left ends at %f, right starts at %f", leftEdge, rightEdge)
	}
}

func TestDispatcherTerminalBlocksPreserveRuneLayoutCursor(t *testing.T) {
	line := etree.NewElement("text")
	line.CreateAttr("x", "10.00px")
	line.CreateAttr("y", "16.80px")
	group := etree.NewElement("g")
	group.CreateAttr("fill", "#c4c4c4")
	group.AddChild(line)
	config := Config{
		ANSIBlocks: ansiBlocksTerminal,
		Font:       Font{Size: 14},
		LineHeight: 1.2,
		Margin:     []float64{0, 0, 0, 0},
		Padding:    []float64{0, 0, 0, 0},
	}
	d := dispatcher{
		lines:  []*etree.Element{line},
		svg:    group,
		config: &config,
		scale:  1,
	}
	for _, r := range "A█B" {
		d.Print(r)
	}

	span := line.SelectElement("tspan")
	if span == nil {
		t.Fatal("missing rune-layout text span")
	}
	if span.Text() != "A B" {
		t.Fatalf("rune-layout text: got %q, want %q", span.Text(), "A B")
	}
	block := group.FindElement("g[@class='ansi-blocks']/rect")
	if block == nil {
		t.Fatal("missing terminal block rectangle")
	}
	wantX := 10 + config.Font.Size/fontHeightToWidthRatio
	if got := parseSVGLength(block.SelectAttrValue("x", "")); math.Abs(got-wantX) > 0.00001 {
		t.Errorf("block x: got %f, want %f", got, wantX)
	}
}
