package pdf

import (
	"bytes"
	"fmt"
)

// emitDestinations writes the /Dests dictionary that maps named
// destinations to explicit destination arrays, and returns its
// indirect-object ID. Returns 0 when the document carries no
// destinations.
//
// Each /Dests entry maps a name (the lookup key from /GoTo /D
// actions) to an array `[pageRef /XYZ x y zoom]`. The writer uses
// /XYZ with `null` x and zoom so the reader keeps the current
// horizontal scroll and zoom level — only Y changes.
func emitDestinations(ow *objectWriter, dests []NamedDestination, pageIDs []int) int {
	if len(dests) == 0 {
		return 0
	}
	id := ow.allocID()
	var buf bytes.Buffer
	buf.WriteString("<<")
	for _, d := range dests {
		if d.PageIndex < 0 || d.PageIndex >= len(pageIDs) {
			continue
		}
		fmt.Fprintf(&buf, " %s [%s /XYZ null %.4f null]",
			escapeLiteralString(d.Name),
			ref(pageIDs[d.PageIndex]),
			d.Y,
		)
	}
	buf.WriteString(" >>")
	ow.writeObject(id, buf.String())
	return id
}
