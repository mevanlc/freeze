package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/aymanbagabas/go-udiff"
)

var binary = "./test/freeze-test"

func init() {
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
}

var (
	update = flag.Bool("update", false, "update golden files")
	png    = flag.Bool("png", false, "update pngs")
)

func TestMain(m *testing.M) {
	flag.Parse()
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	exit := m.Run()
	err = os.Remove(binary)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(exit)
}

func TestFreeze(t *testing.T) {
	cmd := exec.Command(binary)
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFreezeOutput(t *testing.T) {
	output := "artichoke-test.svg"
	defer os.Remove(output)

	cmd := exec.Command(binary, "test/input/artichoke.hs", "-o", output)
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFreezeHelp(t *testing.T) {
	out := bytes.Buffer{}
	cmd := exec.Command(binary)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		t.Fatal("unexpected error")
	}

	got := out.String()

	contains := []string{
		"Generate images of code and terminal output.",
		"freeze main.go [-o code.svg] [--flags]",
		"--theme", "Theme to use for syntax highlighting",
		"--border.color", "Border color.",
		"--shadow.blur", "Shadow Gaussian Blur.",
		"--font.family", "Font family to use for code.",
		"--rasterizer", "PNG rasterizer: auto, rsvg-pdf, rsvg, resvg, sips, or chromium.",
		"--ansi-layout", "ANSI text layout: rune or grapheme.",
		"--ansi-blocks", "ANSI block rendering: font or terminal.",
		"--scale", "Scale automatically sized PNG output (defaults to 4x, or 2x above 4096px).",
	}

	for _, c := range contains {
		if !strings.Contains(got, c) {
			t.Fatalf("expected %s to contain \"%s\"", got, c)
		}
	}
}

func TestPNGFlagValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "PNG only",
			args: []string{"test/input/artichoke.hs", "--rasterizer=chromium", "--output", "out.svg"},
			want: "--rasterizer requires PNG output",
		},
		{
			name: "scale must be positive",
			args: []string{"test/input/artichoke.hs", "--scale=0", "--output", "out.png"},
			want: "--scale must be greater than zero",
		},
		{
			name: "scale requires PNG",
			args: []string{"test/input/artichoke.hs", "--scale=2", "--output", "out.svg"},
			want: "--scale requires PNG output",
		},
		{
			name: "scale requires automatic dimensions",
			args: []string{"test/input/artichoke.hs", "--scale=2", "--width=800", "--output", "out.png"},
			want: "--scale requires automatic width and height",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binary, tc.args...)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("expected command to fail")
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected %q in output:\n%s", tc.want, output)
			}
		})
	}
}

func TestFreezeErrorFileMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this fails on windows for some reason")
	}

	out := bytes.Buffer{}
	cmd := exec.Command(binary, "this-file-does-not-exist")
	cmd.Stdout = &out
	err := cmd.Run()

	if err == nil {
		t.Fatal("expected error")
	}

	got := out.String()

	contains := []string{"ERROR", "File not found", "open this-file-does-not-exist: no such file or directory"}

	for _, c := range contains {
		if !strings.Contains(got, c) {
			t.Fatalf("expected %s to contain \"%s\"", got, c)
		}
	}
}

