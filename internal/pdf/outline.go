package pdf

import (
	"bytes"
	"fmt"
)

// emitOutlines writes a PDF /Outlines tree from doc.Outlines and
// returns the indirect-object ID of the root outline dictionary, or
// 0 when the document carries no entries. Page references resolve
// through pageIDs (0-based positional mapping).
//
// Each outline node has /Title, /Parent, optional /First / /Last and
// /Next / /Prev sibling pointers, /Count for collapsed visibility,
// and /Dest pointing at a [pageRef /XYZ x y zoom] explicit destination.
// Acrobat reads the result as a sidebar of clickable bookmarks.
func emitOutlines(ow *objectWriter, entries []OutlineEntry, pageIDs []int) int {
	if len(entries) == 0 {
		return 0
	}
	rootID := ow.allocID()
	first, last, total := emitOutlineLevel(ow, rootID, entries, pageIDs)

	body := fmt.Sprintf(
		"<< /Type /Outlines /First %s /Last %s /Count %d >>",
		ref(first), ref(last), total,
	)
	ow.writeObject(rootID, body)
	return rootID
}

// emitOutlineLevel writes a peer chain of outline entries beneath the
// given parent. Returns the first child ID, the last child ID and the
// total descendant count to satisfy the parent's /Count.
func emitOutlineLevel(ow *objectWriter, parentID int, entries []OutlineEntry, pageIDs []int) (int, int, int) {
	ids := make([]int, len(entries))
	for i := range entries {
		ids[i] = ow.allocID()
	}
	totalDescendants := 0
	for i, e := range entries {
		var firstChild, lastChild, childCount int
		if len(e.Children) > 0 {
			firstChild, lastChild, childCount = emitOutlineLevel(ow, ids[i], e.Children, pageIDs)
		}
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "<< /Title %s /Parent %s",
			escapeLiteralString(e.Title), ref(parentID))
		if i > 0 {
			fmt.Fprintf(&buf, " /Prev %s", ref(ids[i-1]))
		}
		if i < len(entries)-1 {
			fmt.Fprintf(&buf, " /Next %s", ref(ids[i+1]))
		}
		if firstChild > 0 {
			fmt.Fprintf(&buf, " /First %s /Last %s /Count %d",
				ref(firstChild), ref(lastChild), childCount)
		}
		dest := outlineDestination(e, pageIDs)
		if dest != "" {
			fmt.Fprintf(&buf, " /Dest %s", dest)
		}
		buf.WriteString(" >>")
		ow.writeObject(ids[i], buf.String())
		totalDescendants += 1 + childCount
	}
	if len(ids) == 0 {
		return 0, 0, 0
	}
	return ids[0], ids[len(ids)-1], totalDescendants
}

// outlineDestination returns the /Dest value pointing at the entry's
// page and Y coordinate. Falls back to an empty string when the page
// index is out of range so the outline still renders as a clickable
// title without a destination.
func outlineDestination(e OutlineEntry, pageIDs []int) string {
	if e.PageIndex < 0 || e.PageIndex >= len(pageIDs) {
		return ""
	}
	return fmt.Sprintf("[%s /XYZ null %.4f null]", ref(pageIDs[e.PageIndex]), e.Y)
}
