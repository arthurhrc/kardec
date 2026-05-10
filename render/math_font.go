package render

import (
	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/pdf"
	"github.com/go-fonts/latin-modern/lmmath"
)

// appendMathFontIfUsed embeds Latin Modern Math (OTF/CFF) when at
// least one PlacedItem on the supplied pages flagged itself as
// math (PlacedItem.IsMath). Returns the font ID assigned to the
// math face, or -1 when no math content was emitted. The math
// font streams in via the same EmbeddedFont mechanism the body
// fonts use; the writer detects the OTTO scaler in parseTTF and
// routes to emitCFFFont (Type 0 + CIDFontType0 + FontFile3 with
// /Subtype /CIDFontType0C).
//
// v0.15 enables this — through v0.14 the function was a stub
// because the writer only knew TrueType. Now that CFF embedding
// landed in internal/pdf, the math font finally renders with its
// own glyphs (∑, ∫, √, …) instead of falling back to whichever
// glyph Liberation Sans happens to share the codepoint with.
func appendMathFontIfUsed(existing []pdf.EmbeddedFont, pages []layout.Page) (int, []pdf.EmbeddedFont) {
	used := false
	for _, p := range pages {
		for _, it := range p.Items {
			if it.IsMath {
				used = true
				break
			}
		}
		if used {
			break
		}
	}
	if !used {
		return -1, existing
	}
	if len(lmmath.TTF) == 0 {
		return -1, existing // build issue; bundled font missing — skip silently
	}
	idx := len(existing)
	return idx, append(existing, pdf.EmbeddedFont{
		Name:    "LatinModernMath",
		TTFData: lmmath.TTF,
	})
}
