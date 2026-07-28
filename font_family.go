package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	findfont "github.com/flopp/go-findfont"
	"golang.org/x/image/font/sfnt"
)

type fontFamilyLister func() []string

func preprocessFontFamily(value string) (string, error) {
	return preprocessFontFamilyWith(value, installedFontFamilies)
}

func preprocessFontFamilyWith(value string, list fontFamilyLister) (string, error) {
	var (
		families       []string
		familiesLoaded bool
		output         []string
	)

	for entry := range strings.SplitSeq(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		pattern, ok := fontFamilyRegex(entry)
		if !ok {
			output = append(output, entry)
			continue
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid font family regex %q: %w", entry, err)
		}
		if !familiesLoaded {
			families = list()
			familiesLoaded = true
		}
		for _, family := range families {
			if re.MatchString(family) {
				output = append(output, quoteCSSFontFamily(family))
				break
			}
		}
	}

	return strings.Join(output, ","), nil
}

func fontFamilyRegex(entry string) (string, bool) {
	if len(entry) < 2 || entry[0] != '/' || entry[len(entry)-1] != '/' {
		return "", false
	}
	return entry[1 : len(entry)-1], true
}

func quoteCSSFontFamily(family string) string {
	family = strings.ReplaceAll(family, `\`, `\\`)
	family = strings.ReplaceAll(family, `"`, `\"`)
	return `"` + family + `"`
}

func installedFontFamilies() []string {
	fontFiles := findfont.List()
	seenFamilies := make(map[string]struct{}, len(fontFiles))
	families := make([]string, 0, len(fontFiles))

	add := func(family string) {
		family = strings.TrimSpace(family)
		if family == "" {
			return
		}
		if _, ok := seenFamilies[family]; ok {
			return
		}
		seenFamilies[family] = struct{}{}
		families = append(families, family)
	}

	for _, filename := range fontFiles {
		for _, family := range fontFamiliesFromFile(filename) {
			add(family)
		}
	}

	return families
}

func fontFamiliesFromFile(filename string) []string {
	f, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer f.Close() //nolint:errcheck

	collection, err := sfnt.ParseCollectionReaderAt(f)
	if err != nil {
		return nil
	}

	families := make([]string, 0, collection.NumFonts())
	for i := range collection.NumFonts() {
		font, err := collection.Font(i)
		if err != nil {
			continue
		}
		family, err := font.Name(nil, sfnt.NameIDTypographicFamily)
		if err != nil || family == "" {
			family, err = font.Name(nil, sfnt.NameIDFamily)
		}
		if err == nil && family != "" {
			families = append(families, family)
		}
	}
	return families
}
