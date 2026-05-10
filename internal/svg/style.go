package svg

import (
	"strconv"
	"strings"
)

// paintState carries the inherited graphics-state values an SVG
// element resolves to. The converter merges the parent's state
// with the element's own attributes before drawing — same flow
// SVG renderers use.
type paintState struct {
	fill          color
	stroke        color
	strokeWidth   float64
	fillOpacity   float64
	strokeOpacity float64
	opacity       float64
}

// color carries an RGB triple plus two flags. set distinguishes
// "paint visible" from "paint=none / transparent / unrecognised";
// changed records whether the SVG attribute was explicitly present
// on the originating element so paintState.merge can tell "child
// said nothing, keep parent's color" from "child said paint=none,
// disable inherited color".
type color struct {
	r, g, b uint8
	set     bool
	changed bool
}

// merge layers the child's attribute-derived style over the parent
// state. SVG's "currentColor" / inherited values are not modeled —
// values explicitly missing from the child keep the parent's.
func (p paintState) merge(child paintState) paintState {
	out := p
	if child.fill.changed {
		out.fill = child.fill
	}
	if child.stroke.changed {
		out.stroke = child.stroke
	}
	if child.strokeWidth > 0 {
		out.strokeWidth = child.strokeWidth
	}
	if child.fillOpacity > 0 {
		out.fillOpacity = child.fillOpacity
	}
	if child.strokeOpacity > 0 {
		out.strokeOpacity = child.strokeOpacity
	}
	if child.opacity > 0 {
		out.opacity = child.opacity
	}
	return out
}

func parseStyle(attrs map[string]string) paintState {
	s := paintState{}
	if v, ok := attrs["fill"]; ok {
		s.fill = parseColor(v)
		s.fill.changed = true
	}
	if v, ok := attrs["stroke"]; ok {
		s.stroke = parseColor(v)
		s.stroke.changed = true
	}
	if v, ok := attrs["stroke-width"]; ok {
		if w, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && w > 0 {
			s.strokeWidth = w
		}
	}
	if v, ok := attrs["opacity"]; ok {
		if o, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && o > 0 {
			s.opacity = o
		}
	}
	if v, ok := attrs["fill-opacity"]; ok {
		if o, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && o > 0 {
			s.fillOpacity = o
		}
	}
	if v, ok := attrs["stroke-opacity"]; ok {
		if o, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && o > 0 {
			s.strokeOpacity = o
		}
	}
	return s
}

// parseColor resolves a small but practically useful subset of CSS
// color values: #rgb, #rrggbb, "none", "transparent", and a
// canonical named-color palette covering the colors most icons
// use. Unknown inputs return the zero color (set=false, behaves
// like "none").
func parseColor(s string) color {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "none" || s == "transparent" {
		return color{set: false}
	}
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		switch len(hex) {
		case 3:
			r := hexNib(hex[0])
			g := hexNib(hex[1])
			b := hexNib(hex[2])
			return color{r: r*16 + r, g: g*16 + g, b: b*16 + b, set: true}
		case 6:
			r := hexNib(hex[0])*16 + hexNib(hex[1])
			g := hexNib(hex[2])*16 + hexNib(hex[3])
			b := hexNib(hex[4])*16 + hexNib(hex[5])
			return color{r: r, g: g, b: b, set: true}
		}
	}
	if c, ok := namedColors[s]; ok {
		return c
	}
	return color{set: false}
}

func hexNib(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// namedColors covers the SVG named colors most likely to appear
// in human-authored icons. The full CSS3 list (~140 names) is
// out of scope for v0.19.0 — adding remaining names is a
// mechanical follow-up if real-world content needs them.
var namedColors = map[string]color{
	"black":   {r: 0x00, g: 0x00, b: 0x00, set: true},
	"white":   {r: 0xFF, g: 0xFF, b: 0xFF, set: true},
	"red":     {r: 0xFF, g: 0x00, b: 0x00, set: true},
	"green":   {r: 0x00, g: 0x80, b: 0x00, set: true},
	"blue":    {r: 0x00, g: 0x00, b: 0xFF, set: true},
	"yellow":  {r: 0xFF, g: 0xFF, b: 0x00, set: true},
	"cyan":    {r: 0x00, g: 0xFF, b: 0xFF, set: true},
	"magenta": {r: 0xFF, g: 0x00, b: 0xFF, set: true},
	"gray":    {r: 0x80, g: 0x80, b: 0x80, set: true},
	"grey":    {r: 0x80, g: 0x80, b: 0x80, set: true},
	"silver":  {r: 0xC0, g: 0xC0, b: 0xC0, set: true},
	"orange":  {r: 0xFF, g: 0xA5, b: 0x00, set: true},
	"purple":  {r: 0x80, g: 0x00, b: 0x80, set: true},
	"navy":    {r: 0x00, g: 0x00, b: 0x80, set: true},
	"teal":    {r: 0x00, g: 0x80, b: 0x80, set: true},
	"maroon":  {r: 0x80, g: 0x00, b: 0x00, set: true},
	"olive":   {r: 0x80, g: 0x80, b: 0x00, set: true},
	"lime":    {r: 0x00, g: 0xFF, b: 0x00, set: true},
}
