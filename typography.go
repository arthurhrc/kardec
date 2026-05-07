package kardec

import (
	"github.com/arthurhrc/kardec/internal/typography"
)

// RegisterFont parses ttfBytes and registers the resulting face under the
// given (family, weight, italic) tuple in the Document's font registry. The
// first registered face becomes the default unless overridden later.
//
// The weight argument uses the public Weight type whose values map 1:1 onto
// the internal typography.Weight (Regular through Black). Errors are
// captured in the Document's deferred error chain and surfaced by Err.
func (d *Document) RegisterFont(family string, weight Weight, italic bool, ttfBytes []byte) *Document {
	if d.err != nil {
		return d
	}
	if d.fonts == nil {
		d.fonts = typography.NewRegistry()
	}
	if err := d.fonts.Register(family, toInternalWeight(weight), italic, ttfBytes); err != nil {
		return d.fail(err)
	}
	return d
}

// MeasureText returns the rendered advance width of text in the requested
// face. The boolean is false when the family is not registered (or when no
// fallback face matches), letting callers distinguish a missing font from a
// zero-width string.
//
// This entry point is provided for the layout track; user code rarely calls
// it directly.
func (d *Document) MeasureText(text string, family string, size Length, weight Weight, italic bool) (Length, bool) {
	if d.fonts == nil {
		return 0, false
	}
	face, ok := d.fonts.Resolve(family, toInternalWeight(weight), italic)
	if !ok {
		return 0, false
	}
	return Length(face.Measure(text, size.Points())), true
}

// FontRegistry exposes the underlying typography registry. It exists so the
// public render package (github.com/arthurhrc/kardec/render) can construct a
// FontProvider for the layout engine without going through every measurement
// via MeasureText. End-user code should prefer MeasureText / RegisterFont.
//
// The returned *typography.Registry is the document's own backing store;
// callers must not register fonts directly through it (use RegisterFont so
// errors flow through the deferred-error chain).
func (d *Document) FontRegistry() *typography.Registry {
	if d.fonts == nil {
		d.fonts = typography.NewRegistry()
	}
	return d.fonts
}

// toInternalWeight maps the public Weight constants to their typography
// counterparts. The two enums share the same ordinal numbering by design,
// but the conversion is explicit so the public package may add Weight
// values that lack a one-to-one internal mapping later.
func toInternalWeight(w Weight) typography.Weight {
	switch w {
	case WeightMedium:
		return typography.Medium
	case WeightSemiBold:
		return typography.SemiBold
	case WeightBold:
		return typography.Bold
	case WeightBlack:
		return typography.Black
	default:
		return typography.Regular
	}
}
