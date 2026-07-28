package main

import (
	"strings"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultPNGScale       = 4.0
	largePNGScale         = 2.0
	maxAutoPNGDimensionPx = 4096.0
)

func cliFlagSet(ctx *kong.Context, name string) bool {
	for _, path := range ctx.Path {
		if path.Flag != nil && path.Flag.Name == name && !path.Resolved {
			return true
		}
	}
	return false
}

func logicalAutoPNGDimensions(config Config, strippedInput string, isANSI bool, sourceHeight int) (float64, float64) {
	margin := expandMargin(config.Margin, 1)
	padding := expandPadding(config.Padding, 1)

	tabWidth := 4
	if isANSI {
		tabWidth = 6
	}
	longestLine := lipgloss.Width(strings.ReplaceAll(strippedInput, "\t", strings.Repeat(" ", tabWidth)))
	width := float64(longestLine+1)*(config.Font.Size/fontHeightToWidthRatio) +
		padding[left] + padding[right] + margin[left] + margin[right]
	if config.ShowLineNumbers {
		width += config.Font.Size * 3
	}

	height := float64(sourceHeight) * (config.Font.Size / defaultFontSize) *
		(config.LineHeight / defaultLineHeight)
	height += padding[top] + padding[bottom] + margin[top] + margin[bottom]

	return width, height
}

func selectAutoPNGScale(requested, logicalWidth, logicalHeight float64) float64 {
	if requested > 0 {
		return requested
	}
	if logicalWidth*defaultPNGScale > maxAutoPNGDimensionPx ||
		logicalHeight*defaultPNGScale > maxAutoPNGDimensionPx {
		return largePNGScale
	}
	return defaultPNGScale
}
