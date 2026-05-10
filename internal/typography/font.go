package typography

import (
	"github.com/tdewolff/canvas"
)

// Font is the measurement-side abstraction the layout engine consumes. A Font
// is independent of the eventual render path; it exposes only the geometric
// queries needed to compute line breaking and page flow.
//
// All sizes are in PDF points (1/72 inch) to match the public Length type.
type Font interface {
	// Name returns the human-readable PostScript-style font name, primarily
	// for logging and debugging.
	Name() string

	// Measure returns the advance width of text rendered at the given size.
	// The returned value matches what the renderer will produce, modulo
	// kerning quirks at glyph boundaries.
	Measure(text string, sizePt float64) float64

	// Ascent returns the typographic ascent in points at the given size,
	// i.e. the height above the baseline used for line layout.
	Ascent(sizePt float64) float64

	// Descent returns the absolute value of the typographic descent in
	// points (always non-negative).
	Descent(sizePt float64) float64

	// LineHeight returns the natural line height in points at the given
	// size, which equals Ascent + Descent + LineGap from the font metrics.
	LineHeight(sizePt float64) float64
}

// canvasFont adapts a *canvas.Font (from github.com/tdewolff/canvas) to the
// Font interface above. It builds a transient FontFace per measurement call
// because FontFace caches size-dependent state.
//
// canvas.Face accepts size in PDF points; FontFace.TextWidth and the
// metrics it exposes (Ascent, Descent, LineHeight) come back in
// millimetres. The Measure / Ascent / Descent / LineHeight wrappers
// below divide by mmPerPoint so every value the rest of Kardec sees is
// in points — the unit the layout engine and PDF emit assume.
//
// (The size-input unit was wrong through v0.21: callers passed
// `sizePt*mmPerPoint` to Face under the assumption canvas wanted mm.
// That produced a font ~2.83x smaller than requested, advance widths
// followed suit, and word-positions in the PDF were too tight — every
// rendered document had visibly overlapping words. Fixed in v0.21.1.)
type canvasFont struct {
	f    *canvas.Font
	name string
}

// mmPerPoint is the conversion factor used to translate millimetre
// measurements out of the canvas library back into PDF points.
const mmPerPoint = 25.4 / 72.0

// newCanvasFont parses ttf bytes via canvas.LoadFont and returns a Font.
// The style argument lets callers preserve weight/italic information so
// canvas can render faux-bold or faux-italic when an actual face is missing.
func newCanvasFont(ttf []byte, style canvas.FontStyle) (Font, error) {
	f, err := canvas.LoadFont(ttf, 0, style)
	if err != nil {
		return nil, err
	}
	return &canvasFont{f: f, name: f.Name()}, nil
}

// face builds a FontFace at the requested size in points. canvas.Face
// takes the size argument in points directly — empirically, M's
// advance width on LiberationSans-Bold at Face(24) reports 7.05mm =
// 20pt, which matches the OpenType advance for a 24pt M in that face.
func (c *canvasFont) face(sizePt float64) *canvas.FontFace {
	return c.f.Face(sizePt, canvas.Black)
}

// Name implements Font.
func (c *canvasFont) Name() string { return c.name }

// Measure implements Font by delegating to FontFace.TextWidth, which returns
// millimeters; the result is converted back to points.
func (c *canvasFont) Measure(text string, sizePt float64) float64 {
	if text == "" {
		return 0
	}
	return c.face(sizePt).TextWidth(text) / mmPerPoint
}

// Ascent implements Font.
func (c *canvasFont) Ascent(sizePt float64) float64 {
	return c.face(sizePt).Metrics().Ascent / mmPerPoint
}

// Descent implements Font. Canvas reports descent as a negative value below
// the baseline; the Font interface promises an absolute distance.
func (c *canvasFont) Descent(sizePt float64) float64 {
	d := c.face(sizePt).Metrics().Descent / mmPerPoint
	if d < 0 {
		return -d
	}
	return d
}

// LineHeight implements Font.
func (c *canvasFont) LineHeight(sizePt float64) float64 {
	return c.face(sizePt).Metrics().LineHeight / mmPerPoint
}
