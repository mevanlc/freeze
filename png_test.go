package main

import (
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
	if err == nil || !strings.Contains(err.Error(), "use rsvg, sips, or chromium") {
		t.Fatalf("got error %v, want a rasterizer recommendation", err)
	}
}
