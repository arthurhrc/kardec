// Package svg converts a small subset of SVG into the PDF graphics
// operators a Form XObject content stream consumes. The result is
// embedded as a /Subtype /Form XObject so a page can place the
// drawing exactly like a raster image, but vector-precise at any
// scale.
//
// Supported in v0.19.0:
//
//   - <svg width="..." height="..." viewBox="x y w h"> root
//   - <rect x y width height fill stroke stroke-width />
//   - <circle cx cy r fill stroke stroke-width />
//   - <ellipse cx cy rx ry fill stroke stroke-width />
//   - <line x1 y1 x2 y2 stroke stroke-width />
//   - <polyline points="..." />
//   - <polygon points="..." />
//   - <path d="..." /> with absolute M, L, H, V, C, Q, Z commands
//   - <g> nesting (style inheritance); transforms are ignored
//   - fill / stroke colors as #rrggbb or a small named-color palette;
//     "none" disables that paint
//   - opacity / fill-opacity / stroke-opacity (0..1)
//
// Out of scope (silently ignored, return as much as we can):
// gradients, patterns, filters, masks, clipPath, defs/use,
// <text>, <image>, transforms, dashed strokes, line caps, line
// joins. The renderer best-effort-renders what it understands and
// drops the rest.
//
// Coordinate handling: SVG's Y axis grows downward; PDF's grows
// upward. Convert handles the flip by inverting Y after loading
// the viewBox.
package svg

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// Convert parses src as SVG and returns:
//
//	bbWidth, bbHeight  — the natural canvas size in points (used as
//	                     the Form XObject /BBox upper-right corner)
//	contentStream     — the PDF operator bytes for the Form XObject
//
// Errors come from malformed XML or unparseable numeric attributes;
// unknown elements/attributes are dropped silently.
func Convert(src []byte) (bbWidth, bbHeight float64, contentStream []byte, err error) {
	root, err := parseRoot(src)
	if err != nil {
		return 0, 0, nil, err
	}

	w, h := root.canvasSize()
	vbX, vbY, vbW, vbH := root.viewBox(w, h)
	if vbW <= 0 || vbH <= 0 {
		return 0, 0, nil, fmt.Errorf("svg: viewBox/dimensions resolved to zero (%v %v)", vbW, vbH)
	}

	var b strings.Builder
	// Map SVG → Form XObject coordinate space:
	//   1. translate by -vbX, -vbY so the viewBox origin becomes (0,0)
	//   2. flip Y by scaling y by -1
	//   3. translate up by vbH so y=0 lands at the bottom of the box
	//   4. scale x/y so vbW maps to w and vbH maps to h
	// Folded into a single CTM via "a b c d e f cm" (PDF 8.3.4).
	scaleX := w / vbW
	scaleY := h / vbH
	// Final matrix: x' = scaleX * (x - vbX); y' = h - scaleY * (y - vbY)
	// In matrix form the cm operator takes [scaleX 0 0 -scaleY tx ty]
	tx := -scaleX * vbX
	ty := h + scaleY*vbY
	fmt.Fprintf(&b, "%.6f 0 0 %.6f %.6f %.6f cm\n", scaleX, -scaleY, tx, ty)

	state := paintState{
		fill:         color{r: 0, g: 0, b: 0, set: true},
		stroke:       color{set: false},
		strokeWidth:  1,
		fillOpacity:  1,
		strokeOpacity: 1,
		opacity:      1,
	}
	if err := emitChildren(&b, root.children, state); err != nil {
		return 0, 0, nil, err
	}
	return w, h, []byte(b.String()), nil
}

