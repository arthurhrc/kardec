package sign

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// injectSignaturePlaceholder appends the PDF objects a signature
// needs (the /Sig dict, the AcroForm catalog reference, the
// signature field) to pdfBytes and returns:
//
//	(newPDF, byteRange, err)
//
// byteRange is the 4-element /ByteRange that says "everything
// except the /Contents value": [0, before-contents, after-contents,
// rest-of-file].
//
// The strategy is "incremental update" (PDF 7.5.6): re-emit the
// original PDF verbatim, then append new objects + xref + trailer
// pointing at them. The original xref stays valid for everything
// it covered; readers find the new objects via the new xref's
// /Prev chain.
//
// The placeholder /Contents is filled with zeros (a fixed-size
// hex string). The caller computes the hash over byteRange,
// signs it, hex-encodes the result, and replaces the zeros with
// the real signature.
func injectSignaturePlaceholder(
	pdfBytes []byte,
	placeholderHex string,
	reason, location, signerName string,
	signingTime time.Time,
) (newPDF []byte, byteRange [4]int, err error) {
	// Find the existing xref offset (the number after the last
	// "startxref" line) — we need it for the incremental update's
	// /Prev entry.
	prevXref, err := findLastStartXref(pdfBytes)
	if err != nil {
		return nil, byteRange, fmt.Errorf("find startxref: %w", err)
	}
	// Find the highest indirect-object ID currently in use. We
	// allocate the next 3 IDs for our new objects (sig dict,
	// signature field, acroform).
	nextID, err := findNextObjID(pdfBytes)
	if err != nil {
		return nil, byteRange, fmt.Errorf("find next id: %w", err)
	}
	// Find the current /Root (catalog) ID — the trailer points at
	// it, and we need to extend it with our /AcroForm reference.
	catalogID, err := findCatalogID(pdfBytes)
	if err != nil {
		return nil, byteRange, fmt.Errorf("find catalog: %w", err)
	}

	sigID := nextID
	sigFieldID := nextID + 1
	acroFormID := nextID + 2
	catalogReplacementID := nextID + 3

	// The PDF date format is `D:YYYYMMDDHHmmSSOHH'mm'`. We use the
	// `Z` (UTC) form for simplicity.
	pdfDate := signingTime.UTC().Format("20060102150405")

	// Build the signature dictionary. /Contents is the hex
	// placeholder — when the caller computes the real signature
	// it will replace these zeros byte-for-byte.
	var sigBuf bytes.Buffer
	fmt.Fprintf(&sigBuf, "%d 0 obj\n", sigID)
	sigBuf.WriteString("<< /Type /Sig /Filter /Adobe.PPKLite /SubFilter /adbe.pkcs7.detached")
	if reason != "" {
		fmt.Fprintf(&sigBuf, " /Reason %s", escapePDFString(reason))
	}
	if location != "" {
		fmt.Fprintf(&sigBuf, " /Location %s", escapePDFString(location))
	}
	if signerName != "" {
		fmt.Fprintf(&sigBuf, " /Name %s", escapePDFString(signerName))
	}
	fmt.Fprintf(&sigBuf, " /M (D:%sZ)", pdfDate)
	// /ByteRange placeholder — fixed-width so the offset of
	// /Contents stays predictable. Format `[0 NNNNNNNNNN NNNNNNNNNN NNNNNNNNNN]`
	// reserves ten decimal digits per slot, more than enough for any
	// PDF this writer produces.
	sigBuf.WriteString(" /ByteRange [0000000000 0000000000 0000000000 0000000000] /Contents <")
	contentsStart := sigBuf.Len() // record where the < is
	sigBuf.WriteString(placeholderHex)
	contentsEnd := sigBuf.Len() // record where the > would go
	sigBuf.WriteString("> >>\nendobj\n")

	// Build the signature widget annotation (the field).
	var fieldBuf bytes.Buffer
	fmt.Fprintf(&fieldBuf, "%d 0 obj\n", sigFieldID)
	fmt.Fprintf(&fieldBuf,
		"<< /Type /Annot /Subtype /Widget /F 132 /FT /Sig /Rect [0 0 0 0] /T (Signature1) /V %d 0 R >>\nendobj\n",
		sigID)

	// AcroForm catalog entry: /Fields [sigFieldID], /SigFlags 3
	// (signed + appended).
	var acroBuf bytes.Buffer
	fmt.Fprintf(&acroBuf, "%d 0 obj\n", acroFormID)
	fmt.Fprintf(&acroBuf,
		"<< /Fields [%d 0 R] /SigFlags 3 >>\nendobj\n",
		sigFieldID)

	// We also need to PATCH the catalog so it carries /AcroForm.
	// PDF incremental updates allow re-emitting the catalog with
	// the same indirect ID — readers use the newer copy.
	// Recover the existing catalog body so the new copy preserves
	// every entry.
	existingCatalog, err := extractCatalogBody(pdfBytes, catalogID)
	if err != nil {
		return nil, byteRange, fmt.Errorf("extract catalog: %w", err)
	}
	patchedCatalog := injectAcroFormIntoCatalog(existingCatalog, acroFormID)
	var catBuf bytes.Buffer
	fmt.Fprintf(&catBuf, "%d 0 obj\n%s\nendobj\n", catalogID, patchedCatalog)

	// Assemble the appendage. Order: sig, field, acroform,
	// catalog-replacement, then xref + trailer.
	var append1 bytes.Buffer
	if !bytes.HasSuffix(pdfBytes, []byte("\n")) {
		append1.WriteByte('\n')
	}
	originalLen := len(pdfBytes) + append1.Len()
	append1.Write(sigBuf.Bytes())
	append1.Write(fieldBuf.Bytes())
	append1.Write(acroBuf.Bytes())
	append1.Write(catBuf.Bytes())

	// xref offsets — relative to the start of the final file.
	sigOffset := originalLen
	fieldOffset := sigOffset + sigBuf.Len()
	acroOffset := fieldOffset + fieldBuf.Len()
	catReplaceOffset := acroOffset + acroBuf.Len()
	xrefOffset := catReplaceOffset + catBuf.Len()

	// xref + trailer for the incremental update. /Prev links to
	// the original xref so the reader's chain reaches every
	// object.
	var trailerBuf bytes.Buffer
	trailerBuf.WriteString("xref\n")
	// Section 1: object 0 (free).
	trailerBuf.WriteString("0 1\n0000000000 65535 f \n")
	// Section 2: the updated catalog at catalogID.
	fmt.Fprintf(&trailerBuf, "%d 1\n%010d 00000 n \n", catalogID, catReplaceOffset)
	// Section 3: new objects sigID, sigFieldID, acroFormID.
	fmt.Fprintf(&trailerBuf, "%d 3\n%010d 00000 n \n%010d 00000 n \n%010d 00000 n \n",
		sigID, sigOffset, fieldOffset, acroOffset)
	// Trailer.
	fmt.Fprintf(&trailerBuf,
		"trailer\n<< /Size %d /Prev %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		catalogReplacementID+1, prevXref, catalogID, xrefOffset)

	out := make([]byte, 0, len(pdfBytes)+append1.Len()+trailerBuf.Len())
	out = append(out, pdfBytes...)
	out = append(out, append1.Bytes()...)
	out = append(out, trailerBuf.Bytes()...)

	// Compute /ByteRange. The /Contents value lives between
	// `<` and `>` inside the signature dict. ByteRange[0] = 0,
	// ByteRange[1] = byte offset of the `<` + 1 (inclusive of
	// the `<`), ByteRange[2] = byte offset of the matching `>`,
	// ByteRange[3] = file length − ByteRange[2].
	//
	// sigOffset is where the `%d 0 obj\n` header lands. We
	// recorded contentsStart / contentsEnd as offsets within
	// sigBuf — the `<` is at contentsStart-1, the `>` is at
	// contentsEnd. (Counting from after the placeholder.)
	contentsLT := sigOffset + contentsStart - 1
	contentsGT := sigOffset + contentsEnd
	byteRange[0] = 0
	byteRange[1] = contentsLT + 1 // bytes 0..contentsLT inclusive
	byteRange[2] = contentsGT     // bytes from contentsGT onward
	byteRange[3] = len(out) - contentsGT

	// Patch the /ByteRange placeholder with the actual values.
	out = writeByteRange(out, sigOffset, byteRange)

	return out, byteRange, nil
}

// writeByteRange substitutes the 4 ten-digit zero placeholders in
// the signature dict with the real /ByteRange values. The dict's
// `/ByteRange [0000000000 0000000000 0000000000 0000000000]`
// becomes e.g. `/ByteRange [0000000000 0000010234 0000026766 0001234567]`
// while keeping the same byte count so file offsets stay valid.
func writeByteRange(pdf []byte, sigOffset int, br [4]int) []byte {
	// Find the literal `[0000000000 0000000000 0000000000 0000000000]`
	// inside the sig object and replace with the formatted values.
	old := []byte("[0000000000 0000000000 0000000000 0000000000]")
	new := []byte(fmt.Sprintf("[%010d %010d %010d %010d]", br[0], br[1], br[2], br[3]))
	if len(old) != len(new) {
		panic("byterange replacement length mismatch")
	}
	// We know it sits inside the signature dict — search forward
	// from sigOffset to avoid accidentally matching another zero
	// run earlier in the file.
	if sigOffset >= len(pdf) {
		return pdf
	}
	idx := bytes.Index(pdf[sigOffset:], old)
	if idx < 0 {
		return pdf
	}
	copy(pdf[sigOffset+idx:sigOffset+idx+len(new)], new)
	return pdf
}

// escapePDFString wraps s in parens with the four PDF-literal
// escapes (\n, \r, \(, \)) applied.
func escapePDFString(s string) string {
	var b strings.Builder
	b.WriteByte('(')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '(':
			b.WriteString("\\(")
		case ')':
			b.WriteString("\\)")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(')')
	return b.String()
}
