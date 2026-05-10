package chart

import (
	"bytes"
	"fmt"
	"math"
)

// PieChart renders a circle divided into sectors. One sector per
// Slice entry; the colors come from Palette (or DefaultPalette).
// Slice labels appear in a legend to the right of the circle.
//
//	pie := chart.Pie(chart.PieChart{
//	    Title: "Browser share",
//	    Slices: []chart.PieSlice{
//	        {Label: "Chrome",  Value: 65},
//	        {Label: "Safari",  Value: 18},
//	        {Label: "Firefox", Value: 12},
//	        {Label: "Other",   Value: 5},
//	    },
//	})
//	doc.Image(pie.Render(400, 280)).Build()
type PieChart struct {
	Title   string
	Slices  []PieSlice
	Palette []Color
}

// PieSlice is one sector's label + value. Values are normalised
// to the sum across all slices so callers can pass raw counts,
// percentages, or arbitrary units.
type PieSlice struct {
	Label string
	Value float64
}

// Pie constructs a PieChart from data.
func Pie(data PieChart) *PieChart { return &data }

// Render returns the SVG bytes for this chart at width × height
// points. The circle is positioned on the left half of the
// canvas; the legend occupies the right half.
func (p *PieChart) Render(width, height float64) []byte {
	var buf bytes.Buffer
	svgHeader(&buf, width, height)

	if len(p.Slices) == 0 {
		svgFooter(&buf)
		return buf.Bytes()
	}

	// Total — defines each slice's fraction.
	total := 0.0
	for _, s := range p.Slices {
		if s.Value > 0 {
			total += s.Value
		}
	}
	if total == 0 {
		svgFooter(&buf)
		return buf.Bytes()
	}

	titleH := 0.0
	if p.Title != "" {
		titleH = 22
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="13" font-weight="bold" text-anchor="middle">%s</text>`,
			width/2, 16.0, escapeXML(p.Title))
	}

	// Circle on the left half, legend on the right.
	legendW := math.Min(width*0.4, 180)
	plotW := width - legendW - 20
	plotH := height - titleH - 10
	cx := plotW/2 + 10
	cy := titleH + plotH/2
	radius := math.Min(plotW, plotH)*0.45 - 4
	if radius < 20 {
		radius = 20
	}

	// Slices — emit each as an SVG <path d="M cx,cy L x1,y1 A r,r 0 large,1 x2,y2 Z" />
	angle := -math.Pi / 2 // start at top
	for i, s := range p.Slices {
		if s.Value <= 0 {
			continue
		}
		frac := s.Value / total
		sweep := frac * 2 * math.Pi
		x1 := cx + radius*math.Cos(angle)
		y1 := cy + radius*math.Sin(angle)
		endAngle := angle + sweep
		x2 := cx + radius*math.Cos(endAngle)
		y2 := cy + radius*math.Sin(endAngle)
		large := 0
		if sweep > math.Pi {
			large = 1
		}
		color := pickColor(p.Palette, i)
		fmt.Fprintf(&buf,
			`<path d="M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 %d 1 %.2f,%.2f Z" fill="%s" stroke="white" stroke-width="1"/>`,
			cx, cy, x1, y1, radius, radius, large, x2, y2, color)
		angle = endAngle
	}

	// Legend on the right side: color box + label + percentage.
	legendX := width - legendW
	legendY := titleH + 16
	for i, s := range p.Slices {
		if s.Value <= 0 {
			continue
		}
		color := pickColor(p.Palette, i)
		pct := s.Value / total * 100
		fmt.Fprintf(&buf,
			`<rect x="%.2f" y="%.2f" width="10" height="10" fill="%s"/>`,
			legendX, legendY-9, color)
		fmt.Fprintf(&buf,
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="10">%s (%s%%)</text>`,
			legendX+15, legendY, escapeXML(s.Label), numberFormat(pct))
		legendY += 16
	}

	svgFooter(&buf)
	return buf.Bytes()
}
