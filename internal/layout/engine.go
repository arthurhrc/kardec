package layout

import (
	"fmt"

	"github.com/arthurhrc/kardec"
)

// Engine is the layout entry point. The Algorithm field selects the
// line-breaking strategy used by paragraph and table-cell layout —
// the zero value (LineBreakFirstFit) keeps the legacy first-fit
// behaviour with hyphenation; LineBreakOptimal switches to the
// Knuth-Plass DP optimum-fit breaker.
type Engine struct {
	Algorithm LineBreakAlgorithm
}

// LineBreakAlgorithm selects the paragraph line-breaking strategy.
// The values mirror kardec.LineBreakAlgorithm; the layout package
// keeps a private copy so internals do not transitively expose the
// public enum to other internal callers.
type LineBreakAlgorithm uint8

const (
	// LineBreakFirstFit is the v0.1-era greedy breaker plus the
	// v0.4 Knuth-Liang hyphenation fallback. Predictable, fast, and
	// the default.
	LineBreakFirstFit LineBreakAlgorithm = iota
	// LineBreakOptimal runs the Knuth-Plass DP optimum-fit
	// algorithm: lines are chosen to minimise summed badness² so
	// whitespace distributes more evenly and rivers in justified
	// paragraphs shrink. O(n²) in space count; sub-millisecond for
	// paragraph-sized inputs.
	LineBreakOptimal
)

// NewEngine returns a layout engine using the default first-fit
// breaker. Callers that want Knuth-Plass set Algorithm to
// LineBreakOptimal after construction (or rely on the package-level
// Layout function to read kardec.Document.LineBreakAlgorithm()).
func NewEngine() Engine { return Engine{Algorithm: LineBreakFirstFit} }

// breakLines dispatches paragraph line-break to the algorithm
// currently configured on the engine. Centralising the switch here
// keeps placeTextBlock and table layout unaware of the breaker
// choice.
func (e Engine) breakLines(tokens []token, available float64) []line {
	if e.Algorithm == LineBreakOptimal {
		return breakLinesOptimal(tokens, available)
	}
	return breakLines(tokens, available)
}

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
	// {{totalPages}} placeholders left in header / footer items, and
	// patch TOC page-number placeholders against the laid-out
	// heading positions.
	SubstituteTotalPages(pages, len(pages))
	patchTOCsAcrossSections(pages, doc)
	patchRefPagesAcrossSections(pages)
	return pages, nil
}

