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

	// Resolve the timestamp once so /Info /CreationDate, the XMP
	// xmp:CreateDate, the trailer /ID, and the encryption key
	// derivation all share the same value.
	now := w_clockOrDefault(writerClock)

	// /ID array is emitted in the trailer when either PDF/A or
	// encryption is on. Encryption needs the first /ID component
	// to derive the file encryption key (spec algorithm 3.2 step
	// 4), so compute it up front.
	idA, idB := stableDocumentID(doc, now)

	ow := newObjectWriter()

	// Set up Standard Security Handler (V=4 / R=4 / AES-128) when
	// the caller asked for it. Streams are then encrypted on the
	// fly inside writeStreamObject; the /Encrypt indirect-object
	// is emitted before any stream objects so the trailer's
	// /Encrypt entry has its target ID resolved.
	encryptID := 0
	if doc.Encryption != nil {
		idABytes := decodeHexID(idA)
		oHash := computeOwnerHash(doc.Encryption.UserPwd, doc.Encryption.OwnerPwd)
		fileKey := computeEncryptionKey(doc.Encryption.UserPwd, oHash, doc.Encryption.Permissions, idABytes)
		uHash := computeUserHash(fileKey, idABytes)
		ow.fileKey = fileKey
		encryptID = ow.allocID()
		ow.writeObject(encryptID, buildEncryptDict(oHash, uHash, doc.Encryption.Permissions))
	}

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

	// Watermark /ExtGState (alpha) is allocated up front so every
	// page's resource dict can forward-reference the same object.
	// Opacity ≥ 1 (or ≤ 0) returns 0,"" — the resource entry is
	// then skipped entirely.
	wmAlphaID := 0
	wmAlphaName := ""
	if doc.Watermark != nil && doc.Watermark.Text != "" {
		wmAlphaID, wmAlphaName = emitWatermarkAlpha(ow, doc.Watermark.Opacity)
	}

	// Pre-allocate per-block StructElem IDs in tagged mode so each
	// /Page dict can forward-reference its owning structure
	// elements through /StructParents N. Each page allocates one
	// ID per StructElem in its tree (counting nested children).
	// Pages without StructBlocks fall back to one synthetic /P
	// element covering the whole page (lite mode).
	var pageBlockElemIDs [][]int
	if doc.Tagged {
		pageBlockElemIDs = make([][]int, len(doc.Pages))
		for i, p := range doc.Pages {
			total := 0
			for _, b := range p.StructBlocks {
				total += countBlocksInTree(b)
			}
			if total == 0 {
				total = 1 // lite-mode placeholder
			}
			pageBlockElemIDs[i] = make([]int, total)
			for j := range pageBlockElemIDs[i] {
				pageBlockElemIDs[i][j] = ow.allocID()
			}
		}
	}

	// Emit each page. We decide which fonts and images are referenced
	// by the page and only list those in /Resources — keeps the dict
	// small for docs with many fonts or many images.
	pageIDs := make([]int, 0, len(doc.Pages))
	for pageIdx, p := range doc.Pages {
		usedFonts := pageFonts(p, handles)
		usedImages := pageImages(p, imageHandles)
		// Make sure the watermark font is in the page's /Font dict
		// even if no body text on that page references it.
		if doc.Watermark != nil && doc.Watermark.Text != "" {
			usedFonts = ensureFontIncluded(usedFonts, handles, doc.Watermark.FontID)
		}
		raw := buildContentStreamWithWatermark(p, handles, imageHandles, doc.Watermark, wmAlphaName)
		// In tagged mode wrap each block (or the whole page in
		// lite mode) in its own marked-content sequence. The
		// per-block path produces real PDF/UA semantics; the
		// fallback keeps backward-compat with v0.17 docs that
		// don't carry block roles.
		if doc.Tagged {
			if len(p.StructBlocks) > 0 {
				raw = wrapMarkedContentByBlocks(p, handles, imageHandles, doc.Watermark, wmAlphaName)
			} else {
				raw = wrapMarkedContent(raw, "P", 0)
			}
		}
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
				"/Resources %s /Contents %s%s",
			ref(pagesID),
			p.Width, p.Height,
			resourcesDict(usedFonts, usedImages, wmAlphaName, wmAlphaID),
			ref(streamID),
			renderAnnotsArray(annotIDs),
		)
		if doc.Tagged {
			// /StructParents is an integer key into the
			// /StructTreeRoot's ParentTree number tree
			// (PDF 14.7.4.4). One MCID per page in lite mode
			// means a 1:1 map — page index N → ParentTree[N].
			pageBody += fmt.Sprintf(" /StructParents %d", pageIdx)
			// /Tabs /S enforces logical reading order for
			// assistive tech: tab through annotations in
			// structure order. PDF/UA requires this.
			pageBody += " /Tabs /S"
		}
		pageBody += " >>"
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

	// Optional PDF/A metadata stream — emitted before the catalog
	// so the catalog can reference it.
	metadataID := 0
	outputIntentsID := 0
	if doc.PDFA {
		metadataID = emitPDFAMetadata(ow, doc, now)
		outputIntentsID = emitOutputIntent(ow, doc)
	}

	// Optional structure tree (PDF/UA tagging) — emitted before the
	// catalog so the catalog can reference the StructTreeRoot.
	structTreeRootID := 0
	if doc.Tagged {
		structTreeRootID = emitStructTree(ow, pageIDs, doc.Pages, pageBlockElemIDs)
	}

	// Catalog last among the structural objects — it points at /Pages
	// and optionally at /Outlines / /Dests / /Metadata / /OutputIntents
	// / /StructTreeRoot / /MarkInfo / /Lang.
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
	if structTreeRootID > 0 {
		catalogBody += fmt.Sprintf(" /StructTreeRoot %s /MarkInfo << /Marked true >>", ref(structTreeRootID))
	}
	if doc.Lang != "" {
		catalogBody += fmt.Sprintf(" /Lang %s", escapeLiteralString(doc.Lang))
	}
	catalogBody += " >>"
	ow.writeObject(catalogID, catalogBody)

	// Info dict (optional; /Producer "Kardec" + Title/Author when set).
	infoID := ow.allocID()
	infoBody := buildInfoDict(doc, now, infoID, ow.fileKey)
	ow.writeObject(infoID, infoBody)

	// Final emission: header, body, xref, trailer, startxref.
	if _, err := io.WriteString(w, pdfHeader); err != nil {
		return err
	}
	xrefOffset := int64(len(pdfHeader)) + ow.bodyLen()
	if _, err := ow.writeTo(w); err != nil {
		return err
	}
	// /ID is required in the trailer for both PDF/A AND any
	// encryption configuration (spec §7.6.3.4). Always emit
	// when either is on; otherwise the trailer omits /ID.
	var idPair [2]string
	if doc.PDFA || doc.Encryption != nil {
		idPair = [2]string{idA, idB}
	}
	if err := writeXrefAndTrailer(w, ow.offsets, int64(len(pdfHeader)), ow.nextID, catalogID, infoID, idPair, encryptID); err != nil {
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

// ensureFontIncluded appends all[fontID] to used when it isn't
// already present. The watermark text uses an ID outside the body's
// reachable set on pages that have no matching glyph runs; without
// this the page's /Font dict would omit the face and Acrobat would
// render the watermark as missing-glyph blocks.
func ensureFontIncluded(used []*fontHandle, all []*fontHandle, fontID int) []*fontHandle {
	if fontID < 0 || fontID >= len(all) {
		return used
	}
	target := all[fontID]
	for _, fh := range used {
		if fh == target {
			return used
		}
	}
	return append(used, target)
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

// buildInfoDict assembles the /Info dictionary body. Title/Author/...
// are written as PDF literal strings; when fileKey is non-nil they
// are first encrypted with the indirect object's per-object key
// (PDF 7.6.3.4 algorithm 1.A) so confidential metadata stays
// confidential under encryption — the v0.22 fix for the v0.16
// regression that left /Info plaintext.
//
// objNum is the indirect-object ID this dict will live inside;
// the encryption key is derived from it.
func buildInfoDict(doc Document, now time.Time, objNum int, fileKey []byte) string {
	var buf bytes.Buffer
	buf.WriteString("<<")
	if doc.Title != "" {
		fmt.Fprintf(&buf, " /Title %s", encryptString(fileKey, objNum, doc.Title))
	}
	if doc.Author != "" {
		fmt.Fprintf(&buf, " /Author %s", encryptString(fileKey, objNum, doc.Author))
	}
	if doc.Subject != "" {
		fmt.Fprintf(&buf, " /Subject %s", encryptString(fileKey, objNum, doc.Subject))
	}
	if doc.Keywords != "" {
		fmt.Fprintf(&buf, " /Keywords %s", encryptString(fileKey, objNum, doc.Keywords))
	}
	fmt.Fprintf(&buf, " /Producer %s", encryptString(fileKey, objNum, "Kardec PDF Writer v0.1"))
	fmt.Fprintf(&buf, " /Creator %s", encryptString(fileKey, objNum, "Kardec"))
	// PDF date format: D:YYYYMMDDHHmmSSOHH'mm — see PDF 7.9.4. The 'Z'
	// timezone form (UTC) keeps the value comparable across machines;
	// the caller passes the wall-clock or a fixed moment for
	// reproducible builds.
	stamp := now.UTC().Format("20060102150405")
	fmt.Fprintf(&buf, " /CreationDate %s", encryptString(fileKey, objNum, "D:"+stamp+"Z"))
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
