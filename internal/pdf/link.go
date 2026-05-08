package pdf

import (
	"bytes"
	"fmt"
)

// emitLinkAnnots writes one annotation object per LinkAnnot on page
// and returns the slice of indirect-object IDs that the page's
// /Annots array should reference. Each annotation is a `/Subtype
// /Link` with a `/URI` action carrying the target URL.
//
// The Border [0 0 0] is set so PDF readers that render annotation
// borders by default do not draw a visible box around the link
// rectangle. The PDF spec defaults to a solid 1pt border which
// otherwise distracts from textual hyperlinks.
func emitLinkAnnots(ow *objectWriter, p Page) []int {
	if len(p.Links) == 0 {
		return nil
	}
	ids := make([]int, 0, len(p.Links))
	for _, ln := range p.Links {
		body := fmt.Sprintf(
			"<< /Type /Annot /Subtype /Link /Border [0 0 0] "+
				"/Rect [%.4f %.4f %.4f %.4f] "+
				"/A << /Type /Action /S /URI /URI %s >> >>",
			ln.X, ln.Y, ln.X+ln.W, ln.Y+ln.H,
			escapeLiteralString(ln.URI),
		)
		ids = append(ids, ow.allocAndWrite(body))
	}
	return ids
}

// renderAnnotsArray returns the value of the /Annots entry given the
// list of annotation IDs assembled for a page. Returns the empty
// string when no annotations exist so the page dictionary omits the
// key entirely.
func renderAnnotsArray(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString(" /Annots [")
	for i, id := range ids {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(ref(id))
	}
	buf.WriteString("]")
	return buf.String()
}
