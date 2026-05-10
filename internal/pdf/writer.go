package pdf

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

// Writer turns a Document into a PDF 1.7 byte stream. The zero value
// is ready to use — Writer holds no inter-call state of its own.
//
// Clock supplies the timestamp written to the /Info /CreationDate
// entry. When nil, time.Now is used; tests and reproducible-build
// callers inject a fixed clock so two renders of the same document
// produce identical bytes.
type Writer struct {
	Clock func() time.Time
}

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
func (wr Writer) Write(w io.Writer, doc Document) error {
	if w == nil {
		return fmt.Errorf("pdf: nil io.Writer")
	}
	writerClock := wr.Clock

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

	// Embed every image as an XObject so multiple pages can reference
	// the same payload.
	imageHandles := make([]*imageHandle, 0, len(doc.Images))
	for i, img := range doc.Images {
		ih, err := emitImage(ow, i, img)
		if err != nil {
			return err
		}
		imageHandles = append(imageHandles, ih)
	}

	// Emit each page. We decide which fonts and images are referenced
	// by the page and only list those in /Resources — keeps the dict
	// small for docs with many fonts or many images.
	pageIDs := make([]int, 0, len(doc.Pages))
	for _, p := range doc.Pages {
		usedFonts := pageFonts(p, handles)
		usedImages := pageImages(p, imageHandles)
		raw := buildContentStream(p, handles, imageHandles)
		data, compressed := maybeFlate(raw)

		streamID := ow.allocID()
		dict := fmt.Sprintf("/Length %d", len(data))
		if compressed {
			dict += " /Filter /FlateDecode"
		}
		ow.writeStreamObject(streamID, dict, data)

		annotIDs := emitLinkAnnots(ow, p)
		pageBody := fmt.Sprintf(
			"<< /Type /Page /Parent %s /MediaBox [0 0 %.4f %.4f] "+
				"/Resources %s /Contents %s%s >>",
			ref(pagesID),
			p.Width, p.Height,
			resourcesDict(usedFonts, usedImages),
			ref(streamID),
			renderAnnotsArray(annotIDs),
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

	// Outlines come before the catalog so the catalog can reference
	// the root outline ID. emitOutlines returns 0 when no entries
	// exist, in which case we omit the /Outlines entry from the
	// catalog and skip the /PageMode hint as well.
	outlinesID := emitOutlines(ow, doc.Outlines, pageIDs)
	destsID := emitDestinations(ow, doc.Destinations, pageIDs)

	// Resolve the timestamp once so /Info /CreationDate, the XMP
	// xmp:CreateDate and the trailer /ID all share the same value.
	now := w_clockOrDefault(writerClock)

	// Optional PDF/A metadata stream — emitted before the catalog
	// so the catalog can reference it.
	metadataID := 0
	outputIntentsID := 0
	if doc.PDFA {
		metadataID = emitPDFAMetadata(ow, doc, now)
		outputIntentsID = emitOutputIntent(ow, doc)
	}

	// Catalog last among the structural objects — it points at /Pages
	// and optionally at /Outlines / /Dests / /Metadata / /OutputIntents.
	catalogBody := fmt.Sprintf("<< /Type /Catalog /Pages %s", ref(pagesID))
	if outlinesID > 0 {
		catalogBody += fmt.Sprintf(" /Outlines %s /PageMode /UseOutlines", ref(outlinesID))
	}
	if destsID > 0 {
		catalogBody += fmt.Sprintf(" /Dests %s", ref(destsID))
	}
	if metadataID > 0 {
		catalogBody += fmt.Sprintf(" /Metadata %s", ref(metadataID))
	}
	if outputIntentsID > 0 {
		catalogBody += fmt.Sprintf(" /OutputIntents %s", ref(outputIntentsID))
	}
	catalogBody += " >>"
	ow.writeObject(catalogID, catalogBody)

	// Info dict (optional; /Producer "Kardec" + Title/Author when set).
	infoID := ow.allocID()
	infoBody := buildInfoDict(doc, now)
	ow.writeObject(infoID, infoBody)

	// Final emission: header, body, xref, trailer, startxref.
	if _, err := io.WriteString(w, pdfHeader); err != nil {
		return err
	}
	xrefOffset := int64(len(pdfHeader)) + ow.bodyLen()
	if _, err := ow.writeTo(w); err != nil {
		return err
	}
	var idPair [2]string
	if doc.PDFA {
		idA, idB := stableDocumentID(doc, now)
		idPair = [2]string{idA, idB}
	}
	if err := writeXrefAndTrailer(w, ow.offsets, int64(len(pdfHeader)), ow.nextID, catalogID, infoID, idPair); err != nil {
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

// pageImages returns the subset of image handles actually referenced
// by p, deduplicated. Pages without images return nil so the resource
// dict can omit the /XObject section entirely.
func pageImages(p Page, all []*imageHandle) []*imageHandle {
	if len(all) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(all))
	var used []*imageHandle
	for _, im := range p.Images {
		if im.ImageID < 0 || im.ImageID >= len(all) {
			continue
		}
		if seen[im.ImageID] {
			continue
		}
		seen[im.ImageID] = true
		used = append(used, all[im.ImageID])
	}
	return used
}

// buildInfoDict assembles the /Info dictionary body. Title/Author are
// written as PDF literal strings (UTF-8 inside parens, escaped); Acrobat
// reads ASCII subsets correctly. Non-ASCII metadata would need a UTF-16BE
// "BOM-prefixed" string in v0.2.
func buildInfoDict(doc Document, now time.Time) string {
	var buf bytes.Buffer
	buf.WriteString("<<")
	if doc.Title != "" {
		fmt.Fprintf(&buf, " /Title %s", escapeLiteralString(doc.Title))
	}
	if doc.Author != "" {
		fmt.Fprintf(&buf, " /Author %s", escapeLiteralString(doc.Author))
	}
	if doc.Subject != "" {
		fmt.Fprintf(&buf, " /Subject %s", escapeLiteralString(doc.Subject))
	}
	if doc.Keywords != "" {
		fmt.Fprintf(&buf, " /Keywords %s", escapeLiteralString(doc.Keywords))
	}
	fmt.Fprintf(&buf, " /Producer %s", escapeLiteralString("Kardec PDF Writer v0.1"))
	fmt.Fprintf(&buf, " /Creator %s", escapeLiteralString("Kardec"))
	// PDF date format: D:YYYYMMDDHHmmSSOHH'mm — see PDF 7.9.4. The 'Z'
	// timezone form (UTC) keeps the value comparable across machines;
	// the caller passes the wall-clock or a fixed moment for
	// reproducible builds.
	stamp := now.UTC().Format("20060102150405")
	fmt.Fprintf(&buf, " /CreationDate (D:%sZ)", stamp)
	buf.WriteString(" >>")
	return buf.String()
}

// w_clockOrDefault resolves the Writer.Clock seam. A nil clock falls
// back to time.Now so the default Writer remains usable without any
// configuration; non-nil clocks are returned verbatim so callers can
// inject deterministic timestamps for reproducibility tests.
func w_clockOrDefault(c func() time.Time) time.Time {
	if c == nil {
		return time.Now()
	}
	return c()
}
