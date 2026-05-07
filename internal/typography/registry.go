package typography

import (
	"fmt"

	"github.com/tdewolff/canvas"
)

// Registry stores the set of Font instances available to a Document. It is
// keyed by (family, weight, italic). Lookups fall back to the closest
// available weight when the requested face is missing, then to the
// registered default font as a last resort.
//
// A Registry is not safe for concurrent use by multiple goroutines, in line
// with the rest of the Kardec API.
type Registry struct {
	fonts        map[faceKey]Font
	defaultFont  Font
	familyOrder  []string         // insertion order, for deterministic iteration
	familySeen   map[string]bool  // dedup helper
	familyFaces  map[string][]faceKey
}

// faceKey is the composite map key for a font face.
type faceKey struct {
	family string
	weight Weight
	italic bool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		fonts:       make(map[faceKey]Font),
		familySeen:  make(map[string]bool),
		familyFaces: make(map[string][]faceKey),
	}
}

// Register parses ttfBytes and stores it under the given (family, weight,
// italic) tuple. The first font registered becomes the registry's default
// unless a later call to SetDefault overrides it. An error is returned if
// the byte slice is empty or canvas fails to parse it.
func (r *Registry) Register(family string, weight Weight, italic bool, ttfBytes []byte) error {
	if family == "" {
		return fmt.Errorf("typography: empty family name")
	}
	if len(ttfBytes) == 0 {
		return fmt.Errorf("typography: empty font bytes for %q", family)
	}
	style := canvasStyle(weight, italic)
	f, err := newCanvasFont(ttfBytes, style)
	if err != nil {
		return fmt.Errorf("typography: load %q %s italic=%t: %w", family, weight, italic, err)
	}
	k := faceKey{family: family, weight: weight, italic: italic}
	r.fonts[k] = f
	if !r.familySeen[family] {
		r.familySeen[family] = true
		r.familyOrder = append(r.familyOrder, family)
	}
	r.familyFaces[family] = append(r.familyFaces[family], k)
	if r.defaultFont == nil {
		r.defaultFont = f
	}
	return nil
}

// Resolve returns the font registered for (family, weight, italic). If that
// exact face is missing it tries weight fallbacks (closest CSS weight first,
// preserving italic), then the family's first registered face, then false.
func (r *Registry) Resolve(family string, weight Weight, italic bool) (Font, bool) {
	if f, ok := r.fonts[faceKey{family, weight, italic}]; ok {
		return f, true
	}
	// Same family, prefer matching italic, closest weight.
	if best, ok := r.bestWeightMatch(family, weight, italic); ok {
		return best, true
	}
	// Same family, opposite italic.
	if best, ok := r.bestWeightMatch(family, weight, !italic); ok {
		return best, true
	}
	return nil, false
}

// bestWeightMatch returns the registered face for family with the requested
// italic flag whose weight is numerically closest to weight.
func (r *Registry) bestWeightMatch(family string, weight Weight, italic bool) (Font, bool) {
	keys, ok := r.familyFaces[family]
	if !ok {
		return nil, false
	}
	want := weight.CSS()
	var best Font
	bestDelta := -1
	for _, k := range keys {
		if k.italic != italic {
			continue
		}
		delta := abs(k.weight.CSS() - want)
		if bestDelta == -1 || delta < bestDelta {
			best = r.fonts[k]
			bestDelta = delta
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

// Default returns the registry's default font. The first registered face is
// adopted automatically; SetDefault overrides it. Returns nil if no fonts
// have been registered.
func (r *Registry) Default() Font { return r.defaultFont }

// SetDefault selects the named (family, weight, italic) tuple as the default.
// Returns false if no such face is registered.
func (r *Registry) SetDefault(family string, weight Weight, italic bool) bool {
	f, ok := r.Resolve(family, weight, italic)
	if !ok {
		return false
	}
	r.defaultFont = f
	return true
}

// Families returns the registered family names in insertion order.
func (r *Registry) Families() []string {
	out := make([]string, len(r.familyOrder))
	copy(out, r.familyOrder)
	return out
}

// canvasStyle maps a (weight, italic) pair to the canvas package's
// FontStyle bitfield used by canvas.LoadFont.
func canvasStyle(w Weight, italic bool) canvas.FontStyle {
	var s canvas.FontStyle
	switch w {
	case Medium:
		s = canvas.FontMedium
	case SemiBold:
		s = canvas.FontSemiBold
	case Bold:
		s = canvas.FontBold
	case Black:
		s = canvas.FontBlack
	default:
		s = canvas.FontRegular
	}
	if italic {
		s |= canvas.FontItalic
	}
	return s
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
