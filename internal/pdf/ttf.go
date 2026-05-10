package pdf

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ttfMetrics is the minimal slice of TrueType font metadata the writer
// needs to assemble a /FontDescriptor and a simple-font /Widths array.
//
// All values are in font design units (1/UnitsPerEm). The Writer scales
// them to 1/1000 em (the unit PDF uses for /Widths and the FontDescriptor)
// at emission time.
type ttfMetrics struct {
	UnitsPerEm uint16
	XMin       int16
	YMin       int16
	XMax       int16
	YMax       int16

	Ascent    int16 // OS/2 sTypoAscender (preferred) or hhea ascent
	Descent   int16 // OS/2 sTypoDescender (preferred) or hhea descent
	CapHeight int16 // OS/2 sCapHeight (0 if absent)

	ItalicAngle  float64 // post.italicAngle, fixed16.16
	StemV        int16   // not in TTF; the writer fills a heuristic
	IsItalic     bool
	IsBold       bool
	IsFixedPitch bool

	// AdvanceWidth maps glyph index -> advance width in design units.
	AdvanceWidth []uint16

	// CmapUnicode maps Unicode codepoint -> glyph index, sourced from a
	// format 4 cmap subtable (platformID=3, encodingID=1) if present.
	// Used to look up widths for WinAnsiEncoding bytes.
	CmapUnicode map[uint32]uint16

	// PostScriptName from the 'name' table (nameID=6); falls back to a
	// sanitized version of EmbeddedFont.Name when absent.
	PostScriptName string

	// IsCFF reports that this font uses CFF outlines (the 'OTTO'
	// SFNT scaler). The emit path uses Type 0 + CIDFontType0 +
	// FontFile3 / Subtype /CIDFontType0C for these instead of the
	// simple TrueType path.
	IsCFF bool

	// CFFData carries the raw bytes of the 'CFF ' table when IsCFF
	// is true. Empty for TrueType fonts. The writer streams these
	// straight into the FontFile3 stream.
	CFFData []byte
}