// emitChildren walks the parsed element tree, applying SVG paint
// inheritance: each element's own attributes layer on top of the
// inherited state before being applied to the underlying drawing
// primitive.
func emitChildren(b *strings.Builder, kids []element, parent paintState) error {
	for _, el := range kids {
		s := parent.merge(el.style)
		switch el.tag {
		case "g":
			if err := emitChildren(b, el.children, s); err != nil {
				return err
			}
		case "rect":
			emitRect(b, el, s)
		case "circle":
			emitCircle(b, el, s)
		case "ellipse":
			emitEllipse(b, el, s)
		case "line":
			emitLine(b, el, s)
		case "polyline":
			emitPolyline(b, el, s, false)
		case "polygon":
			emitPolyline(b, el, s, true)
		case "path":
			emitPath(b, el, s)
		}
	}
	return nil
}

func emitRect(b *strings.Builder, el element, s paintState) {
	x := getNum(el.attrs, "x")
	y := getNum(el.attrs, "y")
	w := getNum(el.attrs, "width")
	h := getNum(el.attrs, "height")
	if w <= 0 || h <= 0 {
		return
	}
	emitPaintPrologue(b, s)
	fmt.Fprintf(b, "%.6f %.6f %.6f %.6f re\n", x, y, w, h)
	emitPaintEpilogue(b, s)
}

func emitCircle(b *strings.Builder, el element, s paintState) {
	cx := getNum(el.attrs, "cx")
	cy := getNum(el.attrs, "cy")
	r := getNum(el.attrs, "r")
	if r <= 0 {
		return
	}
	emitPaintPrologue(b, s)
	emitEllipsePath(b, cx, cy, r, r)
	emitPaintEpilogue(b, s)
}

func emitEllipse(b *strings.Builder, el element, s paintState) {
	cx := getNum(el.attrs, "cx")
	cy := getNum(el.attrs, "cy")
	rx := getNum(el.attrs, "rx")
	ry := getNum(el.attrs, "ry")
	if rx <= 0 || ry <= 0 {
		return
	}
	emitPaintPrologue(b, s)
	emitEllipsePath(b, cx, cy, rx, ry)
	emitPaintEpilogue(b, s)
}

// emitEllipsePath approximates an ellipse with four cubic Beziers.
// The 0.552284... constant is the canonical Bezier control-point
// offset that matches a quarter circle to ≤ 0.05% radial error.
func emitEllipsePath(b *strings.Builder, cx, cy, rx, ry float64) {
	const kappa = 0.5522847498
	ox := rx * kappa
	oy := ry * kappa
	fmt.Fprintf(b, "%.6f %.6f m\n", cx-rx, cy)
	fmt.Fprintf(b, "%.6f %.6f %.6f %.6f %.6f %.6f c\n", cx-rx, cy-oy, cx-ox, cy-ry, cx, cy-ry)
	fmt.Fprintf(b, "%.6f %.6f %.6f %.6f %.6f %.6f c\n", cx+ox, cy-ry, cx+rx, cy-oy, cx+rx, cy)
	fmt.Fprintf(b, "%.6f %.6f %.6f %.6f %.6f %.6f c\n", cx+rx, cy+oy, cx+ox, cy+ry, cx, cy+ry)
	fmt.Fprintf(b, "%.6f %.6f %.6f %.6f %.6f %.6f c\n", cx-ox, cy+ry, cx-rx, cy+oy, cx-rx, cy)
}

func emitLine(b *strings.Builder, el element, s paintState) {
	x1 := getNum(el.attrs, "x1")
	y1 := getNum(el.attrs, "y1")
	x2 := getNum(el.attrs, "x2")
	y2 := getNum(el.attrs, "y2")
	// <line> is stroke-only by SVG default — disable any inherited
	// fill so a "fill=red" on a parent <g> doesn't paint a phantom
	// triangle behind the stroke.
	s.fill.set = false
	emitPaintPrologue(b, s)
	fmt.Fprintf(b, "%.6f %.6f m %.6f %.6f l\n", x1, y1, x2, y2)
	emitPaintEpilogue(b, s)
}

