package pdf

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// emitPDFAMetadata writes an XMP metadata stream declaring PDF/A-2b
// conformance and returns its indirect-object ID. The catalog
// references this object via a /Metadata entry.
//
// The XMP packet declares the canonical PDF/A namespace tags
// (pdfaid:part=2, pdfaid:conformance=B) plus a small set of
// dc:title / dc:creator entries derived from doc metadata. PDF/A
// requires the document-info /CreationDate to be mirrored here in
// xmp:CreateDate (ISO-8601 form); the supplied now is used so the
// timestamps line up.
func emitPDFAMetadata(ow *objectWriter, doc Document, now time.Time) int {
	xmp := buildXMPPacket(doc, now)
	id := ow.allocID()
	dict := fmt.Sprintf("/Type /Metadata /Subtype /XML /Length %d", len(xmp))
	ow.writeStreamObject(id, dict, []byte(xmp))
	return id
}

// buildXMPPacket assembles the XMP packet body. Kept verbatim and
// minimal — strict PDF/A-2b validators want a specific set of
// rdf:Description blocks and namespace declarations; this packet
// matches the layout most off-the-shelf XMP-aware tools produce.
func buildXMPPacket(doc Document, now time.Time) string {
	createDate := now.UTC().Format("2006-01-02T15:04:05Z")
	title := xmpEscape(doc.Title)
	author := xmpEscape(doc.Author)
	var b strings.Builder
	b.WriteString("<?xpacket begin=\"\xEF\xBB\xBF\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>")
	b.WriteString(`<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Kardec PDF Writer">`)
	b.WriteString(`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">`)
	b.WriteString(`<rdf:Description rdf:about=""`)
	b.WriteString(` xmlns:dc="http://purl.org/dc/elements/1.1/"`)
	b.WriteString(` xmlns:pdf="http://ns.adobe.com/pdf/1.3/"`)
	b.WriteString(` xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/"`)
	b.WriteString(` xmlns:xmp="http://ns.adobe.com/xap/1.0/">`)
	b.WriteString(`<pdfaid:part>2</pdfaid:part>`)
	b.WriteString(`<pdfaid:conformance>B</pdfaid:conformance>`)
	b.WriteString(`<pdf:Producer>Kardec PDF Writer v0.5</pdf:Producer>`)
	b.WriteString(`<xmp:CreateDate>` + createDate + `</xmp:CreateDate>`)
	b.WriteString(`<xmp:CreatorTool>Kardec</xmp:CreatorTool>`)
	if title != "" {
		b.WriteString(`<dc:title><rdf:Alt><rdf:li xml:lang="x-default">`)
		b.WriteString(title)
		b.WriteString(`</rdf:li></rdf:Alt></dc:title>`)
	}
	if author != "" {
		b.WriteString(`<dc:creator><rdf:Seq><rdf:li>`)
		b.WriteString(author)
		b.WriteString(`</rdf:li></rdf:Seq></dc:creator>`)
	}
	b.WriteString(`</rdf:Description></rdf:RDF></x:xmpmeta>`)
	b.WriteString(`<?xpacket end="w"?>`)
	return b.String()
}

// xmpEscape escapes the four characters XML packets cannot carry
// raw inside element bodies. Attribute values would also need
// quote escaping; the helper currently only handles element-body
// contexts since dc:title / dc:creator land there.
func xmpEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return r.Replace(s)
}

// stableDocumentID derives a 16-byte hex pair the trailer's /ID
// array carries. Inputs are doc.Title + doc.Author + the supplied
// timestamp, MD5'd to produce a stable hash. Two renders of the
// same document with the same SetCreationDate yield byte-identical
// IDs, preserving the reproducibility guarantee shipped in v0.3.
func stableDocumentID(doc Document, now time.Time) (string, string) {
	seed := doc.Title + "\x00" + doc.Author + "\x00" + now.UTC().Format(time.RFC3339Nano)
	sum := md5.Sum([]byte(seed))
	hexed := strings.ToUpper(hex.EncodeToString(sum[:]))
	return hexed, hexed
}
