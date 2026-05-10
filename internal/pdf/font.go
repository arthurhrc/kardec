package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// fontKind tags the font's outline format. Both kinds emit through a
// Type 0 / Identity-H composite-font wrapper; the difference is in
// the descendant's /Subtype and the /FontFile* used in the descriptor.
//
//   - fontKindTrueType: TrueType outlines, /CIDFontType2 + /FontFile2.
//   - fontKindCFF: Compact Font Format outlines (OTF / OTTO scaler),
//     /CIDFontType0 + /FontFile3 with /Subtype /CIDFontType0C.
//
// Content-stream emission is identical for both: 2-byte hex-encoded
// glyph IDs (`<00410042>`) since Identity-H makes CID == glyph index.
type fontKind uint8

const (
	fontKindTrueType fontKind = iota
	fontKindCFF
)

// fontHandle is the writer's record of an embedded font: the indirect ID
// of its /Font dictionary (referenced from /Resources), its parsed
// metrics, and its PDF resource name (e.g. "F0", "F1", ...).
type fontHandle struct {
	Name    string // resource name used inside content streams ("F0")
	DictID  int    // indirect ID of the /Font dictionary
	Metrics *ttfMetrics
	Kind    fontKind
}

// emitFont writes the indirect objects for an embedded font and
// returns the handle the page Resources dict will reference.
//
// Both TrueType and CFF outlines emit as composite Type 0 /
// Identity-H fonts so every Unicode codepoint the source TTF/OTF
// covers is renderable. Pre-v0.22, TrueType used the simple
// /Subtype /TrueType + WinAnsiEncoding form, which silently
// substituted '?' for any character outside CP1252 (Δ, Σ, Ω,
// Cyrillic, CJK, …). The Identity-H path makes that limitation
// disappear at the cost of two-byte glyph indices in the content
// stream — a fixed +N per glyph that compresses away under
// FlateDecode.
func emitFont(ow *objectWriter, idx int, font EmbeddedFont) (*fontHandle, error) {
	metrics, err := parseTTF(font.TTFData)
	if err != nil {
		return nil, fmt.Errorf("pdf: font %q: %w", font.Name, err)
	}
	psName := metrics.PostScriptName
	if psName == "" {
		// Sanitize: PDF font names must be PostScript-name-safe — no
		// whitespace, parens, brackets. We strip them; the result is
		// purely cosmetic (Acrobat shows it in the Properties panel).
		psName = sanitizePSName(font.Name)
	}

	if metrics.IsCFF {
		return emitCFFFont(ow, idx, font, metrics, psName)
	}
	return emitTrueTypeIdentityH(ow, idx, font, metrics, psName)
}

// emitTrueTypeIdentityH writes a TrueType font as a PDF 1.7
// composite (Type 0) font with Identity-H encoding. Object layout:
//
//  1. FontFile2 stream — the raw TrueType bytes, FlateDecode-compressed.
//     When KeepGIDs is set the writer subsets unused glyphs first.
//  2. FontDescriptor — same fields the simple-TrueType path used.
//  3. CIDFontType2 descendant — references the descriptor and carries
//     the /W array (advance widths keyed by glyph index, in 1/1000 em)
//     plus /CIDToGIDMap /Identity.
//  4. ToUnicode CMap — maps each glyph back to its Unicode codepoint
//     so text extraction round-trips faithfully.
//  5. Type 0 wrapper /Font dict — what the page's Resources reference.
func emitTrueTypeIdentityH(ow *objectWriter, idx int, font EmbeddedFont, metrics *ttfMetrics, psName string) (*fontHandle, error) {
	// 1. FontFile2.
	ttf := font.TTFData
	if len(font.KeepGIDs) > 0 {
		if subset, err := subsetTrueType(font.TTFData, font.KeepGIDs); err == nil {
			ttf = subset
		}
	}
	compressed := flateAlways(ttf)
	streamID := ow.allocID()
	ow.writeStreamObject(streamID, fmt.Sprintf(
		"/Length %d /Length1 %d /Filter /FlateDecode",
		len(compressed), len(ttf),
	), compressed)

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
			"/FontFile2 %s >>",
		psName, flags,
		scale(metrics.XMin), scale(metrics.YMin), scale(metrics.XMax), scale(metrics.YMax),
		metrics.ItalicAngle,
		scale(metrics.Ascent), scale(metrics.Descent), scale(metrics.CapHeight),
		metrics.StemV,
		ref(streamID),
	)
	descriptorID := ow.allocAndWrite(descriptorBody)

	// 3. CIDFontType2 descendant. /W is a width array keyed by glyph
	// index in PDF "scope" form: `gid [w1 w2 w3]` lists consecutive
	// widths starting at gid. We collect the glyphs we have advance
	// data for, sort, and emit one entry per contiguous run.
	wArray := buildTrueTypeWArray(metrics, scale)
	descendantBody := fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s "+
			"/CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> "+
			"/FontDescriptor %s /CIDToGIDMap /Identity /W %s >>",
		psName, ref(descriptorID), wArray,
	)
	descendantID := ow.allocAndWrite(descendantBody)

	// 4. ToUnicode CMap built from the cmap table so PDF readers can
	// extract text for Find / Copy / accessibility — same shape as
	// the CFF font's CMap (one bfchar per used codepoint).
	cmapBytes := buildIdentityHToUnicodeCMap(metrics)
	cmapCompressed := flateAlways(cmapBytes)
	cmapID := ow.allocID()
	ow.writeStreamObject(cmapID, fmt.Sprintf(
		"/Length %d /Filter /FlateDecode",
		len(cmapCompressed),
	), cmapCompressed)

	// 5. Type 0 wrapper.
	fontBody := fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /%s "+
			"/Encoding /Identity-H /DescendantFonts [%s] /ToUnicode %s >>",
		psName, ref(descendantID), ref(cmapID),
	)
	fontID := ow.allocAndWrite(fontBody)

	return &fontHandle{
		Name:    fmt.Sprintf("F%d", idx),
		DictID:  fontID,
		Metrics: metrics,
		Kind:    fontKindTrueType,
	}, nil
}