func emitPolyline(b *strings.Builder, el element, s paintState, closed bool) {
	pts := parsePoints(el.attrs["points"])
	if len(pts) < 2 {
		return
	}
	if !closed {
		// As with <line>, polyline is stroke-only by default.
		s.fill.set = false
	}
	emitPaintPrologue(b, s)
	fmt.Fprintf(b, "%.6f %.6f m\n", pts[0].x, pts[0].y)
	for i := 1; i < len(pts); i++ {
		fmt.Fprintf(b, "%.6f %.6f l\n", pts[i].x, pts[i].y)
	}
	if closed {
		b.WriteString("h\n")
	}
	emitPaintEpilogue(b, s)
}

func emitPath(b *strings.Builder, el element, s paintState) {
	cmds, err := parsePath(el.attrs["d"])
	if err != nil || len(cmds) == 0 {
		return
	}
	emitPaintPrologue(b, s)
	var cx, cy float64
	for _, c := range cmds {
		switch c.op {
		case 'M':
			cx, cy = c.coords[0], c.coords[1]
			fmt.Fprintf(b, "%.6f %.6f m\n", cx, cy)
		case 'L':
			cx, cy = c.coords[0], c.coords[1]
			fmt.Fprintf(b, "%.6f %.6f l\n", cx, cy)
		case 'H':
			cx = c.coords[0]
			fmt.Fprintf(b, "%.6f %.6f l\n", cx, cy)
		case 'V':
			cy = c.coords[0]
			fmt.Fprintf(b, "%.6f %.6f l\n", cx, cy)
		case 'C':
			fmt.Fprintf(b, "%.6f %.6f %.6f %.6f %.6f %.6f c\n",
				c.coords[0], c.coords[1], c.coords[2], c.coords[3], c.coords[4], c.coords[5])
			cx, cy = c.coords[4], c.coords[5]
		case 'Q':
			// PDF has no native quadratic Bezier; promote to cubic by
			// the standard control-point reparameterisation:
			//   c1 = q0 + 2/3 * (qc - q0), c2 = q1 + 2/3 * (qc - q1)
			qx, qy := c.coords[0], c.coords[1]
			ex, ey := c.coords[2], c.coords[3]
			c1x := cx + (2.0/3.0)*(qx-cx)
			c1y := cy + (2.0/3.0)*(qy-cy)
			c2x := ex + (2.0/3.0)*(qx-ex)
			c2y := ey + (2.0/3.0)*(qy-ey)
			fmt.Fprintf(b, "%.6f %.6f %.6f %.6f %.6f %.6f c\n", c1x, c1y, c2x, c2y, ex, ey)
			cx, cy = ex, ey
		case 'Z':
			b.WriteString("h\n")
		}
	}
	emitPaintEpilogue(b, s)
}

// emitPaintPrologue writes the colour-setup operators (rg, RG, w)
// matching the resolved paint state. Called immediately before the
// path-construction operators so the path inherits the right
// graphics-state values.
func emitPaintPrologue(b *strings.Builder, s paintState) {
	b.WriteString("q\n")
	if s.fill.set {
		fmt.Fprintf(b, "%.6f %.6f %.6f rg\n",
			float64(s.fill.r)/255.0, float64(s.fill.g)/255.0, float64(s.fill.b)/255.0)
	}
	if s.stroke.set {
		fmt.Fprintf(b, "%.6f %.6f %.6f RG\n",
			float64(s.stroke.r)/255.0, float64(s.stroke.g)/255.0, float64(s.stroke.b)/255.0)
		fmt.Fprintf(b, "%.6f w\n", s.strokeWidth)
	}
}

// emitPaintEpilogue writes the path-painting operator (B / B* / f /
// f* / S / n) followed by Q. The choice depends on which paints are
// active.
func emitPaintEpilogue(b *strings.Builder, s paintState) {
	switch {
	case s.fill.set && s.stroke.set:
		b.WriteString("B\n")
	case s.fill.set:
		b.WriteString("f\n")
	case s.stroke.set:
		b.WriteString("S\n")
	default:
		b.WriteString("n\n")
	}
	b.WriteString("Q\n")
}

