package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func handleIntrospection(config Config, output io.Writer) (bool, error) {
	actions := 0
	for _, selected := range []bool{
		config.ListTemplates,
		config.PrintTemplate != "",
		config.ListLanguages,
		config.ListThemes,
		config.PrintTheme != "",
	} {
		if selected {
			actions++
		}
	}
	if actions == 0 {
		return false, nil
	}
	if actions > 1 {
		return true, errors.New("introspection flags may not be combined")
	}

	switch {
	case config.ListTemplates:
		names, err := templateNames()
		if err != nil {
			return true, err
		}
		return true, writeLines(output, names)
	case config.PrintTemplate != "":
		contents, err := templateContents(config.PrintTemplate)
		if err != nil {
			return true, err
		}
		if _, err = output.Write(contents); err != nil {
			return true, fmt.Errorf("write template %q: %w", config.PrintTemplate, err)
		}
		return true, nil
	case config.ListLanguages:
		return true, writeLines(output, languageNames())
	case config.ListThemes:
		return true, writeLines(output, styles.Names())
	case config.PrintTheme != "":
		return true, writeTheme(output, config.PrintTheme)
	default:
		return false, nil
	}
}

func templateNames() ([]string, error) {
	entries, err := configs.ReadDir("configurations")
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".json"))
	}
	sort.Strings(names)
	return names, nil
}

func templateContents(name string) ([]byte, error) {
	names, err := templateNames()
	if err != nil {
		return nil, err
	}
	for _, candidate := range names {
		if name == candidate {
			contents, readErr := configs.ReadFile("configurations/" + name + ".json")
			if readErr != nil {
				return nil, fmt.Errorf("read template %q: %w", name, readErr)
			}
			return contents, nil
		}
	}
	return nil, fmt.Errorf("unknown template %q", name)
}

func languageNames() []string {
	names := lexers.Names(false)
	for _, name := range names {
		if strings.EqualFold(name, "ansi") {
			return names
		}
	}
	names = append(names, "ANSI")
	sort.Strings(names)
	return names
}

func writeLines(output io.Writer, lines []string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(output, line); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	return nil
}

func writeTheme(output io.Writer, name string) error {
	theme, ok := styles.Registry[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("unknown theme %q", name)
	}

	encoder := xml.NewEncoder(output)
	encoder.Indent("", "  ")
	if err := encoder.Encode(theme); err != nil {
		return fmt.Errorf("write theme %q: %w", name, err)
	}
	return nil
}
