package pdf

// Document is the writer's local input model. It is deliberately decoupled
// from the public kardec.Document so the layout track can populate it
// without circular imports — the renderer never sees blocks or styles, only
// positioned glyph runs.
type Document struct {
	Title        string
	Author       string
	Subject      string
	Keywords     string
	Pages        []Page
	Fonts        []EmbeddedFont
	Images       []EmbeddedImage
	Outlines     []OutlineEntry
	Destinations []NamedDestination
	// PDFA, when true, makes the writer emit XMP metadata declaring
	// PDF/A-2b conformance, attach a /Metadata catalog entry, and
	// write a stable /ID array in the trailer.
	PDFA bool
	// ICCProfile is the embedded sRGB ICC profile bytes used as the
	// destination color space for the OutputIntent. When non-nil and
	// PDFA is true, the writer emits a /GTS_PDFA1 OutputIntent
	// referencing this profile — the missing piece between PDF/A
	// "lite" and strict veraPDF-passing PDF/A-2b. nil leaves the
	// writer in lite mode (markers without OutputIntent).
	ICCProfile []byte
	// ICCProfileN is the number of color components in ICCProfile
	// (3 for sRGB, 4 for CMYK). Must match the profile bytes;
	// the writer does not validate.
	ICCProfileN int

	// Encryption, when non-nil, opts the document into the
	// Standard Security Handler V=4 R=4 (AES-128). All streams
	// and string literals in the produced PDF are encrypted with
	// per-object keys derived from the supplied passwords, and
	// the trailer carries the /Encrypt indirect-object reference.
	Encryption *Encryption
}

// Encryption configures the PDF Standard Security Handler. UserPwd
// is what a reader prompts for to OPEN the document; OwnerPwd
// authorises bypassing the permissions. Empty UserPwd produces a
// "permissions-only" PDF anyone can open but only the owner
// password can re-edit / re-print.
//
// Permissions is the OR of the AllowXxx bits. Spec §7.6.3.2 Table
// 22 — Kardec follows the bit numbering exactly.
type Encryption struct {
	UserPwd     string
	OwnerPwd    string
	Permissions int32
}

// Permission bits for the standard security handler /P entry.
// Kardec mirrors the spec layout (§7.6.3.2 Table 22) with
// readable Go names. Bits 1-2, 7-8, and 13-32 are reserved-must-
// be-1 per the spec; pdfBaseP folds them in by default so
// hand-constructed P values stay valid.
const (
	pdfBaseP            = -3904 // bits 7, 8, 13-32 set; reserved
	PermissionPrint     = 1 << 2
	PermissionModify    = 1 << 3
	PermissionCopy      = 1 << 4
	PermissionAnnotate  = 1 << 5
	PermissionFillForms = 1 << 8
	PermissionAccess    = 1 << 9
	PermissionAssemble  = 1 << 10
	PermissionPrintHigh = 1 << 11
)

// NamedDestination is one entry in the PDF's /Dests dictionary. Name
// is the lookup key (referenced from /GoTo actions); PageIndex is
// 0-based into Document.Pages; Y is the destination's Y coordinate
// in PDF user space (bottom-left origin).
type NamedDestination struct {
	Name      string
	PageIndex int
	Y         float64
}

// OutlineEntry is one bookmark in the PDF's `/Outlines` tree.
// Children are indented one level under their parent in the reader's
// sidebar. PageIndex is 0-based into Document.Pages; Y is the
// destination's Y coordinate in PDF user space (bottom-left origin).
//
// Most documents derive these from heading blocks: an H1 becomes a
// top-level entry; an H2 nests as a child; deeper headings continue
// the chain. The renderer track shapes the tree before populating
// Document.Outlines.
type OutlineEntry struct {
	Title     string
	PageIndex int
	Y         float64
	Children  []OutlineEntry
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
	Links         []LinkAnnot
}

// LinkAnnot is one rectangular hyperlink area on a page. The
// rectangle is in PDF user-space (bottom-left origin). When URI is
// non-empty the link emits a /URI external action; when DestName is
// non-empty it emits a /GoTo /D action targeting a named destination
// from Document.Destinations. URI takes precedence if both are set —
// callers should set exactly one.
type LinkAnnot struct {
	X, Y     float64 // bottom-left of the clickable rectangle
	W, H     float64 // width and height in points
	URI      string  // external URL; mutually exclusive with DestName
	DestName string  // named destination key; resolves through /Dests
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

// EmbeddedFont carries a font's display name and its raw TrueType
// bytes. KeepGIDs (when non-nil) opts the writer into glyf-table
// subsetting: every glyph not in the set is zeroed out before
// embedding so the FontFile2 stream compresses to a fraction of
// the full TTF. nil keeps the legacy "embed the entire font"
// behaviour, which is still the safer choice for documents that
// register a font without a known glyph-usage profile.
type EmbeddedFont struct {
	Name     string
	TTFData  []byte
	KeepGIDs map[uint16]bool
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
