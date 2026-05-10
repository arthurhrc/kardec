package chart

import (
	"bytes"
	"fmt"
	"math"
)

// BarChart renders a vertical bar chart. One bar per Series entry,
// labelled along the X axis, with a numeric Y axis on the left.
//
//	bar := chart.Bar(chart.BarChart{
//	    Title:  "Q4 revenue",
//	    YLabel: "R$ thousands",
//	    Series: []chart.BarSeries{
//	        {Label: "Oct", Value: 12.5},
//	        {Label: "Nov", Value: 14.2},
//	        {Label: "Dec", Value: 11.8},
//	    },
//	})
//	doc.Image(bar.Render(400, 250)).Build()
type BarChart struct {
	Title   string
	YLabel  string
	Series  []BarSeries
	Palette []Color // optional; nil = DefaultPalette
}

// BarSeries is one bar's label + value.
type BarSeries struct {
	Label string
	Value float64
}

// Bar constructs a BarChart from data. Convenience for callers
// who prefer the function form over the literal.
func Bar(data BarChart) *BarChart { return &data }

// Render returns the SVG bytes for this chart at width × height
// points. The image is responsive: glyphs, ticks, and padding
// scale proportionally so a 200-pt thumbnail and a 600-pt poster
// share the same source.
func (b *BarChart) Render(width, height float64) []byte {
	var buf bytes.Buffer
	svgHeader(&buf, width, height)

	if len(b.Series) == 0 {
		svgFooter(&buf)
		return buf.Bytes()
	}

	// Layout regions — title strip on top, axis labels on left,
	// axis-tick labels on bottom. The body sits in the middle.
	const titleH, axisLeftW, axisBottomH = 26.0, 50.0, 28.0
	titleY := 18.0
	if b.Title == "" {
		titleY = 0
	}
	plotX := axisLeftW
	plotY := titleY
	plotW := width - plotX - 10
	plotH := height - plotY - axisBottomH
	if plotW < 50 || plotH < 50 {
		svgFooter(&buf)
		return buf.Bytes()
	}

	// Title.
	if b.Title != "" {
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="13" font-weight="bold" text-anchor="middle">%s</text>`,
			width/2, 16.0, escapeXML(b.Title))
	}

	// Y axis scale: max value rounded to a nice ceiling, 5 ticks.
	maxV := 0.0
	for _, s := range b.Series {
		if s.Value > maxV {
			maxV = s.Value
		}
	}
	yMax := niceMax(maxV)
	const ticks = 5
	for i := 0; i <= ticks; i++ {
		v := yMax * float64(i) / float64(ticks)
		y := plotY + plotH - plotH*float64(i)/float64(ticks)
		// Grid line.
		fmt.Fprintf(&buf,
			`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#dddddd" stroke-width="0.5"/>`,
			plotX, y, plotX+plotW, y)
		// Tick label.
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="9" text-anchor="end">%s</text>`,
			plotX-4, y+3, numberFormat(v))
	}

	// Y label, vertical.
	if b.YLabel != "" {
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="10" text-anchor="middle" transform="rotate(-90 %.2f %.2f)">%s</text>`,
			14.0, plotY+plotH/2, 14.0, plotY+plotH/2, escapeXML(b.YLabel))
	}

	// Bars. Allocate equal slots per series with a 20 % padding
	// between bars so each bar takes 80 % of its slot width.
	slot := plotW / float64(len(b.Series))
	barW := slot * 0.7
	for i, s := range b.Series {
		x := plotX + slot*float64(i) + (slot-barW)/2
		ratio := 0.0
		if yMax > 0 {
			ratio = math.Max(0, s.Value) / yMax
		}
		h := plotH * ratio
		y := plotY + plotH - h
		color := pickColor(b.Palette, i)
		fmt.Fprintf(&buf,
			`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			x, y, barW, h, color)
		// X tick label centred under the bar.
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="9" text-anchor="middle">%s</text>`,
			x+barW/2, plotY+plotH+12, escapeXML(s.Label))
		// Value above the bar (only when there's room).
		if h > 14 {
			fmt.Fprintf(&buf,
				`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="9" text-anchor="middle" fill="#444">%s</text>`,
				x+barW/2, y-3, numberFormat(s.Value))
		}
	}

	// Axis lines.
	fmt.Fprintf(&buf,
		`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#666666" stroke-width="0.8"/>`,
		plotX, plotY+plotH, plotX+plotW, plotY+plotH)
	fmt.Fprintf(&buf,
		`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#666666" stroke-width="0.8"/>`,
		plotX, plotY, plotX, plotY+plotH)

	svgFooter(&buf)
	return buf.Bytes()
}

// escapeXML replaces the five characters reserved in XML element
// bodies / attribute values: &, <, >, ", '.
func escapeXML(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
