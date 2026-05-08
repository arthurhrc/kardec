package typography

// MathFont is the measurement-side abstraction for a math-quality font face.
// A layout engine uses it to resolve LaTeX command names (or literal runes)
// into glyphs and to size sub/superscripts and choose vertical placement.
//
// The contract is intentionally narrow — only the queries needed to box a
// math atom — and parallels the plain-text Font interface in this package.
// All sizes are in PDF points (1/72 inch) to match the public Length type.
//
// Implementations MUST make GlyphFor / Measure / AscentDescent cheap to
// call (constant-time lookups, no per-call font reload), because the layout
// engine invokes them once per AST atom.
type MathFont interface {
	// GlyphFor returns the glyph metrics for a LaTeX command name
	// (e.g., "\\alpha", "\\sum") or a literal rune (e.g., 'a', '+').
	// The string form accepts either:
	//
	//   - a backslash-prefixed command name like "\\alpha" or "\\sum";
	//   - a single-rune string for ASCII / Unicode pass-through.
	//
	// The boolean is false when the symbol cannot be resolved.
	GlyphFor(symbol string) (MathGlyph, bool)

	// Measure returns the advance width of a single glyph at sizePt.
	Measure(g MathGlyph, sizePt float64) float64

	// AscentDescent returns the vertical extents above and below the
	// baseline at sizePt for a glyph. Descent is reported as a positive
	// number (i.e. an absolute distance below the baseline), in line
	// with Font.Descent.
	AscentDescent(g MathGlyph, sizePt float64) (float64, float64)
}

// MathGlyph carries the minimum information the layout engine needs to
// position a math atom. The exported Rune lets renderers turn the glyph
// back into text for the PDF content stream; remaining metric fields are
// kept unexported so future additions (kerning pairs, italic correction,
// stretchy-glyph variants) can land without breaking call sites.
//
// MathGlyph is a value type; copying is cheap and explicit aliasing of
// the underlying font is unnecessary.
type MathGlyph struct {
	// Rune is the Unicode codepoint that represents this glyph in
	// the math font. Layout writes it into the PDF text run.
	Rune rune

	// advance1000 is the horizontal advance in 1/1000 em (font design
	// units). Multiplying by sizePt/1000 yields the advance in points.
	advance1000 int16

	// ascent1000 is the height above the baseline in 1/1000 em.
	ascent1000 int16

	// descent1000 is the absolute distance below the baseline in
	// 1/1000 em (i.e. always non-negative for typical glyphs).
	descent1000 int16
}