func TestFreezeConfigurations(t *testing.T) {
	tests := []struct {
		input  string
		flags  []string
		output string
	}{
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--config", "test/configurations/base.json"},
			output: "artichoke-base",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--config", "test/configurations/full.json"},
			output: "artichoke-full",
		},
		{
			input:  "test/input/eza.ansi",
			flags:  []string{"--config", "full"},
			output: "eza",
		},
		{
			flags:  []string{"--execute", `echo "Hello, world!"`},
			output: "execute",
		},
		{
			input:  "test/input/bubbletea.model",
			flags:  []string{"--language", "go", "--height", "800", "--width", "750", "--config", "full", "--window=false", "--show-line-numbers"},
			output: "bubbletea",
		},
		{
			input:  "test/input/layout.ansi",
			flags:  []string{},
			output: "layout",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--language", "haskell"},
			output: "haskell",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--theme", "dracula"},
			output: "dracula",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--border.radius", "8"},
			output: "border-radius",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--border.radius", "8", "--window"},
			output: "window",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--border.radius", "8", "--window", "--border.width", "1"},
			output: "border-width",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--border.radius", "8", "--window", "--border.width", "1", "--padding", "30,50,30,30"},
			output: "padding",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--border.radius", "8", "--window", "--border.width", "1", "--padding", "30,50,30,30", "--margin", "50,60,100,60"},
			output: "margin",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--config", "full"},
			output: "shadow",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--width", "1920", "--height", "1080"},
			output: "dimensions",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--margin", "50", "--width", "600", "--height", "300"},
			output: "dimensions-margin",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--margin", "50", "--width", "600", "--height", "300", "--show-line-numbers"},
			output: "dimensions-margin-line-numbers",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--padding", "50", "--width", "600", "--height", "300"},
			output: "dimensions-padding",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--config", "full", "--width", "600", "--height", "300"},
			output: "dimensions-config",
		},
		{
			input:  "test/input/goreleaser-full.yml",
			flags:  []string{"--config", "full", "--width", "600", "--height", "900"},
			output: "overflow",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--config", "full", "--lines", "4,8", "--show-line-numbers"},
			output: "lines",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--font.size", "28"},
			output: "font-size-28",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--font.size", "14"},
			output: "font-size-14",
		},
		{
			input:  "test/input/artichoke.hs",
			flags:  []string{"--line-height", "2"},
			output: "line-height-2",
		},
		{
			input:  "test/input/goreleaser-full.yml",
			flags:  []string{"--config", "full", "--height", "2000", "--show-line-numbers"},
			output: "overflow-line-numbers",
		},
		{
			input:  "test/input/helix.ansi",
			flags:  []string{"--background", "#0d1116"},
			output: "helix",
		},
		{
			input:  "test/input/glow.ansi",
			flags:  []string{},
			output: "glow",
		},
		{
			input:  "test/input/tab.go",
			flags:  []string{},
			output: "tab",
		},
		{
			input:  "test/input/wrap.go",
			flags:  []string{"--wrap", "80", "--width", "600"},
			output: "wrap",
		},
	}

	err := os.RemoveAll("test/output/svg")
	if err != nil {
		t.Fatal("unable to remove output files")
	}
	err = os.MkdirAll("test/output/svg", 0o755)
	if err != nil {
		t.Fatal("unable to create output directory")
	}
	err = os.MkdirAll("test/golden/svg", 0o755)
	if err != nil {
		t.Fatal("unable to create output directory")
	}
	err = os.MkdirAll("test/output/png", 0o755)
	if err != nil {
		t.Fatal("unable to create output directory")
	}

	for _, tc := range tests {
		t.Run(tc.output, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("this fails on windows for some reason")
			}

			// output SVG
			out := bytes.Buffer{}
			args := []string{tc.input}
			args = append(args, tc.flags...)
			args = append(args, "--output", "test/output/svg/"+tc.output+".svg")
			cmd := exec.Command(binary, args...)
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				t.Log(err)
				t.Log(out.String())
				t.Fatal("unexpected error")
			}
			gotfile := "test/output/svg/" + tc.output + ".svg"
			got, err := os.ReadFile(gotfile)
			if err != nil {
				t.Fatal("no output file for:", gotfile)
			}
			goldenfile := "test/golden/svg/" + tc.output + ".svg"
			if *update {
				if err := os.WriteFile(goldenfile, got, 0o644); err != nil {
					t.Log(err)
					t.Fatal("unexpected error")
				}
			}
			want, err := os.ReadFile(goldenfile)
			if err != nil {
				t.Fatal("no golden file for:", goldenfile)
			}
			if normalizeNewlines(want) != normalizeNewlines(got) {
				t.Log(udiff.Unified("want", "got", normalizeNewlines(want), normalizeNewlines(got)))
				t.Fatalf("%s != %s", goldenfile, gotfile)
			}

			// output PNG
			if png != nil && *png {
				out = bytes.Buffer{}
				args = []string{tc.input}
				args = append(args, tc.flags...)
				args = append(args, "--output", "test/output/png/"+tc.output+".png")
				cmd = exec.Command(binary, args...)
				cmd.Stdout = &out
				err = cmd.Run()
				if err != nil {
					t.Log(err)
					t.Log(out.String())
					t.Fatal("unexpected error")
				}
			}
		})
	}
}

func normalizeNewlines[T string | []byte](s T) string {
	return strings.ReplaceAll(string(s), "\r\n", "\n")
}
