// Package chart renders simple charts (bar, line, pie) into SVG
// byte streams that drop straight into a kardec.Document via
// Document.Image. Pure Go, no external dependencies — vector
// output, scales cleanly at any rendered size, and inherits the
// SVG → Form XObject path the renderer already has in place.
//
// Three chart types ship in v0.25:
//
//   chart.Bar(data).Render(width, height)   — vertical bars + axis
//   chart.Line(data).Render(width, height)  — connected points (multiple series)
//   chart.Pie(data).Render(width, height)   — sectors with auto color palette
//
// Each constructor returns a *ChartName carrying the data plus
// optional knobs (Title, AxisLabel, Palette, …). Render produces
// the SVG bytes; the caller embeds them via doc.Image(svgBytes)
// like any other image.
//
// The library is intentionally narrow. Anyone who outgrows it
// reaches for gonum/plot or go-echarts and embeds those as PNGs.
// The goal here is "pure Go, zero deps, ~5 lines of code from
// data to PDF" — the 80 % case for engineering reports and
// simple dashboards.
package chart

import (
	"bytes"
	"fmt"
	"math"
	"strings"
)

// Color is a device-RGB triple. Mirrors kardec.Color so callers
// can reuse the same color values across charts and body text.
type Color struct{ R, G, B uint8 }

// String returns the SVG hex form (`#rrggbb`).
func (c Color) String() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// DefaultPalette is the 8-color sequence Bar / Line / Pie use when
// no custom Palette is provided. Values borrowed from a Tableau-10
// subset adjusted for printer-safe contrast.
var DefaultPalette = []Color{
	{0x4E, 0x79, 0xA7}, // blue
	{0xF2, 0x8E, 0x2B}, // orange
	{0xE1, 0x57, 0x59}, // red
	{0x76, 0xB7, 0xB2}, // teal
	{0x59, 0xA1, 0x4F}, // green
	{0xED, 0xC9, 0x48}, // yellow
	{0xAF, 0x7A, 0xA1}, // purple
	{0xB0, 0x70, 0x80}, // mauve
}

// pickColor returns palette[i mod len(palette)]. Empty palette
// falls back to DefaultPalette.
func pickColor(palette []Color, i int) Color {
	p := palette
	if len(p) == 0 {
		p = DefaultPalette
	}
	return p[i%len(p)]
}

// svgHeader writes the SVG opening tag with the supplied
// dimensions plus a white background rectangle. Returns the byte
// position where the caller can append chart content.
func svgHeader(buf *bytes.Buffer, width, height float64) {
	fmt.Fprintf(buf,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%.2f" height="%.2f" viewBox="0 0 %.2f %.2f">`,
		width, height, width, height)
	fmt.Fprintf(buf,
		`<rect width="%.2f" height="%.2f" fill="white"/>`,
		width, height)
}

// svgFooter closes the SVG.
func svgFooter(buf *bytes.Buffer) {
	buf.WriteString(`</svg>`)
}

// numberFormat renders v with up to 2 decimal places, stripping
// trailing zeros. Used for tick labels on bar / line charts so
// 1000.00 reads as "1000" but 12.5 stays "12.5".
func numberFormat(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	// Strip trailing zeros + trailing dot.
	for strings.HasSuffix(s, "0") {
		s = s[:len(s)-1]
	}
	if strings.HasSuffix(s, ".") {
		s = s[:len(s)-1]
	}
	return s
}

// niceMax rounds v up to a "nice" axis ceiling: a multiple of 1,
// 2, or 5 times a power of 10. Ensures axis ticks land on round
// numbers (10, 20, 50, 100, 200, …) rather than odd fractions.
func niceMax(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(v))
	mantissa := v / math.Pow(10, exp)
	switch {
	case mantissa <= 1:
		mantissa = 1
	case mantissa <= 2:
		mantissa = 2
	case mantissa <= 5:
		mantissa = 5
	default:
		mantissa = 10
	}
	return mantissa * math.Pow(10, exp)
}
