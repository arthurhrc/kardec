package pdf

// Document is the writer's local input model. It is deliberately decoupled
// from the public kardec.Document so the layout track can populate it
// without circular imports — the renderer never sees blocks or styles, only
// positioned glyph runs.
type Document struct {
	Title  string
	Author string
	Pages  []Page
	Fonts  []EmbeddedFont
}

// Page is one rendered page in PDF user space (1/72 inch). Width and Height
// are in points and become the page's /MediaBox.
type Page struct {
	Width, Height float64
	Items         []TextItem
}

// TextItem is one show-text operation. X and Y use PDF user space with
// origin at the bottom-left corner of the page; the layout engine — which
// natively works in top-left-origin space — converts via:
//
//	pdfY = pageHeight - layoutY - itemAscent
//
// FontID indexes into Document.Fonts. FontSize is in points.
type TextItem struct {
	X, Y     float64
	Text     string
	FontID   int
	FontSize float64
	Color    Color
}

// Color is the writer's local sRGB triple. The public package mirrors
// kardec.Color one-to-one onto this type at the conversion boundary in
// Document.toPDFModel; keeping a separate type in internal/pdf prevents
// the import cycle that would arise from depending on the root package.
type Color struct {
	R, G, B uint8
}

// EmbeddedFont carries a font's display name and its raw TrueType bytes.
// The writer embeds the full TTF as a FontFile2 stream — no subsetting in
// v0.1. The size penalty (often 100–500 KB per font) is accepted for
// simplicity; subsetting is planned for v0.2 once a glyph-coverage pass
// runs over each Document.
type EmbeddedFont struct {
	Name    string
	TTFData []byte
}
