# a Freeze fork

This is a fork of Freeze. Freeze generates images of code and terminal output.
See the [upstream repository](https://github.com/charmbracelet/freeze) for full
documentation.

## about this fork

This fork is based on upstream `main` and adds:

- selectable PNG rasterizers, including macOS `sips` and headless Chromium;
- an adaptive scale for automatically sized PNG output;
- grapheme-based ANSI text layout, so emoji ZWJ sequences stay in one span;
- terminal-cell rendering for ANSI block graphics;
- regular-expression font-family resolution against installed fonts;
- a built-in `terminal` configuration; and
- exact one-cell-per-column ANSI background spans, plus SGR 49 handling.

Everything else behaves like upstream Freeze unless noted below.

### PNG rasterizers

Use `--rasterizer=auto|rsvg-pdf|rsvg|resvg|sips|chromium` to choose how
Freeze converts its internal SVG to PNG:

- `auto` (the default) uses `rsvg-pdf` when both of its commands are available,
  then falls back to `rsvg` and finally Freeze's built-in `resvg` rasterizer.
- `rsvg-pdf` requires `rsvg-convert` and Poppler's `pdftocairo` on `PATH`. It
  renders SVG through librsvg's PDF path before producing a transparent PNG,
  preserving color emoji that direct librsvg PNG rendering reduces to monochrome
  glyph outlines.
- `rsvg` requires `rsvg-convert` on `PATH`.
- `resvg` uses the built-in rasterizer. It accepts only font-family lists that
  contain its bundled JetBrains Mono, and it refuses input with more than 1000
  `text`/`tspan` elements. Both cases fail with an error naming the alternative
  rasterizers.
- `sips` uses the macOS `sips` command and is rejected on other platforms.
  Freeze rewrites styled `tspan` elements into individually positioned `text`
  elements and flattens nested `svg` elements first, because CoreSVG ignores
  fills on `tspan`.
- `chromium` looks for Google Chrome, Google Chrome Beta, Microsoft Edge, or
  Chromium — in that order — and asks the first one it finds to screenshot the
  internal SVG headlessly, in a throwaway profile.

Any rasterizer choice other than `auto` requires PNG output, whether it comes
from `--rasterizer` or from a `"rasterizer"` key in a configuration file.

```bash
tmux capture-pane -t 40 -peN | freeze -c terminal --rasterizer=sips
tmux capture-pane -t 40 -peN | freeze -c terminal --rasterizer=chromium
tmux capture-pane -t 40 -peN | freeze -c terminal --rasterizer=rsvg-pdf
```

### PNG output scale

Automatically sized PNGs are rendered at 4x, unless 4x would push either
dimension past 4096 pixels, in which case Freeze uses 2x. Set `--scale` to
override that choice; `--scale=4` always restores the previous fixed 4x size.

```bash
freeze artichoke.hs -o artichoke.png --scale 4
```

`--scale` must be greater than zero, requires PNG output, and requires that both
width and height are left automatic.

### ANSI layout

ANSI input uses `--ansi-layout=rune` by default, which preserves upstream's SVG
output. `--ansi-layout=grapheme` groups user-perceived characters — emoji ZWJ
sequences, combining marks — into terminal cells and gives each cell an explicit
`x` position, so one displayed emoji is no longer split across several SVG text
spans.

```bash
tmux capture-pane -t 40 -peN | freeze -c full \
  --font.family 'MesloLGSDZ Nerd Font,Apple Color Emoji' \
  --ansi-layout=grapheme \
  --rasterizer=chromium
```

### ANSI block rendering

`--ansi-blocks=terminal` draws terminal-cell graphics as SVG rectangles instead
of glyphs, independent of the selected font. Adjacent full and fractional blocks
then meet exactly at cell boundaries, like Kitty and Ghostty, while line spacing
is unchanged. For PNG output the rectangle edges are snapped to whole pixels.
The default, `--ansi-blocks=font`, preserves upstream's font-based output.

Terminal rendering covers Unicode Block Elements (including eighths, quadrants,
and the three shade levels, which become fill patterns), Braille patterns,
sextants, octants, and related legacy-computing grid fills. Other symbols,
including classic box-drawing lines and curved legacy-computing characters,
continue to use the selected font.

```bash
tmux capture-pane -t 40 -peN | freeze -c full \
  --font.family 'MesloLGSDZ Nerd Font,Apple Color Emoji' \
  --ansi-layout=grapheme \
  --ansi-blocks=terminal \
  --rasterizer=chromium
```

### Font families by regular expression

`--font.family` entries are split on commas, trimmed, and empty entries are
dropped. An entry wrapped in `/` is treated as a Go regular expression and
replaced by the first installed font family it matches; unmatched expressions
are omitted entirely. Matched names are quoted in the generated CSS, while
literal entries are passed through as written. This makes fallback lists
portable across machines without knowing the exact installed font names:

```bash
freeze artichoke.hs \
  --font.family '/(?i)Nerd.*Font.*Mono/,monospace,/(?i)emoji/'
```

Installed families are read from the font files themselves, preferring the
typographic family name.

### The `terminal` configuration

`freeze -c terminal` is a built-in configuration for terminal captures. It
matches `full` except that it drops the window controls and shadow, uses a pure
black background, sets `ansi_blocks` to `terminal`, and resolves its font family
from a regex list that picks up an installed Nerd Font and emoji font.

```bash
tmux capture-pane -pet 1 | freeze -c terminal -o pane.png
```

`ansi_layout`, `ansi_blocks`, and `rasterizer` can all be set in any
configuration file, alongside the upstream keys.

### ANSI background spans

A background span is now exactly one cell wide per column it covers; upstream
added an extra half cell, which made adjacent spans overlap. SGR 49
(default background) also ends the current span, which upstream ignored.

## building

Freeze requires Go 1.25.8 or newer.

```bash
go build -o freeze .
```

Run the tests with `go test ./...` or `make test`.

## license

[MIT](https://github.com/charmbracelet/freeze/raw/main/LICENSE), unchanged from
upstream.
