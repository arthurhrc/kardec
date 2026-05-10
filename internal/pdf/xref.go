package pdf

import (
	"fmt"
	"io"
	"strings"
)

// writeXrefAndTrailer emits the cross-reference table, trailer dictionary
// and startxref pointer that close out a PDF file (PDF 7.5.4 / 7.5.5).
//
// offsets stores body-local byte positions per object (as produced by
// objectWriter). headerLen shifts those positions to file-absolute by
// the size of the PDF header that precedes the body. nextID is one past
// the highest used ID.
//
// info is the indirect ID of the /Info dictionary (or 0 to omit it). root
// is the /Catalog ID and is always required.
//
// The xref table is written as a single subsection covering object IDs
// 0..nextID-1. Object 0 is the conventional "free" head of the free list
// with generation 65535.
func writeXrefAndTrailer(w io.Writer, offsets map[int]int64, headerLen int64, nextID, root, info int, idPair [2]string, encrypt int) error {
	var b strings.Builder
	b.WriteString("xref\n")
	fmt.Fprintf(&b, "0 %d\n", nextID)
	// Object 0 is always the free-list head: offset 0, gen 65535, flag 'f'.
	// Per PDF 7.5.4 each entry is exactly 20 bytes including the trailing
	// "\r\n" or " \n"; we use space + newline.
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i < nextID; i++ {
		off, ok := offsets[i]
		if !ok {
			// An allocated-but-never-written ID is a programming error;
			// emit a free entry to keep the table well-formed and let the
			// PDF still open, but the Writer's flow ensures every alloc
			// is paired with a write.
			b.WriteString("0000000000 00000 f \n")
			continue
		}
		fmt.Fprintf(&b, "%010d 00000 n \n", off+headerLen)
	}

	// Trailer: /Size = highest ID + 1 (i.e. nextID), /Root = catalog,
	// optional /Info, optional /ID. The /ID array is required by
	// PDF/A-2 and recommended by every other PDF flavor — it is two
	// 16-byte hex strings, the first identifying the original
	// document and the second the current revision (we make the
	// pair identical for first-revision output).
	b.WriteString("trailer\n<<")
	fmt.Fprintf(&b, " /Size %d /Root %s", nextID, ref(root))
	if info > 0 {
		fmt.Fprintf(&b, " /Info %s", ref(info))
	}
	if idPair[0] != "" {
		fmt.Fprintf(&b, " /ID [<%s><%s>]", idPair[0], idPair[1])
	}
	if encrypt > 0 {
		fmt.Fprintf(&b, " /Encrypt %s", ref(encrypt))
	}
	b.WriteString(" >>\n")

	if _, err := io.WriteString(w, b.String()); err != nil {
		return err
	}
	return nil
}

// writeStartxref emits "startxref\n<offset>\n%%EOF\n" — the file's last
// bytes (PDF 7.5.5).
func writeStartxref(w io.Writer, xrefOffset int64) error {
	_, err := fmt.Fprintf(w, "startxref\n%d\n%%%%EOF\n", xrefOffset)
	return err
}