// parseTTF extracts ttfMetrics from a TrueType / OpenType file. The
// implementation tolerates unknown tables and returns whatever it
// managed to read, defaulting missing fields to safe values — consumer
// code (the writer) survives a partial parse with a less-accurate but
// still-rendering font.
//
// The parser accepts three SFNT scalers:
//   - 0x00010000: classic TrueType (glyf outlines)
//   - 'true':     legacy Apple TrueType
//   - 'OTTO':     OpenType with CFF outlines — populates m.CFFData
//                 from the 'CFF ' table so the writer can embed via
//                 FontFile3 / Subtype /CIDFontType0C.
//
// m.IsCFF reports whether the scaler indicated CFF outlines, so the
// emit path can choose Type 0 + CIDFontType0 instead of simple
// /Subtype /TrueType.
func parseTTF(data []byte) (*ttfMetrics, error) {
	if len(data) < 12 {
		return nil, errors.New("pdf/ttf: file too short for offset table")
	}
	scaler := binary.BigEndian.Uint32(data[0:4])
	const (
		scalerTrueType  = 0x00010000
		scalerTrueLegacy = 0x74727565 // 'true'
		scalerOTTO       = 0x4F54544F // 'OTTO'
	)
	if scaler != scalerTrueType && scaler != scalerTrueLegacy && scaler != scalerOTTO {
		return nil, fmt.Errorf("pdf/ttf: unsupported scaler 0x%08x", scaler)
	}
	isCFF := scaler == scalerOTTO
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 12+numTables*16 {
		return nil, errors.New("pdf/ttf: truncated table directory")
	}

	tables := make(map[string][]byte, numTables)
	for i := 0; i < numTables; i++ {
		entry := data[12+i*16 : 12+(i+1)*16]
		tag := string(entry[0:4])
		off := binary.BigEndian.Uint32(entry[8:12])
		ln := binary.BigEndian.Uint32(entry[12:16])
		if uint64(off)+uint64(ln) > uint64(len(data)) {
			return nil, fmt.Errorf("pdf/ttf: table %q overflows file", tag)
		}
		tables[tag] = data[off : off+ln]
	}

	m := &ttfMetrics{
		StemV:       80, // generic upright sans heuristic
		CmapUnicode: make(map[uint32]uint16),
		IsCFF:       isCFF,
	}
	if isCFF {
		if cff, ok := tables["CFF "]; ok {
			m.CFFData = cff
		}
	}

	if head, ok := tables["head"]; ok && len(head) >= 54 {
		m.UnitsPerEm = binary.BigEndian.Uint16(head[18:20])
		m.XMin = int16(binary.BigEndian.Uint16(head[36:38]))
		m.YMin = int16(binary.BigEndian.Uint16(head[38:40]))
		m.XMax = int16(binary.BigEndian.Uint16(head[40:42]))
		m.YMax = int16(binary.BigEndian.Uint16(head[42:44]))
		macStyle := binary.BigEndian.Uint16(head[44:46])
		m.IsBold = macStyle&0x01 != 0
		m.IsItalic = macStyle&0x02 != 0
	}
	if m.UnitsPerEm == 0 {
		m.UnitsPerEm = 1000
	}

	var hheaAscent, hheaDescent int16
	var numHMetrics uint16
	if hhea, ok := tables["hhea"]; ok && len(hhea) >= 36 {
		hheaAscent = int16(binary.BigEndian.Uint16(hhea[4:6]))
		hheaDescent = int16(binary.BigEndian.Uint16(hhea[6:8]))
		numHMetrics = binary.BigEndian.Uint16(hhea[34:36])
	}
	m.Ascent = hheaAscent
	m.Descent = hheaDescent

	// OS/2 takes precedence for typographic ascent/descent on Windows;
	// some TTFs disagree between hhea and OS/2 and Acrobat trusts OS/2.
	if os2, ok := tables["OS/2"]; ok && len(os2) >= 78 {
		m.Ascent = int16(binary.BigEndian.Uint16(os2[68:70]))  // sTypoAscender
		m.Descent = int16(binary.BigEndian.Uint16(os2[70:72])) // sTypoDescender
		if len(os2) >= 90 {                                    // version >= 2 carries sCapHeight at offset 88
			m.CapHeight = int16(binary.BigEndian.Uint16(os2[88:90]))
		}
		fsSelection := binary.BigEndian.Uint16(os2[62:64])
		if fsSelection&0x01 != 0 {
			m.IsItalic = true
		}
		if fsSelection&0x20 != 0 {
			m.IsBold = true
		}
	}
	if m.CapHeight == 0 {
		// Common fallback when OS/2 is < v2.
		m.CapHeight = m.Ascent
	}

	if post, ok := tables["post"]; ok && len(post) >= 32 {
		// italicAngle is a Fixed16.16 at offset 4.
		raw := int32(binary.BigEndian.Uint32(post[4:8]))
		m.ItalicAngle = float64(raw) / 65536.0
		m.IsFixedPitch = binary.BigEndian.Uint32(post[12:16]) != 0
	}

	if hmtx, ok := tables["hmtx"]; ok && numHMetrics > 0 {
		// hmtx is numHMetrics longHorMetric records (4 bytes each: advance,
		// lsb), followed by leftSideBearing-only entries for trailing
		// glyphs sharing the last advance width. The writer only needs
		// advance widths.
		need := int(numHMetrics) * 4
		if len(hmtx) >= need {
			m.AdvanceWidth = make([]uint16, numHMetrics)
			for i := 0; i < int(numHMetrics); i++ {
				m.AdvanceWidth[i] = binary.BigEndian.Uint16(hmtx[i*4 : i*4+2])
			}
		}
	}

	if cmap, ok := tables["cmap"]; ok {
		parseCmapFormat4(cmap, m.CmapUnicode)
	}

	if name, ok := tables["name"]; ok {
		m.PostScriptName = parsePostScriptName(name)
	}

	return m, nil
}

