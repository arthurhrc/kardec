package layout

import "github.com/arthurhrc/kardec"

// Page is a single laid-out output page: the originating page setup plus
// the ordered list of items the renderer will draw onto it.
//
// Items are stored in paint order (background to foreground) and use a
// top-left origin coordinate system; the PDF writer flips Y to PDF's
// bottom-left convention at emit time.
type Page struct {
	Size  kardec.PageSize
	Items []PlacedItem
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
