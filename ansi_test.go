package main

import (
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
