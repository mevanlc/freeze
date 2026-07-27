package main

import (
	"image"
	imagepng "image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

func TestChromiumConvertWithExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX shell script")
	}

	dir := t.TempDir()
	browser := filepath.Join(dir, "chromium")
	fixture := filepath.Join(dir, "fixture.png")
	fixtureFile, err := os.Create(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := imagepng.Encode(fixtureFile, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	if err := fixtureFile.Close(); err != nil {
		t.Fatal(err)
	}
	argsPath := filepath.Join(dir, "args")
	t.Setenv("FREEZE_TEST_PNG", fixture)
	t.Setenv("FREEZE_TEST_ARGS", argsPath)
	script := `#!/bin/sh
for argument do
  case "$argument" in
    --screenshot=*) output=${argument#*=} ;;
  esac
done
printf '%s\n' "$@" > "$FREEZE_TEST_ARGS"
cp "$FREEZE_TEST_PNG" "$output"
sleep 10
`
	if err := os.WriteFile(browser, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<svg width="10.1" height="20.1" xmlns="http://www.w3.org/2000/svg"/>`); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "out.png")
	if err := chromiumConvertWithExecutable(browser, doc, 10.1, 20.1, output); err != nil {
		t.Fatal(err)
	}
	if !isCompletePNG(output) {
		t.Fatal("output is not a complete PNG")
	}

	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"--headless=new",
		"--force-device-scale-factor=1",
		"--window-size=11,21",
		"--screenshot=",
		"file://",
	} {
		if !strings.Contains(string(args), expected) {
			t.Errorf("expected arguments to contain %q; got:\n%s", expected, args)
		}
	}
}