// element is a single SVG node after parsing. We retain only the
// fields the converter consumes; everything else from the encoding/
// xml token is discarded.
type element struct {
	tag      string
	attrs    map[string]string
	children []element
	style    paintState
}

// rootElement carries the root <svg> attributes alongside its
// children; canvasSize / viewBox derive the drawing canvas.
type rootElement struct {
	attrs    map[string]string
	children []element
}

func (r rootElement) canvasSize() (w, h float64) {
	w = parseLength(r.attrs["width"], 0)
	h = parseLength(r.attrs["height"], 0)
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 100
	}
	return w, h
}

func (r rootElement) viewBox(w, h float64) (x, y, vw, vh float64) {
	vb := strings.TrimSpace(r.attrs["viewBox"])
	if vb == "" {
		return 0, 0, w, h
	}
	parts := splitNums(vb)
	if len(parts) != 4 {
		return 0, 0, w, h
	}
	return parts[0], parts[1], parts[2], parts[3]
}

// parseRoot reads the SVG byte stream into the in-memory element
// tree the converter consumes. Uses encoding/xml to handle
// namespaces and entity decoding correctly.
func parseRoot(src []byte) (rootElement, error) {
	dec := xml.NewDecoder(strings.NewReader(string(src)))
	for {
		tok, err := dec.Token()
		if err != nil {
			return rootElement{}, fmt.Errorf("svg: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "svg" {
			return rootElement{}, fmt.Errorf("svg: root element is %q, expected <svg>", start.Name.Local)
		}
		root := rootElement{attrs: attrsToMap(start.Attr)}
		root.children = parseChildren(dec)
		return root, nil
	}
}

func parseChildren(dec *xml.Decoder) []element {
	var out []element
	for {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		switch t := tok.(type) {
		case xml.StartElement:
			el := element{
				tag:   t.Name.Local,
				attrs: attrsToMap(t.Attr),
			}
			el.style = parseStyle(el.attrs)
			el.children = parseChildren(dec)
			out = append(out, el)
		case xml.EndElement:
			return out
		}
	}
}

func attrsToMap(attrs []xml.Attr) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, a := range attrs {
		out[a.Name.Local] = a.Value
	}
	return out
}

func getNum(m map[string]string, key string) float64 {
	return parseLength(m[key], 0)
}

// parseLength accepts SVG length strings: a number with optional
// "px" / "pt" / "%" suffix. Other suffixes are ignored (treated as
// raw numeric). Parse failures fall through to fallback.
func parseLength(s string, fallback float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	// Strip a trailing unit. "px" and "pt" map 1:1 onto our
	// downstream coordinate system because the converter rescales
	// via viewBox at the end.
	for _, unit := range []string{"px", "pt", "em", "ex", "%", "mm", "cm", "in"} {
		if strings.HasSuffix(s, unit) {
			s = strings.TrimSuffix(s, unit)
			break
		}
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return fallback
	}
	return v
}

type point struct{ x, y float64 }

func parsePoints(s string) []point {
	nums := splitNums(s)
	if len(nums)%2 != 0 {
		nums = nums[:len(nums)-1]
	}
	out := make([]point, 0, len(nums)/2)
	for i := 0; i+1 < len(nums); i += 2 {
		out = append(out, point{x: nums[i], y: nums[i+1]})
	}
	return out
}

// splitNums tokenises a whitespace/comma-separated number list,
// handling SVG's habit of running numbers together (e.g.
// "10,20-30.5 40").
func splitNums(s string) []float64 {
	var out []float64
	var cur strings.Builder
	flush := func() {
		t := strings.TrimSpace(cur.String())
		cur.Reset()
		if t == "" {
			return
		}
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			out = append(out, v)
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ',' || c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
		case c == '-' && cur.Len() > 0 && cur.String()[cur.Len()-1] != 'e' && cur.String()[cur.Len()-1] != 'E':
			// Negative sign starts a new number unless it's an
			// exponent ("1e-3").
			flush()
			cur.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}
