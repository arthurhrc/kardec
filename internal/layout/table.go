package layout

import (
	"github.com/arthurhrc/kardec"
)

// placeTable lays out a kardec.Table on the page. Each cell is line-broken
// inside its column's width budget; the row height is the max breakLines
// height across cells. When a row does not fit the remaining page space the
// page is flushed and the row continues at the top of the next page; if
// RepeatHeader is set, row 0 is reprinted before resuming.
//
// Borders / shading / alternate-row coloring all opt-in via the matching
// kardec.Table flags. They are emitted as PlacedItem.Rect entries before
// the cell glyphs so the renderer paints them under the text.
func (e Engine) placeTable(cur *pageCursor, flush func(), tbl kardec.Table, headerStyle, cellStyle blockStyle, fonts FontProvider) {
	cols := tbl.Columns()
	if len(cols) == 0 {
		return
	}
	available := cur.availableWidth()
	colWidths := computeColumnWidths(cols, available)

	applySpaceBefore(cur, flush, cellStyle.spaceBeforePt)

	for rowIdx, row := range tbl.Rows() {
		style := cellStyle
		if rowIdx == 0 && tbl.RepeatHeader() {
			style = headerStyle
		}
		e.placeTableRow(cur, flush, cols, colWidths, row, style, fonts, tbl, rowIdx)
	}
	cur.cursorY += cellStyle.spaceAfterPt
}

// emitRowShading paints a background rectangle covering the row's
// horizontal extent. Called before cell text emission so the renderer
// paints the rectangle under the glyphs.
func emitRowShading(cur *pageCursor, x, y, width, height float64, color kardec.Color) {
	cur.items = append(cur.items, PlacedItem{
		X: kardec.Pt(x),
		Y: kardec.Pt(y),
		Rect: &PlacedRect{
			Width:     kardec.Pt(width),
			Thickness: kardec.Pt(height),
			Color:     color,
		},
	})
}

// emitHorizontalLine paints a 1-pt-tall rectangle simulating a thin
// horizontal rule. Width is the line's horizontal extent.
func emitHorizontalLine(cur *pageCursor, x, y, width float64) {
	cur.items = append(cur.items, PlacedItem{
		X: kardec.Pt(x),
		Y: kardec.Pt(y),
		Rect: &PlacedRect{
			Width:     kardec.Pt(width),
			Thickness: kardec.Pt(0.6),
			Color:     kardec.Color{R: 80, G: 80, B: 80},
		},
	})
}

// emitVerticalLine paints a 1-pt-wide rectangle simulating a thin
// vertical rule. Height is the line's vertical extent.
func emitVerticalLine(cur *pageCursor, x, y, height float64) {
	cur.items = append(cur.items, PlacedItem{
		X: kardec.Pt(x),
		Y: kardec.Pt(y),
		Rect: &PlacedRect{
			Width:     kardec.Pt(0.6),
			Thickness: kardec.Pt(height),
			Color:     kardec.Color{R: 80, G: 80, B: 80},
		},
	})
}

