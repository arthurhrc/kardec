// Package layout turns a Kardec Document tree into positioned glyph runs
// ready for the PDF writer track to emit. It owns line breaking, page
// breaking and block-level placement; it does not own font shaping (that
// belongs to the typography track) nor PDF byte emission (the renderer
// track).
//
// Coordinates are reported with a top-left origin in Length units (PDF
// points). The renderer track is responsible for flipping to PDF's
// bottom-left origin at write time.
package layout

// FontProvider is the minimum surface the layout engine needs from the
// typography subsystem. It is defined here so the layout package compiles
// against a stub; the typography track (feat/typography) supplies the
// production implementation that wraps a real OpenType shaper.
type FontProvider interface {
	// Resolve returns a Font for the given family with the requested
	// weight/style flags. Implementations are expected to fall back to a
	// sensible default rather than return nil when an unknown family is
	// requested, so the engine can keep laying out without nil checks.
	Resolve(family string, bold, italic bool) Font
}

// Font is the minimum text-measurement surface needed to break lines and
// place glyphs. The typography track wraps a real shaper behind this
// interface; the layout engine only ever asks for advance widths and the
// vertical extents above and below the baseline.
type Font interface {
	// Measure returns the advance width and vertical extents of text when
	// rendered at the requested point size. All return values are in PDF
	// points; ascent is the rise above the baseline, descent the drop
	// below (reported as a positive number).
	Measure(text string, sizePt float64) (widthPt float64, ascentPt float64, descentPt float64)
}
