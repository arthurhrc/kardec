package pdf

import (
	"fmt"
	"strings"
)

// emitCFFFont writes the indirect-object set required for a Type 0
// composite font with CFF outlines:
//
//  1. FontFile3 stream — the raw 'CFF ' table bytes from the OTF
//     SFNT, with /Subtype /CIDFontType0C so PDF readers know the
//     stream is a Compact Font Format dictionary (PDF 9.9 Table
//     127, row 3).
//  2. FontDescriptor — same shape as the TrueType path, but
//     references FontFile3 instead of FontFile2.
//  3. CIDFont (Type 0) — the descendant font that owns the
//     glyph data. /CIDSystemInfo declares the Adobe / Identity / 0
//     ROS so /Encoding /Identity-H on the parent maps CIDs ↔
//     glyph IDs without a separate CIDToGIDMap.
//  4. Type 0 wrapper — the resource the content stream references.
//     /Encoding /Identity-H, /DescendantFonts [<CIDFont>],
//     /ToUnicode <CMap> for faithful text extraction.
//
// The math font path (Latin Modern Math) is the canonical user.
// Generic OTF fonts callers register via Document.RegisterFont
// flow through the same code.
func emitCFFFont(ow *objectWriter, idx int, font EmbeddedFont, metrics *ttfMetrics, psName string) (*fontHandle, error) {
	if len(metrics.CFFData) == 0 {
		return nil, fmt.Errorf("pdf: font %q: OTTO scaler but no 'CFF ' table found", font.Name)
	}

	// 1. FontFile3 stream carrying the raw CFF table bytes.
	cff := metrics.CFFData
	compressed := flateAlways(cff)
	streamID := ow.allocID()
	streamDict := fmt.Sprintf(
		"/Length %d /Length1 %d /Subtype /CIDFontType0C /Filter /FlateDecode",
		len(compressed), len(cff),
	)
	ow.writeStreamObject(streamID, streamDict, compressed)

	// 2. FontDescriptor.
	flags := computeFontFlags(metrics)
	scale := func(v int16) int {
		if metrics.UnitsPerEm == 0 {
			return 0
		}
		return int(float64(v) * 1000.0 / float64(metrics.UnitsPerEm))
	}
	descriptorBody := fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags %d "+
			"/FontBBox [%d %d %d %d] /ItalicAngle %.1f "+
			"/Ascent %d /Descent %d /CapHeight %d /StemV %d "+
			"/FontFile3 %s >>",
		psName, flags,
		scale(metrics.XMin), scale(metrics.YMin), scale(metrics.XMax), scale(metrics.YMax),
		metrics.ItalicAngle,
		scale(metrics.Ascent), scale(metrics.Descent), scale(metrics.CapHeight),
		metrics.StemV,
		ref(streamID),
	)
	descriptorID := ow.allocAndWrite(descriptorBody)

	// 3. /W array — per-glyph advance widths in 1/1000 em. Type 0
	// fonts use a different format than simple-font /Widths: pairs
	// of `cid [w1 w2 ...]` covering ranges of contiguous glyphs.
	// Fonts with a uniform default width can skip the W array
	// entirely and rely on /DW; our case ships W to keep math
	// glyph spacing accurate.
	widthsArr := buildCFFWidthArray(metrics, scale)

	// 4. CIDFont descendant.
	cidFontBody := fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType0 /BaseFont /%s "+
			"/CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> "+
			"/FontDescriptor %s "+
			"/DW 500 "+
			"/W %s >>",
		psName,
		ref(descriptorID),
		widthsArr,
	)
	cidFontID := ow.allocAndWrite(cidFontBody)

	// 5. ToUnicode CMap so text extraction maps CIDs back to
	// Unicode codepoints. Built from the cmap subtable parsed
	// earlier; the helper writes a chunked beginbfchar block
	// covering every code-point the cmap claimed.
	toUnicodeID := emitCFFToUnicodeCMap(ow, metrics)

	// 6. Type 0 wrapper — what the page's /Resources /Font dict
	// actually points at.
	typeZeroBody := fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /%s "+
			"/Encoding /Identity-H /DescendantFonts [%s] "+
			"/ToUnicode %s >>",
		psName,
		ref(cidFontID),
		ref(toUnicodeID),
	)
	fontID := ow.allocAndWrite(typeZeroBody)

	return &fontHandle{
		Name:    fmt.Sprintf("F%d", idx),
		DictID:  fontID,
		Metrics: metrics,
		Kind:    fontKindCFF,
	}, nil
}

// buildCFFWidthArray emits the /W entry for a CIDFontType0 — a
// sparse array describing per-glyph advance widths. The layout
// `cid [w1 w2 w3 ...]` means glyph cid has width w1, cid+1 has w2,
// etc. We emit one such entry per advance-width chunk seen in the
// metrics (typically a single contiguous run from glyph 0).
func buildCFFWidthArray(m *ttfMetrics, scale func(int16) int) string {
	if len(m.AdvanceWidth) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteString("[0 [")
	for i, w := range m.AdvanceWidth {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%d", scale(int16(w)))
	}
	b.WriteString("]]")
	return b.String()
}

// emitCFFToUnicodeCMap writes a /ToUnicode CMap for a CFF font: it
// inverts the cmap (codepoint → glyph) into a glyph → codepoint
// mapping the PDF reader uses for text extraction. Returns the
// indirect-object ID.
func emitCFFToUnicodeCMap(ow *objectWriter, m *ttfMetrics) int {
	// Build glyph-id → codepoint (first one wins on glyph reuse;
	// multiple codepoints sharing a glyph is fine for our purposes
	// because the math font uses 1:1 mapping).
	glyphToUni := make(map[uint16]uint32, len(m.CmapUnicode))
	for cp, gid := range m.CmapUnicode {
		if _, exists := glyphToUni[gid]; !exists {
			glyphToUni[gid] = cp
		}
	}

	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\n")
	b.WriteString("begincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	b.WriteString("/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n")
	b.WriteString("<0000> <FFFF>\n")
	b.WriteString("endcodespacerange\n")

	// Sort glyph IDs for deterministic output.
	gids := make([]uint16, 0, len(glyphToUni))
	for gid := range glyphToUni {
		gids = append(gids, gid)
	}
	sortUint16Asc(gids)

	for i := 0; i < len(gids); i += 100 {
		end := i + 100
		if end > len(gids) {
			end = len(gids)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-i)
		for _, g := range gids[i:end] {
			fmt.Fprintf(&b, "<%04X> <%04X>\n", g, glyphToUni[g])
		}
		b.WriteString("endbfchar\n")
	}

	b.WriteString("endcmap\n")
	b.WriteString("CMapName currentdict /CMap defineresource pop\n")
	b.WriteString("end\n")
	b.WriteString("end\n")

	body := b.String()
	compressed := flateAlways([]byte(body))
	id := ow.allocID()
	ow.writeStreamObject(id, fmt.Sprintf(
		"/Length %d /Filter /FlateDecode",
		len(compressed),
	), compressed)
	return id
}

// sortUint16Asc sorts xs ascending in place. Standalone to keep the
// pdf package out of the sort dependency for one tiny call site.
func sortUint16Asc(xs []uint16) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
			xs[j-1], xs[j] = xs[j], xs[j-1]
		}
	}
}