// placeTableRow emits the items for a single row, paginating if needed.
// On a forced flush, RepeatHeader causes row 0 to be reprinted at the top
// of the new page before this row's content.
func (e Engine) placeTableRow(
	cur *pageCursor,
	flush func(),
	cols []kardec.Column,
	colWidths []float64,
	row kardec.Row,
	style blockStyle,
	fonts FontProvider,
	tbl kardec.Table,
	rowIdx int,
) {
	cellLines := make([][]line, len(cols))
	maxLines := 0
	for i := range cols {
		cellRuns := cellRunsAt(row, i)
		tokens := shapeRuns(cellRuns, fonts, style, kardec.Pt(style.sizePt), style.color)
		ls := breakLines(tokens, colWidths[i])
		cellLines[i] = ls
		if len(ls) > maxLines {
			maxLines = len(ls)
		}
	}
	if maxLines == 0 {
		maxLines = 1
	}

	rowHeight := float64(maxLines) * style.lineHeight * style.sizePt
	if cur.remainingHeight() < rowHeight {
		flush()
		// On the new page, re-emit the header row so the user can read
		// the column meaning even on continuation pages.
		if tbl.RepeatHeader() && rowIdx > 0 && len(tbl.Rows()) > 0 {
			e.placeTableRow(cur, flush, cols, colWidths, tbl.Rows()[0], style, fonts, tbl, 0)
		}
	}

	rowTop := cur.cursorY
	totalWidth := 0.0
	for _, w := range colWidths {
		totalWidth += w
	}

	// Shading lands under the cell text — emit before glyphs.
	if rowIdx == 0 {
		if c, ok := tbl.HeaderShading(); ok {
			emitRowShading(cur, cur.x0, rowTop, totalWidth, rowHeight, c)
		}
	} else if rowIdx%2 == 1 {
		if c, ok := tbl.AlternateRowShading(); ok {
			emitRowShading(cur, cur.x0, rowTop, totalWidth, rowHeight, c)
		}
	}

	// Borders: emit horizontal rule above this row when borders include
	// horizontal lines, plus vertical rules when the full grid is
	// requested. The bottom border of every row is drawn as the top
	// border of the next; the table's outer bottom is added after the
	// final row in placeTable's caller via emitFinalBottomBorder, but
	// for v0.3 we emit a per-row bottom too so the table closes when
	// it ends mid-page without an extra block.
	bs := tbl.BorderStyle()
	if bs == kardec.BordersHorizontal || bs == kardec.BordersAll {
		emitHorizontalLine(cur, cur.x0, rowTop, totalWidth)
	}
	if bs == kardec.BordersAll {
		x := cur.x0
		emitVerticalLine(cur, x, rowTop, rowHeight)
		for _, w := range colWidths {
			x += w
			emitVerticalLine(cur, x, rowTop, rowHeight)
		}
	}

	// Emit each cell's lines at the appropriate column x. The column's
	// alignment is applied inside the column's width budget.
	x := cur.x0
	for i, ls := range cellLines {
		for li, ln := range ls {
			lineY := rowTop + float64(li)*style.lineHeight*style.sizePt
			emitTableCellLine(cur, ln, style, cols[i], x, colWidths[i], lineY)
		}
		x += colWidths[i]
	}

	cur.cursorY = rowTop + rowHeight

	// Outer bottom border of the table — only on the last row.
	if rowIdx == len(tbl.Rows())-1 && (bs == kardec.BordersHorizontal || bs == kardec.BordersAll) {
		emitHorizontalLine(cur, cur.x0, cur.cursorY, totalWidth)
	}
}

// emitTableCellLine emits one already-broken line for a single cell at
// (x, y). Alignment within the cell is honoured: AlignCenter and
// AlignRight shift x; AlignJustify falls back to left within a cell
// (justified table cells look awkward and rarely match user intent).
func emitTableCellLine(cur *pageCursor, ln line, style blockStyle, col kardec.Column, cellX, cellWidth, baselineY float64) {
	extra := cellWidth - ln.width
	if extra < 0 {
		extra = 0
	}
	startX := cellX
	switch col.Alignment {
	case kardec.AlignCenter:
		startX += extra / 2
	case kardec.AlignRight:
		startX += extra
	}
	x := startX
	for _, t := range ln.tokens {
		if t.isSpace {
			x += t.width
			continue
		}
		cur.items = append(cur.items, PlacedItem{
			X:     kardec.Pt(x),
			Y:     kardec.Pt(baselineY),
			Text:  t.text,
			Font:  t.font,
			Size:  kardec.Pt(t.sizePt),
			Color: style.color,
			Link:  t.link,
		})
		cur.recordFootnoteRef(t.footnoteRef)
		x += t.width
	}
}

// computeColumnWidths converts the column descriptors into concrete
// point widths summing to available. Fractional widths in (0, 1] take
// their share of available; fixed widths (>1) are taken as-is. Columns
// with zero or negative width split the remaining space equally.
func computeColumnWidths(cols []kardec.Column, available float64) []float64 {
	out := make([]float64, len(cols))
	used := 0.0
	unsized := 0
	for i, c := range cols {
		switch {
		case c.Width <= 0:
			unsized++
		case c.Width <= 1:
			out[i] = c.Width * available
			used += out[i]
		default:
			out[i] = c.Width
			used += out[i]
		}
	}
	if unsized > 0 {
		share := (available - used) / float64(unsized)
		if share < 0 {
			share = 0
		}
		for i, c := range cols {
			if c.Width <= 0 {
				out[i] = share
			}
		}
	}
	return out
}

// cellRunsAt returns the runs for the i-th column of row, padding with
// an empty slice when the row has fewer cells than the column count.
func cellRunsAt(row kardec.Row, i int) []kardec.Run {
	if i < 0 || i >= len(row.Cells) {
		return nil
	}
	return row.Cells[i].Runs
}