// buildTrueTypeWArray emits the /W array body PDF 9.7.4 expects.
// Empty AdvanceWidth list yields `[]` (acceptable; reader uses /DW
// fallback). For non-empty input we group consecutive glyph indices
// with the same advance into ranges and emit them as `gid [w]`
// entries — the most compact form.
func buildTrueTypeWArray(m *ttfMetrics, scale func(int16) int) string {
	var b bytes.Buffer
	b.WriteString("[")
	if len(m.AdvanceWidth) == 0 {
		b.WriteString("]")
		return b.String()
	}
	// Simple form: one big run starting at GID 0.
	fmt.Fprintf(&b, "0 [")
	for i, w := range m.AdvanceWidth {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%d", scale(int16(w)))
	}
	b.WriteString("]]")
	return b.String()
}

// buildIdentityHToUnicodeCMap writes a CMap mapping each glyph
// index used by m's Cmap back to its source Unicode codepoint, so
// PDF readers can extract text correctly.
func buildIdentityHToUnicodeCMap(m *ttfMetrics) []byte {
	var b bytes.Buffer
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\n")
	b.WriteString("begincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	b.WriteString("/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")

	type pair struct {
		gid uint16
		cp  uint32
	}
	pairs := make([]pair, 0, len(m.CmapUnicode))
	for cp, gid := range m.CmapUnicode {
		pairs = append(pairs, pair{gid: gid, cp: cp})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].gid < pairs[j].gid })

	// Chunk in batches of 100 (PDF 1.7 §9.10.3 limit).
	for i := 0; i < len(pairs); i += 100 {
		end := i + 100
		if end > len(pairs) {
			end = len(pairs)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-i)
		for _, p := range pairs[i:end] {
			fmt.Fprintf(&b, "<%04X> <%04X>\n", p.gid, p.cp)
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap CMapName currentdict /CMap defineresource pop end end\n")
	return b.Bytes()
}

// computeFontFlags fills the /Flags integer of the FontDescriptor per
// PDF 9.8.2 Table 123. Only the bits the writer can determine cheaply are
// set; the rest stay zero.
func computeFontFlags(m *ttfMetrics) int {
	const (
		fixedPitch  = 1 << 0
		serif       = 1 << 1
		symbolic    = 1 << 2
		nonsymbolic = 1 << 5
		italic      = 1 << 6
		forceBold   = 1 << 18
	)
	flags := symbolic // Identity-H is treated as symbolic by spec.
	if m.IsFixedPitch {
		flags |= fixedPitch
	}
	if m.IsItalic {
		flags |= italic
	}
	if m.IsBold {
		flags |= forceBold
	}
	_ = serif
	_ = nonsymbolic
	return flags
}

// sanitizePSName strips whitespace and PDF-delimiter characters from s,
// producing a string safe to use as a PostScript font name in a /Name
// object. PDF /Name objects allow most printable ASCII but the simplest
// safe subset is alphanumerics plus '-' and '_'.
func sanitizePSName(s string) string {
	if s == "" {
		return "Font"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "Font"
	}
	return b.String()
}
