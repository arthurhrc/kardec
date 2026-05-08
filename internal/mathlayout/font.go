package mathlayout

// Font is the minimum measurement surface this package needs from the
// math typography track. Real implementations wrap a math-aware OpenType
// font and resolve glyphs through MATH-table metrics; the layout engine
// only ever asks for advance widths and ascent/descent, plus a way to
// turn a logical symbol into a Glyph it can later place.
//
// Defining the interface locally lets layout compile in isolation and
// lets tests inject a stub Font without depending on a real font file.
type Font interface {
	// GlyphFor resolves a logical symbol (a single rune as a string,
	// such as "+", "x" or "∑") to a Glyph the renderer will be able to
	// emit. The boolean is false when the font cannot supply the
	// symbol; layout treats a missing glyph as a zero-width gap so a
	// missing radical or fraction bar does not crash a document.
	GlyphFor(symbol string) (Glyph, bool)

	// Measure returns the advance width of the given glyph at the
	// requested point size, in PDF points. The result is always
	// non-negative.
	Measure(g Glyph, sizePt float64) float64

	// AscentDescent returns the rise above the baseline and drop below
	// the baseline of the given glyph at the requested point size, both
	// in PDF points. Descent is reported as a positive number.
	AscentDescent(g Glyph, sizePt float64) (float64, float64)
}

// Glyph is the layout engine's handle on a single resolved glyph. It
// carries the rune the renderer will emit; concrete Font implementations
// may store private metric fields alongside Rune (font index, MATH-table
// offsets, ...) — the layout engine never inspects them.
//
// Keeping the type a plain struct rather than an opaque handle means the
// renderer can read PlacedGlyph.Rune directly without round-tripping
// through Font, which simplifies the writer track.
type Glyph struct {
	// Rune is the Unicode code point the renderer should draw. A zero
	// value indicates "no glyph"; layout only emits PlacedGlyphs whose
	// Rune is non-zero.
	Rune rune
}
