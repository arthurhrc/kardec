package pdf

import (
	"fmt"
	"strings"
)

// fontHandle is the writer's record of an embedded font: the indirect ID
// of its /Font dictionary (referenced from /Resources), its parsed
// metrics, and its PDF resource name (e.g. "F0", "F1", ...).
type fontHandle struct {
	Name    string // resource name used inside content streams ("F0")
	DictID  int    // indirect ID of the /Font dictionary
	Metrics *ttfMetrics
}

// emitFont writes three indirect objects for a TrueType font:
//
//  1. the FontFile2 stream (raw TTF bytes, FlateDecode-compressed)
//  2. the FontDescriptor dictionary
//  3. the /Font dictionary (Subtype /TrueType, WinAnsiEncoding)
//
// It returns a fontHandle the Writer attaches to the page Resources dict.
//
// Limitation (v0.1, documented in package doc): WinAnsiEncoding caps the
// representable character set at CP1252. Glyphs outside that range are
// substituted with '?' at content-stream emission time. Migration to a
// composite (Type 0 / CIDFontType2 with Identity-H) font is the v0.2 work
// item; the writer's public API does not change when that lands.
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

	// 1. FontFile2 stream — embed full TTF (no subsetting in v0.1).
	ttf := font.TTFData
	compressed := flateAlways(ttf)
	streamID := ow.allocID()
	streamDict := fmt.Sprintf(
		"/Length %d /Length1 %d /Filter /FlateDecode",
		len(compressed), len(ttf),
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
			"/FontFile2 %s >>",
		psName, flags,
		scale(metrics.XMin), scale(metrics.YMin), scale(metrics.XMax), scale(metrics.YMax),
		metrics.ItalicAngle,
		scale(metrics.Ascent), scale(metrics.Descent), scale(metrics.CapHeight),
		metrics.StemV,
		ref(streamID),
	)
	descriptorID := ow.allocAndWrite(descriptorBody)

	// 3. /Widths array. Simple fonts use FirstChar..LastChar inclusive;
	// each entry is the advance width of the glyph that WinAnsi byte maps
	// to, expressed in 1/1000 em. Bytes that don't map to any glyph get
	// width 0.
	firstChar, lastChar := 32, 255
	widths := make([]int, 0, lastChar-firstChar+1)
	for b := firstChar; b <= lastChar; b++ {
		cp := winAnsiToUnicode[b]
		var w int
		if cp != 0 {
			if g, ok := metrics.CmapUnicode[cp]; ok && int(g) < len(metrics.AdvanceWidth) {
				w = scale(int16(metrics.AdvanceWidth[g]))
			} else if len(metrics.AdvanceWidth) > 0 {
				// Fall back to glyph 0's advance — the spec requires a
				// width entry even for absent glyphs, and 0 is acceptable
				// but uglier in viewers that draw the missing-glyph box.
				w = scale(int16(metrics.AdvanceWidth[0]))
			}
		}
		widths = append(widths, w)
	}

	var sb strings.Builder
	sb.WriteString("[")
	for i, w := range widths {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%d", w)
	}
	sb.WriteString("]")

	// 4. /Font dictionary.
	fontBody := fmt.Sprintf(
		"<< /Type /Font /Subtype /TrueType /BaseFont /%s "+
			"/FirstChar %d /LastChar %d /Widths %s "+
			"/FontDescriptor %s /Encoding /WinAnsiEncoding >>",
		psName, firstChar, lastChar, sb.String(), ref(descriptorID),
	)
	fontID := ow.allocAndWrite(fontBody)

	return &fontHandle{
		Name:    fmt.Sprintf("F%d", idx),
		DictID:  fontID,
		Metrics: metrics,
	}, nil
}

// computeFontFlags fills the /Flags integer of the FontDescriptor per
// PDF 9.8.2 Table 123. Only the bits the writer can determine cheaply are
// set; the rest stay zero.
func computeFontFlags(m *ttfMetrics) int {
	const (
		fixedPitch = 1 << 0
		serif      = 1 << 1
		symbolic   = 1 << 2
		nonsymbolic = 1 << 5
		italic     = 1 << 6
		forceBold  = 1 << 18
	)
	flags := nonsymbolic // WinAnsiEncoding implies the font is treated as nonsymbolic.
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
	_ = symbolic
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
