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
// The adapter falls back to the registry's default font when the requested
// family is missing, and uses an inert sentinel font (zero-width glyphs,
// zero metrics) only if the registry is empty — that shouldn't happen for
// documents created via kardec.New, since LoadBuiltinFonts runs in the
// constructor.
type layoutFontProvider struct {
	registry *typography.Registry
}

func newLayoutFontProvider(reg *typography.Registry) layout.FontProvider {
	return &layoutFontProvider{registry: reg}
}

// Resolve implements layout.FontProvider. The boolean bold flag is mapped to
// typography.Weight.Bold; non-bold maps to Weight.Regular. Other CSS weights
// are not addressable from layout's narrower interface in v0.1.
func (p *layoutFontProvider) Resolve(family string, bold, italic bool) layout.Font {
	w := typography.Regular
	if bold {
		w = typography.Bold
	}
	face, ok := p.registry.Resolve(family, w, italic)
	if !ok {
		face = p.registry.Default()
	}
	if face == nil {
		return nullFont{}
	}
	return &measureAdapter{inner: face}
}

// measureAdapter combines typography.Font's three separate metric methods
// into the single Measure call layout.Font expects.
type measureAdapter struct {
	inner typography.Font
}

func (a *measureAdapter) Measure(text string, sizePt float64) (float64, float64, float64) {
	return a.inner.Measure(text, sizePt), a.inner.Ascent(sizePt), a.inner.Descent(sizePt)
}

// nullFont is the last-resort font returned when the registry has nothing.
// Every metric is zero so the layout engine still places glyphs without
// nil-checking; the result is degenerate output but does not panic.
type nullFont struct{}

func (nullFont) Measure(string, float64) (float64, float64, float64) { return 0, 0, 0 }
