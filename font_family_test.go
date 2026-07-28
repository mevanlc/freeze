package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	freezefont "github.com/charmbracelet/freeze/font"
)

func TestPreprocessFontFamily(t *testing.T) {
	called := 0
	list := func() []string {
		called++
		return []string{
			"Hack Nerd Font",
			"MesloLGSDZ Nerd Font Mono",
			"Apple Color Emoji",
		}
	}

	got, err := preprocessFontFamilyWith(
		` /(?i)Nerd.*Font.*Mono/ , /(?i)Nerd.*Font/ ,, monospace, /missing/, /(?i)emoji/ `,
		list,
	)
	if err != nil {
		t.Fatal(err)
	}
	const want = `"MesloLGSDZ Nerd Font Mono","Hack Nerd Font",monospace,"Apple Color Emoji"`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if called != 1 {
		t.Fatalf("font family lister called %d times, want 1", called)
	}
}

func TestPreprocessFontFamilyWithoutRegex(t *testing.T) {
	got, err := preprocessFontFamilyWith(` JetBrains Mono, , "Apple Color Emoji" `, func() []string {
		t.Fatal("font family lister called without a regex entry")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	const want = `JetBrains Mono,"Apple Color Emoji"`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPreprocessFontFamilyInvalidRegex(t *testing.T) {
	_, err := preprocessFontFamilyWith(`/[/`, func() []string { return nil })
	if err == nil || !strings.Contains(err.Error(), `invalid font family regex "/[/"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQuoteCSSFontFamily(t *testing.T) {
	const value = `A "quoted" \\ family`
	const want = `"A \"quoted\" \\\\ family"`
	if got := quoteCSSFontFamily(value); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEscapeXMLFontFamily(t *testing.T) {
	const value = `"A & B"`
	const want = `&quot;A &amp; B&quot;`
	if got := escapeXMLFontFamily(value); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFontFamiliesFromFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "JetBrainsMono.ttf")
	if err := os.WriteFile(filename, freezefont.JetBrainsMonoTTF, 0o600); err != nil {
		t.Fatal(err)
	}

	families := fontFamiliesFromFile(filename)
	if !slices.Contains(families, "JetBrains Mono") {
		t.Fatalf("font families are %q, want JetBrains Mono", families)
	}
}