// parseCmapFormat4 walks the cmap header looking for a Windows Unicode BMP
// (platform 3 / encoding 1) format-4 subtable, which is universally
// present in TTFs distributed for Windows. The result populates dst with
// codepoint -> glyph index mappings for the BMP only — characters outside
// U+0000..U+FFFF are not used by WinAnsiEncoding so the omission is safe
// for v0.1.
func parseCmapFormat4(data []byte, dst map[uint32]uint16) {
	if len(data) < 4 {
		return
	}
	numTables := int(binary.BigEndian.Uint16(data[2:4]))
	if len(data) < 4+numTables*8 {
		return
	}
	var subtableOff uint32
	for i := 0; i < numTables; i++ {
		entry := data[4+i*8 : 4+(i+1)*8]
		platformID := binary.BigEndian.Uint16(entry[0:2])
		encodingID := binary.BigEndian.Uint16(entry[2:4])
		off := binary.BigEndian.Uint32(entry[4:8])
		if platformID == 3 && encodingID == 1 {
			subtableOff = off
			break
		}
	}
	if subtableOff == 0 || uint64(subtableOff)+14 > uint64(len(data)) {
		return
	}
	sub := data[subtableOff:]
	if binary.BigEndian.Uint16(sub[0:2]) != 4 {
		return // only format 4 is supported in v0.1
	}
	segCountX2 := int(binary.BigEndian.Uint16(sub[6:8]))
	segCount := segCountX2 / 2
	headerSize := 14
	endOff := headerSize
	startOff := endOff + segCountX2 + 2 // +2 reservedPad
	idDeltaOff := startOff + segCountX2
	idRangeOff := idDeltaOff + segCountX2
	if len(sub) < idRangeOff+segCountX2 {
		return
	}
	for s := 0; s < segCount; s++ {
		end := binary.BigEndian.Uint16(sub[endOff+s*2 : endOff+s*2+2])
		start := binary.BigEndian.Uint16(sub[startOff+s*2 : startOff+s*2+2])
		idDelta := int16(binary.BigEndian.Uint16(sub[idDeltaOff+s*2 : idDeltaOff+s*2+2]))
		idRangeOffsetField := idRangeOff + s*2
		idRangeOffset := binary.BigEndian.Uint16(sub[idRangeOffsetField : idRangeOffsetField+2])
		for c := uint32(start); c <= uint32(end); c++ {
			if c == 0xFFFF {
				continue
			}
			var glyph uint16
			if idRangeOffset == 0 {
				glyph = uint16(int32(c) + int32(idDelta))
			} else {
				// The TrueType-spec described dance: the glyphIdArray
				// pointer is computed relative to the idRangeOffset
				// location itself.
				glyphIDPos := int(idRangeOffset) + 2*int(c-uint32(start)) + idRangeOffsetField
				if glyphIDPos+2 > len(sub) {
					continue
				}
				g := binary.BigEndian.Uint16(sub[glyphIDPos : glyphIDPos+2])
				if g != 0 {
					glyph = uint16(int32(g) + int32(idDelta))
				}
			}
			if glyph != 0 {
				dst[c] = glyph
			}
		}
	}
}

// parsePostScriptName extracts the PostScript-name string (nameID=6) from
// the 'name' table. Prefers the Windows Unicode BMP record (platform 3,
// encoding 1, language 0x0409); falls back to Macintosh Roman (platform 1,
// encoding 0). Returns "" if neither is present.
func parsePostScriptName(data []byte) string {
	if len(data) < 6 {
		return ""
	}
	count := int(binary.BigEndian.Uint16(data[2:4]))
	storageOff := int(binary.BigEndian.Uint16(data[4:6]))
	recordsStart := 6
	if len(data) < recordsStart+count*12 {
		return ""
	}
	type cand struct {
		platform, encoding, length, offset int
	}
	var win, mac *cand
	for i := 0; i < count; i++ {
		rec := data[recordsStart+i*12 : recordsStart+(i+1)*12]
		platformID := int(binary.BigEndian.Uint16(rec[0:2]))
		encodingID := int(binary.BigEndian.Uint16(rec[2:4]))
		nameID := int(binary.BigEndian.Uint16(rec[6:8]))
		length := int(binary.BigEndian.Uint16(rec[8:10]))
		offset := int(binary.BigEndian.Uint16(rec[10:12]))
		if nameID != 6 {
			continue
		}
		c := &cand{platformID, encodingID, length, offset}
		if platformID == 3 && encodingID == 1 && win == nil {
			win = c
		} else if platformID == 1 && encodingID == 0 && mac == nil {
			mac = c
		}
	}
	pick := win
	if pick == nil {
		pick = mac
	}
	if pick == nil {
		return ""
	}
	start := storageOff + pick.offset
	end := start + pick.length
	if end > len(data) {
		return ""
	}
	raw := data[start:end]
	if pick.platform == 3 {
		// UTF-16BE -> ASCII (PostScript names are ASCII per spec).
		var buf []byte
		for i := 0; i+1 < len(raw); i += 2 {
			lo := raw[i+1]
			if raw[i] == 0 && lo >= 32 && lo < 127 {
				buf = append(buf, lo)
			}
		}
		return string(buf)
	}
	return string(raw)
}