// patchTOCsAcrossSections walks every TOC block the document
// declared and patches the corresponding `{{tocpage:hN}}` markers
// on the laid-out pages. Multiple TOCs with different maxLevel
// values are tolerated — each receives its own pass.
func patchTOCsAcrossSections(pages []Page, doc *kardec.Document) {
	for _, sec := range doc.Sections() {
		for _, b := range sec.Blocks {
			toc, ok := b.(kardec.TableOfContents)
			if !ok {
				continue
			}
			patchTOCPlaceholders(pages, doc, toc.MaxLevel())
		}
	}
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
//
// Multi-column layouts treat (x0, y0)–(x1, y1) as the bounds of the
// *current* column rather than the page as a whole. columnIdx tracks
// which column is active; pageX0 / pageX1 retain the full content
// width so chrome (header/footer/section borders) and the post-pass
// can still address the page-level coordinate space.
type pageCursor struct {
	setup        kardec.PageSetup
	items        []PlacedItem
	headings     []HeadingMark
	anchors      []AnchorMark
	footnoteRefs []int   // 1-based numbers, in encounter order, deduped
	x0, y0       float64 // top-left of the active column
	x1, y1       float64 // bottom-right of the active column
	cursorY      float64 // current Y position, top-left origin

	// Multi-column state. columns == 1 keeps single-column geometry
	// (pageX0 == x0, pageX1 == x1).
	columns        int
	columnIdx      int     // 0-based index of the active column
	columnGap      float64 // horizontal gap between columns
	columnWidth    float64 // width of one column
	pageX0, pageX1 float64 // full content area horizontals (margins applied)
}

// startPage builds a fresh cursor positioned at the top of the first
// column for the section's page setup.
func startPage(setup kardec.PageSetup) *pageCursor {
	w, h := pageDimensions(setup)
	left := float64(setup.Margins.Left)
	top := float64(setup.Margins.Top)
	right := w - float64(setup.Margins.Right)
	bottom := h - float64(setup.Margins.Bottom)

	columns := setup.Columns
	if columns < 1 {
		columns = 1
	}
	columnGap := float64(setup.ColumnGap)
	if columns > 1 && columnGap <= 0 {
		columnGap = defaultColumnGapPt
	}
	contentWidth := right - left
	columnWidth := contentWidth
	if columns > 1 {
		columnWidth = (contentWidth - float64(columns-1)*columnGap) / float64(columns)
	}

	return &pageCursor{
		setup:       setup,
		x0:          left,
		y0:          top,
		x1:          left + columnWidth,
		y1:          bottom,
		cursorY:     top,
		columns:     columns,
		columnIdx:   0,
		columnGap:   columnGap,
		columnWidth: columnWidth,
		pageX0:      left,
		pageX1:      right,
	}
}

// defaultColumnGapPt is the gutter between columns when ColumnGap is
// zero. ~12pt is the typographic norm for body-size text.
const defaultColumnGapPt = 12.0

// advanceColumn moves the cursor to the next column on the current
// page, resetting cursorY to the top. Returns true when the move
// succeeded; false when the cursor was already in the last column,
// signalling the caller to flush and start a new page instead.
func (c *pageCursor) advanceColumn() bool {
	if c.columnIdx+1 >= c.columns {
		return false
	}
	c.columnIdx++
	c.x0 = c.pageX0 + float64(c.columnIdx)*(c.columnWidth+c.columnGap)
	c.x1 = c.x0 + c.columnWidth
	c.cursorY = c.y0
	return true
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
	pageFlush := func() {
		pages = append(pages, cur.finish())
		*cur = *startPage(sec.Setup)
	}
	// Standard flush: advance to the next column when the current
	// column ran out, otherwise finish the page and start fresh.
	// Single-column layouts collapse to the page-flush behaviour
	// because cur.advanceColumn always returns false when columns ==
	// 1.
	flush := func() {
		if cur.advanceColumn() {
			return
		}
		pageFlush()
	}

	for _, b := range sec.Blocks {
		if err := e.placeBlock(cur, flush, pageFlush, doc, sec, b, fonts, &pages); err != nil {
			return nil, err
		}
	}
	pages = append(pages, cur.finish())
	return pages, nil
}

// placeBlock dispatches a single block onto the current page. Extracted
// from layoutSection so KeepTogether can reuse the same per-block
// placement path for its inner children.
//
// flush is the column-aware overflow flush: in a multi-column layout
// it advances to the next column before falling back to a page break.
// pageFlush forces a new page regardless of column position; it is
// used for explicit PageBreak blocks so the user-facing semantics
// stay "next page, not next column". The pages pointer lets the
// keep-together rollback path snapshot and restore the page count
// without exposing the slice header to other call sites.
func (e Engine) placeBlock(
	cur *pageCursor,
	flush func(),
	pageFlush func(),
	doc *kardec.Document,
	sec *kardec.Section,
	b kardec.Block,
	fonts FontProvider,
	pages *[]Page,
) error {
	switch v := b.(type) {
	case kardec.Paragraph:
		style := styleFromKardec(doc.ResolveBlockStyle(v))
		return e.placeTextBlock(cur, flush, v.Runs(), style, fonts)
	case kardec.Heading:
		style := styleFromKardec(doc.ResolveBlockStyle(v))
		title := headingTitle(v)
		// Auto-anchor each heading so TOC entries can hyperlink
		// to the body. Slug derived from the heading title; the
		// TOC placement constructs the same slug to wire Link
		// targets onto the title tokens.
		cur.anchors = append(cur.anchors, AnchorMark{
			Name: tocHeadingAnchor(title),
			Y:    kardec.Pt(cur.cursorY),
		})
		cur.headings = append(cur.headings, HeadingMark{
			Level: v.Level(),
			Title: title,
			Y:     kardec.Pt(cur.cursorY),
		})
		return e.placeTextBlock(cur, flush, v.Runs(), style, fonts)
	case kardec.Table:
		cellStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleTableCell))
		headerStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleTableHeader))
		e.placeTable(cur, flush, v, headerStyle, cellStyle, fonts)
		return nil
	case kardec.Image:
		return e.placeImage(cur, flush, v)
	case kardec.Math:
		mathStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleDefault))
		return e.placeMath(cur, flush, doc, v, mathStyle)
	case kardec.List:
		itemStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleListItem))
		return e.placeList(cur, flush, v, itemStyle, fonts)
	case kardec.Anchor:
		cur.anchors = append(cur.anchors, AnchorMark{
			Name: v.Name(),
			Y:    kardec.Pt(cur.cursorY),
		})
		return nil
	case kardec.TableOfContents:
		tocStyle := styleFromKardec(doc.ResolveStyle(kardec.StyleDefault))
		e.placeTOC(cur, flush, doc, v, tocStyle, fonts)
		return nil
	case kardec.HorizontalRule:
		e.placeHorizontalRule(cur, flush, v)
		return nil
	case kardec.Leader:
		style := styleFromKardec(doc.ResolveStyle(kardec.StyleDefault))
		e.placeLeader(cur, flush, v, style, fonts)
		return nil
	case kardec.KeepTogether:
		return e.placeKeepTogether(cur, flush, pageFlush, doc, sec, v, fonts, pages)
	case kardec.PageBreak:
		pageFlush()
		return nil
	case kardec.Spacer:
		advance := float64(v.Height)
		if cur.cursorY+advance > cur.y1 {
			flush()
			return nil
		}
		cur.cursorY += advance
		return nil
	default:
		placeStub(cur, flush, b, fonts)
		return nil
	}
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
