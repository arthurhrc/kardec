package layout

import (
	"fmt"

	"github.com/arthurhrc/kardec"
)

// Engine is the layout entry point. It is a value type with no mutable
// configuration today; future versions will gain hooks for things like
// pluggable line breakers without breaking the call site.
type Engine struct{}

// NewEngine returns a ready-to-use layout engine. Provided as a function
// rather than relying on the zero value so future required configuration
// has a forward-compatible insertion point.
func NewEngine() Engine { return Engine{} }

// Layout walks the document and returns the laid-out pages. Errors are
// returned only for unrecoverable conditions (a block taller than a whole
// page with no way to split, or a nil FontProvider). Most input shapes
// produce a clean slice of pages.
func (e Engine) Layout(doc *kardec.Document, fonts FontProvider) ([]Page, error) {
	if doc == nil {
		return nil, fmt.Errorf("layout: nil document")
	}
	if fonts == nil {
		return nil, fmt.Errorf("layout: nil font provider")
	}
	if err := doc.Err(); err != nil {
		return nil, err
	}

	var pages []Page
	for secIdx, sec := range doc.Sections() {
		secPages, err := e.layoutSection(doc, sec, fonts)
		if err != nil {
			return nil, err
		}
		// Stamp section chrome (header / footer) on each page that the
		// section produced, with per-page tokens already substituted.
		// {{totalPages}} stays as a marker until the post-pass below.
		if len(sec.Header) > 0 || len(sec.Footer) > 0 {
			chromeStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleHeader))
			for i := range secPages {
				stampSectionChrome(&secPages[i], sec, chromeStyle, fonts, secIdx+1, len(pages)+i+1)
			}
		}
		// Stamp footnotes for any page whose body referenced one.
		// Resolved against doc.Footnotes by matching number.
		footnoteStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleFooter))
		for i := range secPages {
			if len(secPages[i].FootnoteRefs) == 0 {
				continue
			}
			refs := resolveFootnoteRefs(doc.Footnotes(), secPages[i].FootnoteRefs)
			stampFootnotes(&secPages[i], sec, refs, footnoteStyle, fonts)
		}
		pages = append(pages, secPages...)
	}
	// Final pass: now that we know the grand page count, replace any
	// {{totalPages}} placeholders left in header / footer items.
	SubstituteTotalPages(pages, len(pages))
	return pages, nil
}

// resolveFootnoteRefs maps the per-page footnote numbers back to the
// matching FootnoteRef structs from the document. Numbers that do
// not correspond to any registered footnote (a defensive guard, not
// expected during normal flow) are skipped silently.
func resolveFootnoteRefs(all []kardec.FootnoteRef, numbers []int) []kardec.FootnoteRef {
	out := make([]kardec.FootnoteRef, 0, len(numbers))
	for _, n := range numbers {
		for _, ref := range all {
			if ref.Number() == n {
				out = append(out, ref)
				break
			}
		}
	}
	return out
}

// stampFootnotes paints the footnote chrome onto a laid-out page,
// reusing the same pageCursor reconstruction trick as the header/
// footer post-pass.
func stampFootnotes(
	page *Page,
	sec *kardec.Section,
	refs []kardec.FootnoteRef,
	style blockStyle,
	fonts FontProvider,
) {
	cur := startPage(sec.Setup)
	cur.items = page.Items
	emitFootnotesForPage(cur, refs, style, fonts)
	page.Items = cur.items
}

// stampSectionChrome paints header / footer onto an already-laid-out
// page. The page's pageCursor is reconstructed from setup so chrome
// emission can reuse the shared shapeRuns/PlacedItem path.
func stampSectionChrome(
	page *Page,
	sec *kardec.Section,
	style blockStyle,
	fonts FontProvider,
	sectionNumber, pageNumber int,
) {
	cur := startPage(sec.Setup)
	cur.items = page.Items
	emitSectionChrome(cur, sec.Header, sec.Footer, style, fonts, pageNumber, sectionNumber)
	page.Items = cur.items
}

// pageCursor tracks the geometry of the page currently being filled.
type pageCursor struct {
	setup        kardec.PageSetup
	items        []PlacedItem
	headings     []HeadingMark
	anchors      []AnchorMark
	footnoteRefs []int // 1-based numbers, in encounter order, deduped
	x0, y0       float64 // top-left of the content area (after margins)
	x1, y1       float64 // bottom-right of the content area
	cursorY      float64 // current Y position, top-left origin
}

// startPage builds a fresh cursor positioned at the top of the content
// area for the section's page setup.
func startPage(setup kardec.PageSetup) *pageCursor {
	w, h := pageDimensions(setup)
	left := float64(setup.Margins.Left)
	top := float64(setup.Margins.Top)
	right := w - float64(setup.Margins.Right)
	bottom := h - float64(setup.Margins.Bottom)
	return &pageCursor{
		setup:   setup,
		x0:      left,
		y0:      top,
		x1:      right,
		y1:      bottom,
		cursorY: top,
	}
}

// pageDimensions returns the (width, height) of a page after applying the
// section's orientation. PageSize stores width as the short side; in
// landscape the engine swaps the two before computing the content box.
func pageDimensions(setup kardec.PageSetup) (float64, float64) {
	w := float64(setup.Size.Width)
	h := float64(setup.Size.Height)
	if setup.Orientation == kardec.Landscape {
		w, h = h, w
	}
	return w, h
}

// availableWidth returns the horizontal extent (in pt) that block content
// is allowed to fill.
func (c *pageCursor) availableWidth() float64 { return c.x1 - c.x0 }

