package layout

import "github.com/arthurhrc/kardec"

// placeKeepTogether places every block in the group while preserving the
// "all on the same page" guarantee. Strategy:
//
//  1. Snapshot the current cursor + the outer pages slice length.
//  2. Wrap flush so we can detect any flush that fires while inner
//     blocks are being placed.
//  3. Place every inner block in order with the wrapped flush.
//  4. If the wrapped flush never fired, the group fit on the current
//     page and we are done.
//  5. If it fired, undo: restore the cursor + pages; flush the original
//     so the group can start fresh on the next page; re-run placement
//     with the real flush so an oversized group can overflow naturally.
//
// Step (5) prevents an infinite loop when the group is taller than a
// full page: the second pass uses the unwrapped flush, so the group
// degrades to ordinary multi-page placement instead of looping.
func (e Engine) placeKeepTogether(
	cur *pageCursor,
	flush func(),
	pageFlush func(),
	doc *kardec.Document,
	sec *kardec.Section,
	group kardec.KeepTogether,
	fonts FontProvider,
	pages *[]Page,
) error {
	blocks := group.Blocks()
	if len(blocks) == 0 {
		return nil
	}

	snap := snapshotCursor(cur, len(*pages))
	flushed := false
	wrapped := func() {
		flushed = true
		flush()
	}

	for _, b := range blocks {
		if err := e.placeBlock(cur, wrapped, pageFlush, doc, sec, b, fonts, pages); err != nil {
			return err
		}
	}
	if !flushed {
		return nil
	}

	// Group spilled across a flush. Roll back, flush the original
	// column/page, and re-place using the real flush so an oversized
	// group can overflow naturally (no second wrapping = no second
	// rollback).
	restoreCursor(cur, snap)
	*pages = (*pages)[:snap.pageCount]
	flush()
	for _, b := range blocks {
		if err := e.placeBlock(cur, flush, pageFlush, doc, sec, b, fonts, pages); err != nil {
			return err
		}
	}
	return nil
}

// cursorSnapshot captures enough of the cursor's state to restore it
// after a speculative placement. Slice headers are recorded by length;
// the underlying arrays are not copied because restore() truncates
// rather than rewrites — appended-but-discarded items become
// unreachable garbage.
type cursorSnapshot struct {
	items     int
	headings  int
	anchors   int
	footnotes int
	cursorY   float64
	pageCount int
}

func snapshotCursor(cur *pageCursor, pageCount int) cursorSnapshot {
	return cursorSnapshot{
		items:     len(cur.items),
		headings:  len(cur.headings),
		anchors:   len(cur.anchors),
		footnotes: len(cur.footnoteRefs),
		cursorY:   cur.cursorY,
		pageCount: pageCount,
	}
}

func restoreCursor(cur *pageCursor, snap cursorSnapshot) {
	cur.items = cur.items[:snap.items]
	cur.headings = cur.headings[:snap.headings]
	cur.anchors = cur.anchors[:snap.anchors]
	cur.footnoteRefs = cur.footnoteRefs[:snap.footnotes]
	cur.cursorY = snap.cursorY
}
