package main

import (
	"testing"

	"github.com/beevik/etree"
)

func TestPrepareForSips(t *testing.T) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<svg xmlns="http://www.w3.org/2000/svg">
<g font-family="Mono,Apple Color Emoji" font-size="14.00px" fill="#ffffff">
<text x="100.00px" y="20.00px" xml:space="preserve">AA<tspan fill="#ff0000"> 🧑‍</tspan><tspan fill="#ff0000" dx="2.80px">💻 END</tspan></text>
</g>
<svg x="10.00px" y="12.00px"><circle cx="5" cy="5" r="5" fill="#ff0000"/></svg>
</svg>`); err != nil {
		t.Fatal(err)
	}

	prepareForSips(doc)

	root := doc.Root()
	if nested := root.SelectElement("svg"); nested != nil {
		t.Fatal("nested svg was not replaced")
	}
	groups := root.SelectElements("g")
	if len(groups) != 2 {
		t.Fatalf("expected text and window-control groups, got %d", len(groups))
	}
	if got := groups[1].SelectAttrValue("transform", ""); got != "translate(10.00 12.00)" {
		t.Fatalf("unexpected window-control transform: %q", got)
	}

	texts := groups[0].SelectElements("text")
	if len(texts) != 3 {
		t.Fatalf("expected three text runs, got %d", len(texts))
	}
	want := []struct {
		text string
		x    string
		fill string
	}{
		{text: "AA", x: "100.00px"},
		{text: "🧑‍💻", x: "125.00px", fill: "#ff0000"},
		{text: "END", x: "150.00px", fill: "#ff0000"},
	}
	for i, expected := range want {
		if got := texts[i].Text(); got != expected.text {
			t.Errorf("text %d: got %q, want %q", i, got, expected.text)
		}
		if got := texts[i].SelectAttrValue("x", ""); got != expected.x {
			t.Errorf("x %d: got %q, want %q", i, got, expected.x)
		}
		if got := texts[i].SelectAttrValue("fill", ""); got != expected.fill {
			t.Errorf("fill %d: got %q, want %q", i, got, expected.fill)
		}
		if texts[i].SelectElement("tspan") != nil {
			t.Errorf("text %d still contains a tspan", i)
		}
	}
}
