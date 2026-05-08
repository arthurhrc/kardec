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
	Images []EmbeddedImage
}

// Page is one rendered page in PDF user space (1/72 inch). Width and Height
// are in points and become the page's /MediaBox. Items carries text show
// operations; Images carries raster image draws. Both lists are emitted in
// document order — text renders against the same coordinate system as
// images, both with origin at the bottom-left corner.
type Page struct {
	Width, Height float64
	Items         []TextItem
	Images        []ImageDraw
	Rects         []RectDraw
}

// ImageDraw positions one previously-embedded image on a page. X and Y are
// the bottom-left of the image in user space; W and H are its rendered
// width and height in points. ImageID indexes into Document.Images.
type ImageDraw struct {
	X, Y    float64
	W, H    float64
	ImageID int
}

// RectDraw paints a filled rectangle on a page. X and Y are the
// bottom-left of the rectangle in user space; W and H are its width and
// height in points. The rectangle is filled (not stroked) with Color in
// the device-RGB color space.
//
// Used by the math layout engine for fraction bars and square-root
// overlines, and available for any other primitive that needs a thin
// horizontal or vertical rule.
type RectDraw struct {
	X, Y  float64
	W, H  float64
	Color Color
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

// ImageEncoding selects how an EmbeddedImage is written into the PDF.
// JPEG payloads pass through the DCTDecode filter unchanged; raw RGB
// (24-bit per pixel, no alpha) is written through FlateDecode. PNG
// inputs are decoded into raw RGB by the renderer track and embedded
// using ImageRawRGB.
type ImageEncoding uint8

const (
	// ImageJPEG embeds the original JPEG bytes verbatim using
	// /Filter /DCTDecode. Bits-per-component is fixed at 8 and the
	// color space is always /DeviceRGB.
	ImageJPEG ImageEncoding = iota + 1
	// ImageRawRGB embeds packed 8-bit RGB triples (no alpha) using
	// /Filter /FlateDecode. Width and Height describe the pixel grid
	// in Data: len(Data) must equal Width*Height*3.
	ImageRawRGB
)

// EmbeddedImage carries one raster image's payload and pixel geometry.
// Multiple ImageDraw entries can reference the same EmbeddedImage so a
// document re-using a logo only embeds the bytes once.
type EmbeddedImage struct {
	WidthPx  int
	HeightPx int
	Encoding ImageEncoding
	// Data is either the original JPEG (ImageJPEG) or packed RGB
	// triples (ImageRawRGB). The writer applies the appropriate
	// /Filter when emitting the XObject.
	Data []byte
}
