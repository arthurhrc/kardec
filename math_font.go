package kardec

import "github.com/arthurhrc/kardec/internal/typography"

// MathFont returns the document's math-quality font face. The first call
// lazily loads Latin Modern Math (the GUST-OFL face bundled via
// `github.com/go-fonts/latin-modern/lmmath`); subsequent calls return the
// cached value so the layout engine can call GlyphFor / Measure /
// AscentDescent for every math atom without reparsing the font.
//
// If the load fails (e.g. the upstream module ships an empty byte slice
// or canvas refuses the OTF), the error is folded into the document's
// deferred-error chain — exposed via Err and surfaced by Render — and
// MathFont returns nil. This mirrors the rest of the builder API where
// failures accumulate without panicking.
//
// MathFont is the entry point downstream tracks (math parser, math
// layout) are expected to use to obtain the face attached to a Document;
// they should not call typography.LatinModernMath directly because that
// would skip the per-document caching.
func (d *Document) MathFont() typography.MathFont {
	if d.mathFont != nil {
		return d.mathFont
	}
	if d.err != nil {
		return nil
	}
	mf, err := typography.LatinModernMath()
	if err != nil {
		d.fail(err)
		return nil
	}
	d.mathFont = mf
	return mf
}
