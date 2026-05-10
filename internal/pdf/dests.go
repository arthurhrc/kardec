package pdf

import (
	"bytes"
	"fmt"
	"strings"
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
			escapePDFName(d.Name),
			ref(pageIDs[d.PageIndex]),
			d.Y,
		)
	}
	buf.WriteString(" >>")
	ow.writeObject(id, buf.String())
	return id
}

// escapePDFName turns an arbitrary string into a valid PDF Name
// object (PDF 7.3.5). Names start with `/` and may contain any
// regular character; reserved or non-printable bytes must be
// hex-escaped as `#XX`. Empty name returns `/` (technically an
// empty name, accepted by readers).
//
// Reserved chars per spec: whitespace, delimiters `()<>[]{}/%`,
// the `#` escape introducer itself, and bytes < 0x21 or > 0x7E.
func escapePDFName(s string) string {
	var b strings.Builder
	b.WriteByte('/')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x21 || c > 0x7E ||
			c == '(' || c == ')' || c == '<' || c == '>' ||
			c == '[' || c == ']' || c == '{' || c == '}' ||
			c == '/' || c == '%' || c == '#' {
			fmt.Fprintf(&b, "#%02X", c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
