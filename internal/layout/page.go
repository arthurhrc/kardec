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

	// FootnoteRefs collects the 1-based footnote numbers whose
	// in-text markers landed on this page, in encounter order. The
	// engine consults Document.Footnotes to find the matching body
	// text when emitting the per-page footnote chrome.
	FootnoteRefs []int

	// Width and Height are the page's actual dimensions in points
	// after orientation is applied. Landscape sections expose Width
	// > Height here even though Size still reports the un-rotated
	// PageSize. Renderers should consume these values when emitting
	// /MediaBox so multi-section documents with mixed orientations
	// land correct dimensions per page.
	Width, Height kardec.Length

	// BackgroundImage is the raw bytes of an image rendered behind
	// every PlacedItem on this page. Nil means "no background".
	// Format auto-detects through the same path Document.Image
	// uses (PNG / JPEG / SVG).
	BackgroundImage []byte
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

// BlockRole tags a PlacedItem with the source block's PDF/UA role
// so the writer can wrap each block in its own marked-content
// sequence and emit the matching StructElem (PDF 14.7.2 Table 333).
//
// Empty role means "default / paragraph" — the writer treats it as
// "P". Layout sets the role on every item it appends so render and
// the writer can group consecutive same-role items into structural
// blocks without re-walking the source tree.
type BlockRole string

const (
	BlockRoleP       BlockRole = "P"
	BlockRoleH1      BlockRole = "H1"
	BlockRoleH2      BlockRole = "H2"
	BlockRoleH3      BlockRole = "H3"
	BlockRoleH4      BlockRole = "H4"
	BlockRoleH5      BlockRole = "H5"
	BlockRoleH6      BlockRole = "H6"
	BlockRoleFigure  BlockRole = "Figure"
	BlockRoleCaption BlockRole = "Caption"
)

// PlacedItem is a positioned, fully styled fragment ready to be drawn.
// A given PlacedItem is either a text fragment (Text + Font + Size +
// Color set) or an image (Image non-nil); the renderer dispatches on
// whether Image is nil.
type PlacedItem struct {
	// Role tags this item with its source-block PDF/UA role so the
	// renderer can group consecutive same-role items into
	// structural blocks for tagging. Empty value is treated as "P".
	Role BlockRole

	// TableID is non-zero when this item belongs to a table cell.
	// Items with the same TableID share a /Table parent in the
	// PDF/UA structure tree. RowIdx and ColIdx are 0-based; cells
	// in the same row share TableID + RowIdx and aggregate under
	// one /TR; cells in the same column don't aggregate (PDF/UA
	// nests /TD under /TR, not /TR under columns). TableSeq lets
	// multiple tables on the same page disambiguate (1, 2, …).
	TableID  int
	RowIdx   int
	ColIdx   int

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
