package main

import (
	"encoding/json"
	"testing"

	"github.com/alecthomas/kong"
)

func TestConfig(t *testing.T) {
	dir := "configurations"

	entries, err := configs.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 {
		t.Fatal(entries)
	}

	for _, entry := range entries {
		f, err := configs.Open(dir + "/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		_, err = kong.JSON(f)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTerminalConfig(t *testing.T) {
	f, err := configs.Open("configurations/terminal.json")
	if err != nil {
		t.Fatal(err)
	}

	var config Config
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		t.Fatal(err)
	}

	if config.Window {
		t.Fatal("terminal config enables window controls")
	}
	if config.Shadow != (Shadow{}) {
		t.Fatalf("terminal config shadow is %#v, want none", config.Shadow)
	}
	if config.Background != "#000000" {
		t.Fatalf("terminal config background is %q, want #000000", config.Background)
	}
	if config.ANSIBlocks != ansiBlocksTerminal {
		t.Fatalf("terminal config ANSI blocks is %q, want %q", config.ANSIBlocks, ansiBlocksTerminal)
	}
	const family = `/(?i)Nerd.*Font.*Mono/,/(?i)Nerd.*Font/,monospace,/.{3}NF$/,/(?i)emoji/`
	if config.Font.Family != family {
		t.Fatalf("terminal config font family is %q, want %q", config.Font.Family, family)
	}
}
