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
// v0.1 only carries text fragments; later versions add image and graphics
// payloads via additional fields and a Kind discriminator.
type PlacedItem struct {
	// X and Y are the top-left coordinates of the item, in PDF points,
	// relative to the page's top-left corner.
	X, Y kardec.Length

	// Text is the rendered string for this fragment. For non-text stub
	// items (the v0.1 placeholders for tables and images) the field
	// carries a "TODO ..." marker the renderer can recognise.
	Text string

	// Font is the resolved font handle used to draw Text. May be nil for
	// stub items that don't carry shaped glyphs yet.
	Font Font

	// Size is the point size at which Text should be rendered.
	Size kardec.Length

	// Color is the fill colour for the glyphs.
	Color kardec.Color
}
