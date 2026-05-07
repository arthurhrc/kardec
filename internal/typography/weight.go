// Package typography is the internal font registry, embedded font bundle,
// and OpenType measurement layer that backs the public Kardec DSL. The root
// package re-exports a minimal surface; layout and renderer subpackages
// import this package directly.
package typography

// Weight selects a font face within a family. The five-step subset of the
// OpenType usWeightClass scale is sufficient for the document-style PDFs
// Kardec targets; intermediate weights can be added later without breaking
// the existing names.
//
// The internal type is the canonical declaration; the public Style.Weight
// type aliases or wraps this so DSL callers do not need to import the
// internal package.
type Weight uint8

// Weight constants follow common usWeightClass intervals. The zero value is
// Regular so a freshly-allocated Style defaults to a sensible weight.
const (
	Regular  Weight = iota // 400 in OpenType terms
	Medium                 // 500
	SemiBold               // 600
	Bold                   // 700
	Black                  // 900
)

// CSS returns the CSS / OpenType numeric weight (e.g. 400 for Regular). It
// matches the convention used by github.com/tdewolff/canvas.FontStyle.CSS.
func (w Weight) CSS() int {
	switch w {
	case Medium:
		return 500
	case SemiBold:
		return 600
	case Bold:
		return 700
	case Black:
		return 900
	default:
		return 400
	}
}

// String returns a human-readable name for the weight, matching the constant
// identifier ("Regular", "Bold", etc.).
func (w Weight) String() string {
	switch w {
	case Medium:
		return "Medium"
	case SemiBold:
		return "SemiBold"
	case Bold:
		return "Bold"
	case Black:
		return "Black"
	default:
		return "Regular"
	}
}
