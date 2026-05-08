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
// v0.2 scope: text cells with greedy line breaking. No borders, no shading,
// no alternating row colors — those are v0.3.
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

	// Emit each cell's lines at the appropriate column x. The column's
	// alignment is applied inside the column's width budget.
	x := cur.x0
	rowTop := cur.cursorY
	for i, ls := range cellLines {
		for li, ln := range ls {
			lineY := rowTop + float64(li)*style.lineHeight*style.sizePt
			emitTableCellLine(cur, ln, style, cols[i], x, colWidths[i], lineY)
		}
		x += colWidths[i]
	}
	cur.cursorY = rowTop + rowHeight
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

