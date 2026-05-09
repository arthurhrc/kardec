package layout

import "github.com/arthurhrc/kardec"

// Page is a single laid-out output page: the originating page setup plus
// the ordered list of items the renderer will draw onto it.
//
// Items are stored in paint order (background to foreground) and use a
// top-left origin coordinate system; the PDF writer flips Y to PDF's
// bottom-left convention at emit time.
type Page struct {
	Size     kardec.PageSize
	Items    []PlacedItem
	Headings []HeadingMark
	Anchors  []AnchorMark
}

// AnchorMark records a named anchor at a specific Y on a page.
// Render uses these to populate the PDF's named-destinations table so
// `Link("text", "#name")` runs resolve to /GoTo /D actions targeting
// the right page and Y.
type AnchorMark struct {
	Name string
	Y    kardec.Length
}

// HeadingMark is a per-page record of a heading block that started on
// this page. Render uses the slice to build the PDF outline (sidebar
// bookmarks). Y is the heading's baseline in top-left-origin
// coordinates; render flips to PDF user space at emit time.
type HeadingMark struct {
	Level int
	Title string
	Y     kardec.Length
}

// PlacedItem is a positioned, fully styled fragment ready to be drawn.
// A given PlacedItem is either a text fragment (Text + Font + Size +
// Color set) or an image (Image non-nil); the renderer dispatches on
// whether Image is nil.
type PlacedItem struct {
	// X and Y are the top-left coordinates of the item, in PDF points,
	// relative to the page's top-left corner.
	X, Y kardec.Length

	// Text is the rendered string for this fragment. Empty when the
	// item is an image.
	Text string

	// Font is the resolved font handle used to draw Text. Nil when the
	// item is an image or a stub.
	Font Font

	// Size is the point size at which Text should be rendered.
	Size kardec.Length

	// Color is the fill colour for the glyphs.
	Color kardec.Color

	// Image is non-nil when this PlacedItem represents a raster image
	// drawn at (X, Y) with the dimensions stored on PlacedImage.
	Image *PlacedImage

	// IsMath signals that Text was emitted by the math layout engine
	// and must be rendered with the math face (Latin Modern Math)
	// rather than the body font referenced by Font. The renderer
	// detects the flag and routes the glyph to the math-font ID.
	IsMath bool

	// Link is non-empty when this PlacedItem participates in a
	// hyperlink. The renderer collects every link-bearing item with
	// the same Link target into one rectangular annotation per page.
	Link string

	// Rect is non-nil when this PlacedItem represents a filled
	// rectangle drawn at (X, Y) with the dimensions and color stored
	// on PlacedRect. Used by the math layout engine for fraction
	// bars and square-root overlines.
	Rect *PlacedRect
}

// PlacedRect carries the geometry and fill color for a rectangle
// emitted into a Page's items. The renderer translates each PlacedRect
// into a pdf.RectDraw at write time.
type PlacedRect struct {
	Width     kardec.Length
	Thickness kardec.Length
	Color     kardec.Color
}

// PlacedImage carries the raster payload and final geometry the
// renderer needs to embed an image. The Data slice is shared with the
// originating Image block and must not be mutated.
type PlacedImage struct {
	Data   []byte
	Format kardec.ImageFormat
	Width  kardec.Length
	Height kardec.Length
}
