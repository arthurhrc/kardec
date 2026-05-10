package chart

import (
	"bytes"
	"fmt"
)

// LineChart renders one or more series of (x, y) points connected
// by polylines. Each series carries its own label + color; a
// legend strip at the top labels them.
//
//	line := chart.Line(chart.LineChart{
//	    Title:  "Latency over time",
//	    XLabel: "minute",
//	    YLabel: "ms",
//	    Series: []chart.LineSeries{
//	        {Label: "p50", Points: []chart.Point{{X:1,Y:120}, {X:2,Y:130}, ...}},
//	        {Label: "p99", Points: []chart.Point{{X:1,Y:340}, {X:2,Y:380}, ...}},
//	    },
//	})
//	doc.Image(line.Render(450, 280)).Build()
type LineChart struct {
	Title   string
	XLabel  string
	YLabel  string
	Series  []LineSeries
	Palette []Color
}

// LineSeries is one labelled series of points.
type LineSeries struct {
	Label  string
	Points []Point
}

// Point is one (x, y) pair on a line chart.
type Point struct{ X, Y float64 }

// Line constructs a LineChart from data.
func Line(data LineChart) *LineChart { return &data }

// Render returns the SVG bytes for this chart at width × height
// points.
func (l *LineChart) Render(width, height float64) []byte {
	var buf bytes.Buffer
	svgHeader(&buf, width, height)

	if len(l.Series) == 0 {
		svgFooter(&buf)
		return buf.Bytes()
	}

	// Layout regions — title + optional legend on top, axes
	// surrounding the body.
	const axisLeftW, axisBottomH = 50.0, 32.0
	titleH := 0.0
	if l.Title != "" {
		titleH = 22
	}
	legendH := 18.0 // always reserve room for the legend
	plotX := axisLeftW
	plotY := titleH + legendH
	plotW := width - plotX - 12
	plotH := height - plotY - axisBottomH
	if plotW < 50 || plotH < 50 {
		svgFooter(&buf)
		return buf.Bytes()
	}

	// Title.
	if l.Title != "" {
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="13" font-weight="bold" text-anchor="middle">%s</text>`,
			width/2, 16.0, escapeXML(l.Title))
	}

	// Legend: dot + label per series, laid out left-to-right.
	legendY := titleH + 12
	legendX := plotX
	for i, s := range l.Series {
		color := pickColor(l.Palette, i)
		fmt.Fprintf(&buf,
			`<circle cx="%.2f" cy="%.2f" r="3" fill="%s"/>`,
			legendX, legendY, color)
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="10">%s</text>`,
			legendX+8, legendY+3, escapeXML(s.Label))
		legendX += 50 + float64(len(s.Label))*5
	}

	// Domain: min/max across all series.
	var xMin, xMax, yMin, yMax float64
	first := true
	for _, s := range l.Series {
		for _, p := range s.Points {
			if first {
				xMin, xMax, yMin, yMax = p.X, p.X, p.Y, p.Y
				first = false
				continue
			}
			if p.X < xMin {
				xMin = p.X
			}
			if p.X > xMax {
				xMax = p.X
			}
			if p.Y < yMin {
				yMin = p.Y
			}
			if p.Y > yMax {
				yMax = p.Y
			}
		}
	}
	if first {
		// No points anywhere — render an empty plot box.
		fmt.Fprintf(&buf, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="none" stroke="#cccccc"/>`,
			plotX, plotY, plotW, plotH)
		svgFooter(&buf)
		return buf.Bytes()
	}
	if xMax == xMin {
		xMax = xMin + 1
	}
	yMax = niceMax(yMax)
	if yMax == yMin {
		yMax = yMin + 1
	}

	// Y grid + tick labels.
	const ticks = 5
	for i := 0; i <= ticks; i++ {
		v := yMax * float64(i) / float64(ticks)
		y := plotY + plotH - plotH*float64(i)/float64(ticks)
		fmt.Fprintf(&buf,
			`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#eeeeee" stroke-width="0.5"/>`,
			plotX, y, plotX+plotW, y)
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="9" text-anchor="end">%s</text>`,
			plotX-4, y+3, numberFormat(v))
	}

	// X axis labels at the data points' X positions.
	for i := 0; i <= ticks; i++ {
		v := xMin + (xMax-xMin)*float64(i)/float64(ticks)
		x := plotX + plotW*float64(i)/float64(ticks)
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="9" text-anchor="middle">%s</text>`,
			x, plotY+plotH+12, numberFormat(v))
	}

	// Axis labels.
	if l.YLabel != "" {
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="10" text-anchor="middle" transform="rotate(-90 %.2f %.2f)">%s</text>`,
			14.0, plotY+plotH/2, 14.0, plotY+plotH/2, escapeXML(l.YLabel))
	}
	if l.XLabel != "" {
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="10" text-anchor="middle">%s</text>`,
			plotX+plotW/2, height-6, escapeXML(l.XLabel))
	}

	// Polylines per series.
	for i, s := range l.Series {
		if len(s.Points) == 0 {
			continue
		}
		color := pickColor(l.Palette, i)
		var pts bytes.Buffer
		for j, p := range s.Points {
			x := plotX + plotW*(p.X-xMin)/(xMax-xMin)
			y := plotY + plotH - plotH*(p.Y-yMin)/(yMax-yMin)
			if j > 0 {
				pts.WriteByte(' ')
			}
			fmt.Fprintf(&pts, "%.2f,%.2f", x, y)
		}
		fmt.Fprintf(&buf,
			`<polyline points="%s" fill="none" stroke="%s" stroke-width="1.5"/>`,
			pts.String(), color)
		// Point markers.
		for _, p := range s.Points {
			x := plotX + plotW*(p.X-xMin)/(xMax-xMin)
			y := plotY + plotH - plotH*(p.Y-yMin)/(yMax-yMin)
			fmt.Fprintf(&buf,
				`<circle cx="%.2f" cy="%.2f" r="2.5" fill="%s"/>`,
				x, y, color)
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
