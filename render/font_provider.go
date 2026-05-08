package render

import (
	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/typography"
)

// layoutFontProvider adapts a *typography.Registry to layout.FontProvider.
// The two interfaces differ in two places:
//
//   - typography.Resolve uses the typography.Weight enum and returns a
//     (Font, ok) pair where ok is false for unknown families. Layout's
//     contract requires a non-nil Font in every case (the engine has no
//     fallback path of its own).
//
//   - typography.Font exposes Measure/Ascent/Descent as separate methods,
//     while layout.Font collapses the trio into a single call returning
//     (width, ascent, descent).
//
// The adapter falls back to the registry's default font when the
// requested family is missing, and uses an inert sentinel font (zero
// metrics) only if the registry is empty.
//
// Each Resolve call returns a *measureAdapter that remembers the
// (family, weight, italic) tuple that produced it. The render package
// type-asserts back to *measureAdapter when building the PDF model so
// every PlacedItem maps to the correct embedded font ID.
type layoutFontProvider struct {
	registry *typography.Registry
}

func newLayoutFontProvider(reg *typography.Registry) layout.FontProvider {
	return &layoutFontProvider{registry: reg}
}

// Resolve implements layout.FontProvider. The boolean bold flag is
// mapped to typography.Weight.Bold; non-bold maps to Weight.Regular.
// Other CSS weights are not addressable from layout's narrower
// interface in v0.1; user code that wants Medium / SemiBold / Black
// can register custom families and select them through Style.Family.
func (p *layoutFontProvider) Resolve(family string, bold, italic bool) layout.Font {
	w := typography.Regular
	if bold {
		w = typography.Bold
	}
	face, ok := p.registry.Resolve(family, w, italic)
	resolvedFamily := family
	resolvedBold := bold
	resolvedItalic := italic
	if !ok {
		face = p.registry.Default()
		// When falling back to default, the embedding map keys against
		// the original (family, bold, italic) the layout engine asked
		// for. The PDF picks the default font ID instead, but the key
		// is still meaningful for diagnostics.
	}
	if face == nil {
		return nullFont{}
	}
	return &measureAdapter{
		inner:  face,
		family: resolvedFamily,
		bold:   resolvedBold,
		italic: resolvedItalic,
	}
}

// measureAdapter combines typography.Font's three separate metric
// methods into the single Measure call layout.Font expects, while
// remembering which (family, weight, italic) tuple produced it. The
// render package reads these fields when building the embedded-font
// table so each PlacedItem references the right font ID.
type measureAdapter struct {
	inner  typography.Font
	family string
	bold   bool
	italic bool
}

func (a *measureAdapter) Measure(text string, sizePt float64) (float64, float64, float64) {
	return a.inner.Measure(text, sizePt), a.inner.Ascent(sizePt), a.inner.Descent(sizePt)
}

// nullFont is the last-resort font returned when the registry has
// nothing. Every metric is zero so the layout engine still places
// glyphs without nil-checking; the result is degenerate output but
// does not panic.
type nullFont struct{}

func (nullFont) Measure(string, float64) (float64, float64, float64) { return 0, 0, 0 }
