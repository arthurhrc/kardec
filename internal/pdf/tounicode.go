package pdf

import (
	"bytes"
	"fmt"
)

// buildToUnicodeCMap returns the PDF /ToUnicode stream body for a
// font that uses the WinAnsiEncoding emitted by emitFont. The
// stream maps each WinAnsi byte the document actually uses back to
// its Unicode codepoint(s), so PDF readers can produce faithful
// text on copy/paste, find-in-page, and accessibility tooling.
//
// Format reference: PDF 1.7 §9.10.3 + Adobe Tech Note #5411
// (ToUnicode mapping file tutorial). The shape is a small CMap
// declaration:
//
//	/CIDInit /ProcSet findresource begin
//	12 dict begin
//	begincmap
//	  /CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def
//	  /CMapName /Adobe-Identity-UCS def
//	  /CMapType 2 def
//	  1 begincodespacerange
//	    <00> <FF>
//	  endcodespacerange
//	  N beginbfchar
//	    <41> <0041>
//	    <42> <0042>
//	    ...
//	  endbfchar
//	endcmap
//	CMapName currentdict /CMap defineresource pop
//	end end
//
// PDF readers consult this CMap when extracting text. Without it,
// ligatures or extended WinAnsi glyphs (smart quotes, em-dash, euro,
// the entire 0x80..0xFF range) come out as garbage.
func buildToUnicodeCMap() []byte {
	// Collect the WinAnsi → Unicode mapping in encoded form.
	type entry struct{ code, unicode uint32 }
	var entries []entry
	for code, u := range winAnsiToUnicode {
		if u == 0 {
			continue // unassigned slot — skip
		}
		entries = append(entries, entry{code: uint32(code), unicode: u})
	}

	var b bytes.Buffer
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\n")
	b.WriteString("begincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	b.WriteString("/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n")
	b.WriteString("<00> <FF>\n")
	b.WriteString("endcodespacerange\n")

	// PDF spec caps each beginbfchar block at 100 entries; chunk
	// the table so we never exceed that. The total stays well
	// under 256 entries (one per assigned WinAnsi byte) so two
	// blocks at most.
	for i := 0; i < len(entries); i += 100 {
		end := i + 100
		if end > len(entries) {
			end = len(entries)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-i)
		for _, e := range entries[i:end] {
			fmt.Fprintf(&b, "<%02X> <%04X>\n", e.code, e.unicode)
		}
		b.WriteString("endbfchar\n")
	}

	b.WriteString("endcmap\n")
	b.WriteString("CMapName currentdict /CMap defineresource pop\n")
	b.WriteString("end\n")
	b.WriteString("end\n")
	return b.Bytes()
}
