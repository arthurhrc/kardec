package pdf

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

// Writer turns a Document into a PDF 1.7 byte stream. The zero value is
// ready to use — Writer holds no state between calls.
type Writer struct{}

// pdfHeader is emitted verbatim at byte 0 of every output. The four
// high-bit bytes after the version comment are the standard "binary file"
// hint that prevents naive ASCII-mode FTP from corrupting the file
// (PDF 7.5.2).
const pdfHeader = "%PDF-1.7\n%\xE2\xE3\xCF\xD3\n"

// Write encodes doc as PDF 1.7 and writes the bytes to w. The stream is
// produced in a single pass; no temporary files are touched.
//
// Object layout (in write order):
//
//	1: /Catalog
//	2: /Pages tree root
//	3..3+nFonts*3-1: per-font triple (FontFile2 stream, FontDescriptor, /Font dict)
//	then for each page: content stream, /Page dict
//	last: /Info dictionary
//
// IDs are allocated lazily as objects are emitted; the layout above is
// the realized order, not a contract.
func (Writer) Write(w io.Writer, doc Document) error {
	if w == nil {
		return fmt.Errorf("pdf: nil io.Writer")
	}

	ow := newObjectWriter()

	// Allocate the catalog and pages-tree IDs up front because /Catalog
	// references /Pages and each /Page references the /Pages parent.
	catalogID := ow.allocID()
	pagesID := ow.allocID()

	// Embed every font once; the same /Font dict is referenced by any
	// page that uses it.
	handles := make([]*fontHandle, 0, len(doc.Fonts))
	for i, f := range doc.Fonts {
		fh, err := emitFont(ow, i, f)
		if err != nil {
			return err
		}
		handles = append(handles, fh)
	}

	// Emit each page. We decide which fonts are referenced by the page
	// and only list those in /Resources — keeps the dict small for
	// docs with many fonts.
	pageIDs := make([]int, 0, len(doc.Pages))
	for _, p := range doc.Pages {
		used := pageFonts(p, handles)
		raw := buildContentStream(p, handles)
		data, compressed := maybeFlate(raw)

		streamID := ow.allocID()
		dict := fmt.Sprintf("/Length %d", len(data))
		if compressed {
			dict += " /Filter /FlateDecode"
		}
		ow.writeStreamObject(streamID, dict, data)

		pageBody := fmt.Sprintf(
			"<< /Type /Page /Parent %s /MediaBox [0 0 %.4f %.4f] "+
				"/Resources %s /Contents %s >>",
			ref(pagesID),
			p.Width, p.Height,
			resourcesDict(used),
			ref(streamID),
		)
		pageIDs = append(pageIDs, ow.allocAndWrite(pageBody))
	}

	// Now that all page IDs are known, write /Pages.
	var kids bytes.Buffer
	kids.WriteString("[")
	for i, id := range pageIDs {
		if i > 0 {
			kids.WriteByte(' ')
		}
		kids.WriteString(ref(id))
	}
	kids.WriteString("]")
	pagesBody := fmt.Sprintf("<< /Type /Pages /Kids %s /Count %d >>", kids.String(), len(pageIDs))
	ow.writeObject(pagesID, pagesBody)

	// Catalog last among the structural objects — it points at /Pages.
	catalogBody := fmt.Sprintf("<< /Type /Catalog /Pages %s >>", ref(pagesID))
	ow.writeObject(catalogID, catalogBody)

	// Info dict (optional; /Producer "Kardec" + Title/Author when set).
	infoID := ow.allocID()
	infoBody := buildInfoDict(doc)
	ow.writeObject(infoID, infoBody)

	// Final emission: header, body, xref, trailer, startxref.
	if _, err := io.WriteString(w, pdfHeader); err != nil {
		return err
	}
	xrefOffset := int64(len(pdfHeader)) + ow.bodyLen()
	if _, err := ow.writeTo(w); err != nil {
		return err
	}
	if err := writeXrefAndTrailer(w, ow.offsets, ow.nextID, catalogID, infoID); err != nil {
		return err
	}
	if err := writeStartxref(w, xrefOffset); err != nil {
		return err
	}
	return nil
}

// pageFonts returns the subset of font handles actually referenced by p.
// Pages with no text reference no fonts (their /Resources still lists
// /ProcSet so the dict is non-empty).
func pageFonts(p Page, all []*fontHandle) []*fontHandle {
	if len(all) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(all))
	var used []*fontHandle
	for _, it := range p.Items {
		if it.FontID < 0 || it.FontID >= len(all) {
			continue
		}
		if seen[it.FontID] {
			continue
		}
		seen[it.FontID] = true
		used = append(used, all[it.FontID])
	}
	return used
}

// buildInfoDict assembles the /Info dictionary body. Title/Author are
// written as PDF literal strings (UTF-8 inside parens, escaped); Acrobat
// reads ASCII subsets correctly. Non-ASCII metadata would need a UTF-16BE
// "BOM-prefixed" string in v0.2.
func buildInfoDict(doc Document) string {
	var buf bytes.Buffer
	buf.WriteString("<<")
	if doc.Title != "" {
		fmt.Fprintf(&buf, " /Title %s", escapeLiteralString(doc.Title))
	}
	if doc.Author != "" {
		fmt.Fprintf(&buf, " /Author %s", escapeLiteralString(doc.Author))
	}
	fmt.Fprintf(&buf, " /Producer %s", escapeLiteralString("Kardec PDF Writer v0.1"))
	fmt.Fprintf(&buf, " /Creator %s", escapeLiteralString("Kardec"))
	// PDF date format: D:YYYYMMDDHHmmSSOHH'mm — see PDF 7.9.4. The 'Z'
	// timezone form (UTC) keeps this deterministic-ish for tests; callers
	// who need wall-clock dates can post-process.
	now := time.Now().UTC().Format("20060102150405")
	fmt.Fprintf(&buf, " /CreationDate (D:%sZ)", now)
	buf.WriteString(" >>")
	return buf.String()
}