// remainingHeight returns the vertical extent (in pt) still free below the
// cursor.
func (c *pageCursor) remainingHeight() float64 { return c.y1 - c.cursorY }

// finish converts the cursor into a Page value. Width/Height capture
// the orientation-applied dimensions so renderers don't need to look
// at Setup.Orientation themselves.
func (c *pageCursor) finish() Page {
	w, h := pageDimensions(c.setup)
	return Page{
		Size:         c.setup.Size,
		Items:        c.items,
		Headings:     c.headings,
		Anchors:      c.anchors,
		FootnoteRefs: c.footnoteRefs,
		Width:        kardec.Pt(w),
		Height:       kardec.Pt(h),
	}
}

// recordFootnoteRef registers a 1-based footnote number against the
// current page, deduplicating against earlier appearances on the
// same page (a footnote marker may shape into multiple tokens but
// should only show once at the bottom of the page).
func (c *pageCursor) recordFootnoteRef(n int) {
	if n <= 0 {
		return
	}
	for _, existing := range c.footnoteRefs {
		if existing == n {
			return
		}
	}
	c.footnoteRefs = append(c.footnoteRefs, n)
}

// headingTitle reconstructs the plain-text title of a Heading block by
// concatenating the texts of its runs. Lossy for runs that carry rich
// metadata, but the outline only needs a label.
func headingTitle(h kardec.Heading) string {
	var buf []byte
	for _, r := range h.Runs() {
		buf = append(buf, r.Text()...)
	}
	return string(buf)
}

// layoutSection lays out one section, possibly producing multiple pages.
// Style for each block is resolved through doc.ResolveBlockStyle, so a
// caller's DefineStyle / WithStyle / WithNamedStyle decisions actually
// shape the output.
//
// flush is implemented as an in-place mutation of cur (rather than a
// reassignment) so block-level placement code holding a *pageCursor can
// continue appending after a forced page break and still target the
// correct page. Reassignment alone would leave inner callers writing to
// the just-finished page through their stale pointer.
func (e Engine) layoutSection(doc *kardec.Document, sec *kardec.Section, fonts FontProvider) ([]Page, error) {
	var pages []Page
	cur := startPage(sec.Setup)
	flush := func() {
		pages = append(pages, cur.finish())
		*cur = *startPage(sec.Setup)
	}

	for _, b := range sec.Blocks {
		switch v := b.(type) {
		case kardec.Paragraph:
			style := styleFromKardec(doc.ResolveBlockStyle(v))
			if err := e.placeTextBlock(cur, flush, v.Runs(), style, fonts); err != nil {
				return nil, err
			}
		case kardec.Heading:
			style := styleFromKardec(doc.ResolveBlockStyle(v))
			cur.headings = append(cur.headings, HeadingMark{
				Level: v.Level(),
				Title: headingTitle(v),
				Y:     kardec.Pt(cur.cursorY),
			})
			if err := e.placeTextBlock(cur, flush, v.Runs(), style, fonts); err != nil {
				return nil, err
			}
		case kardec.Table:
			cellStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleTableCell))
			headerStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleTableHeader))
			e.placeTable(cur, flush, v, headerStyle, cellStyle, fonts)
		case kardec.Image:
			if err := e.placeImage(cur, flush, v); err != nil {
				return nil, err
			}
		case kardec.Math:
			mathStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleDefault))
			if err := e.placeMath(cur, flush, doc, v, mathStyle); err != nil {
				return nil, err
			}
		case kardec.List:
			itemStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleListItem))
			if err := e.placeList(cur, flush, v, itemStyle, fonts); err != nil {
				return nil, err
			}
		case kardec.Anchor:
			cur.anchors = append(cur.anchors, AnchorMark{
				Name: v.Name(),
				Y:    kardec.Pt(cur.cursorY),
			})
		case kardec.PageBreak:
			flush()
		case kardec.Spacer:
			advance := float64(v.Height)
			if cur.cursorY+advance > cur.y1 {
				flush()
				continue
			}
			cur.cursorY += advance
		default:
			// Unknown block kinds (Image, future v0.3): emit a stub
			// placeholder so the renderer track sees something
			// recognisable and reserves plausible vertical space.
			placeStub(cur, flush, b, fonts)
		}
	}
	pages = append(pages, cur.finish())
	return pages, nil
}

// placeStub emits a debug placeholder for not-yet-implemented block kinds.
// Tables reserve 60pt of vertical space; images 80pt. The fragment carries
// a "TODO ..." text marker so the renderer track can spot it during dry
// runs.
func placeStub(cur *pageCursor, flush func(), b kardec.Block, fonts FontProvider) {
	label, reserve := stubLabel(b)
	if cur.remainingHeight() < reserve {
		flush()
	}
	font := fonts.Resolve("", false, true)
	cur.items = append(cur.items, PlacedItem{
		X:     kardec.Pt(cur.x0),
		Y:     kardec.Pt(cur.cursorY),
		Text:  label,
		Font:  font,
		Size:  kardec.Pt(11),
		Color: kardec.ColorGray,
	})
	cur.cursorY += reserve
}

// stubLabel maps an unhandled block kind to a debug label and reserved
// height. Keeping the mapping centralised means new stubs are one line.
func stubLabel(b kardec.Block) (string, float64) {
	// Detect via fmt.Sprintf("%T", b) to avoid taking a hard dependency on
	// future block types here. The exact string is part of the layout
	// contract callers' tests use.
	switch fmt.Sprintf("%T", b) {
	case "kardec.Table":
		return "TODO table", 60
	case "kardec.Image":
		return "TODO image", 80
	default:
		return "TODO " + fmt.Sprintf("%T", b), 24
	}
}
