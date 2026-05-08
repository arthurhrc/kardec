package render

import (
	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/pdf"
)

// appendMathFontIfUsed reserves a slot for the math face once the PDF
// writer learns to embed OpenType/CFF fonts (v0.3.x). Until then, math
// glyphs are routed to the default body font (Liberation Sans) so the
// pipeline still produces a valid PDF.
//
// Latin Modern Math is OpenType-with-CFF (OTTO sfnt header) and the
// current writer only supports TrueType outlines. Embedding it would
// require a CFF parser path. Tracked as a known limitation in
// CHANGELOG.md.
func appendMathFontIfUsed(existing []pdf.EmbeddedFont, pages []layout.Page) (int, []pdf.EmbeddedFont) {
	_ = pages // placeholder until CFF support lands
	return -1, existing
}
