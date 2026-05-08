package typography

import (
	"fmt"
	"unicode/utf8"

	lmmath "github.com/go-fonts/latin-modern/lmmath"
	"github.com/tdewolff/canvas"
)

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

// latinModernMath wraps a *canvas.Font (loaded once, at construction
// time) and serves it through the MathFont interface. Latin Modern Math
// is the GUST-OFL math companion to Latin Modern Roman; it ships every
// glyph the symbol fallback table references plus the OpenType MATH
// table. The TTF/OTF bytes are obtained from the upstream
// `github.com/go-fonts/latin-modern/lmmath` Go module via its embedded
// `lmmath.TTF` byte slice — no disk I/O, no network at runtime.
type latinModernMath struct {
	font *canvas.Font
}

// LatinModernMath returns a MathFont backed by the bundled Latin
// Modern Math face. The OTF (~500 KB) is pulled from the
// `github.com/go-fonts/latin-modern/lmmath` Go module — it ships the
// font bytes via `lmmath.TTF` embedded with `//go:embed`, so the call
// is allocation-only and never touches the filesystem at runtime.
//
// The returned MathFont caches the parsed `*canvas.Font`; callers
// SHOULD reuse the value across atoms rather than calling
// LatinModernMath repeatedly. The package's public surface
// (`(*kardec.Document).MathFont`) memoises it for exactly that reason.
func LatinModernMath() (MathFont, error) {
	if len(lmmath.TTF) == 0 {
		return nil, fmt.Errorf("typography: lmmath.TTF is empty (build issue)")
	}
	f, err := canvas.LoadFont(lmmath.TTF, 0, canvas.FontRegular)
	if err != nil {
		return nil, fmt.Errorf("typography: load Latin Modern Math: %w", err)
	}
	return &latinModernMath{font: f}, nil
}

// face builds a transient *canvas.FontFace at the requested size in
// points, using the same mm-per-point conversion as canvasFont so the
// numbers line up with the rest of the typography package.
func (m *latinModernMath) face(sizePt float64) *canvas.FontFace {
	return m.font.Face(sizePt*mmPerPoint, canvas.Black)
}

// GlyphFor implements MathFont. The lookup order is:
//
//  1. If symbol starts with '\\', resolve via the LaTeX command table
//     (`lookupLatexSymbol`). Unknown commands return false.
//  2. Otherwise, decode the first rune of symbol and pass it through
//     as a literal glyph (ASCII digits / letters / operators).
//
// The MathGlyph carries the resolved Rune; the unexported metric
// fields are left zero — Measure / AscentDescent query the underlying
// *canvas.FontFace at call time and remain authoritative.
func (m *latinModernMath) GlyphFor(symbol string) (MathGlyph, bool) {
	if symbol == "" {
		return MathGlyph{}, false
	}
	if symbol[0] == '\\' {
		r, ok := lookupLatexSymbol(symbol)
		if !ok {
			return MathGlyph{}, false
		}
		return MathGlyph{Rune: r}, true
	}
	r, _ := utf8.DecodeRuneInString(symbol)
	if r == utf8.RuneError {
		return MathGlyph{}, false
	}
	return MathGlyph{Rune: r}, true
}

// Measure implements MathFont by delegating to FontFace.TextWidth on a
// single-rune string. The canvas library reports widths in millimetres;
// the result is converted back to points.
func (m *latinModernMath) Measure(g MathGlyph, sizePt float64) float64 {
	if g.Rune == 0 {
		return 0
	}
	return m.face(sizePt).TextWidth(string(g.Rune)) / mmPerPoint
}

// AscentDescent implements MathFont. Per-glyph ascent/descent is not
// trivially exposed by the canvas API, so this initial implementation
// returns the font-wide ascent and absolute descent at sizePt — enough
// for sub/superscript placement against a uniform baseline. Refinement
// (per-glyph bbox via the OpenType `glyf` / `CFF` tables) is a follow-up
// the layout track may request once super/subscript boxing is in place.
func (m *latinModernMath) AscentDescent(g MathGlyph, sizePt float64) (float64, float64) {
	mtx := m.face(sizePt).Metrics()
	asc := mtx.Ascent / mmPerPoint
	desc := mtx.Descent / mmPerPoint
	if desc < 0 {
		desc = -desc
	}
	return asc, desc
}
