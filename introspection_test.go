package main

import (
	"bytes"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
)

func TestListTemplates(t *testing.T) {
	output := runFreeze(t, "--list-templates")
	if got, want := string(output), "base\nfull\nterminal\n"; got != want {
		t.Fatalf("unexpected template list:\n%s", got)
	}
}

func TestPrintTemplate(t *testing.T) {
	output := runFreeze(t, "--print-template", "terminal")
	want, err := configs.ReadFile("configurations/terminal.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, want) {
		t.Fatalf("printed template differs from embedded template:\n%s", output)
	}
}

func TestListLanguages(t *testing.T) {
	lines := outputLines(runFreeze(t, "--list-languages"))
	if !sort.StringsAreSorted(lines) {
		t.Fatal("language list is not sorted")
	}
	for _, name := range []string{"ANSI", "Go", "Haskell"} {
		if !containsLine(lines, name) {
			t.Errorf("language list does not contain %q", name)
		}
	}
}

func TestListThemes(t *testing.T) {
	output := runFreeze(t, "--list-themes")
	want := strings.Join(styles.Names(), "\n") + "\n"
	if got := string(output); got != want {
		t.Fatal("theme list does not match Chroma's sorted registry")
	}
}

func TestPrintTheme(t *testing.T) {
	output := runFreeze(t, "--print-theme", "charm")
	theme, err := chroma.NewXMLStyle(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("printed theme is not valid XML: %v\n%s", err, output)
	}
	if theme.Name != "charm" {
		t.Errorf("theme name is %q, want charm", theme.Name)
	}
	if len(theme.Types()) == 0 {
		t.Fatal("printed theme has no entries")
	}
}

func TestIntrospectionErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown template", args: []string{"--print-template", "missing"}, want: `unknown template "missing"`},
		{name: "unknown theme", args: []string{"--print-theme", "missing"}, want: `unknown theme "missing"`},
		{name: "combined actions", args: []string{"--list-themes", "--list-languages"}, want: "introspection flags may not be combined"},
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

func runFreeze(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(binary, args...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("freeze %s failed: %v", strings.Join(args, " "), err)
	}
	return output
}

func outputLines(output []byte) []string {
	return strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
}

func containsLine(lines []string, target string) bool {
	for _, line := range lines {
		if line == target {
			return true
		}
	}
	return false
}
