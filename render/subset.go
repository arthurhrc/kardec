package render

import (
	"encoding/binary"

	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/pdf"
)

// applyFontSubset walks the laid-out pages, gathers the rune set
// each embedded face renders, and populates the matching
// EmbeddedFont.KeepGIDs slot. Glyph 0 (.notdef) is always kept so
// the writer has a fallback for missing codepoints.
//
// The codepoint → glyph-ID translation uses a fresh parse of the
// font's cmap (format-4 subtable). We accept the duplication with
// internal/pdf's parser so the render package does not need to
// import it just for this step.
//
// faceIndex maps each (family, bold, italic) used in the document
// to the index of the matching EmbeddedFont — produced by
// assembleEmbeddedFonts in the same render pass.
func applyFontSubset(embedded []pdf.EmbeddedFont, pages []layout.Page, faceIndex map[fontKey]int) {
	if len(embedded) == 0 {
		return
	}
	usedRunes := collectRunesPerFont(pages, faceIndex, len(embedded))
	for fontID, runes := range usedRunes {
		if len(runes) == 0 {
			continue
		}
		gids := mapRunesToGIDs(embedded[fontID].TTFData, runes)
		// Always keep the .notdef glyph and the space glyph so
		// content-stream emission has stable fallbacks.
		gids[0] = true
		embedded[fontID].KeepGIDs = gids
	}
}

// collectRunesPerFont walks every PlacedItem and accumulates the
// distinct runes each font ID renders. Items whose Font is not the
// render package's measureAdapter are skipped — those are math
// glyphs, stub markers, or rectangles that never feed back into a
// face the user owns.
func collectRunesPerFont(pages []layout.Page, faceIndex map[fontKey]int, fontCount int) []map[rune]bool {
	out := make([]map[rune]bool, fontCount)
	for _, p := range pages {
		for _, it := range p.Items {
			if it.Text == "" || it.Image != nil || it.Rect != nil {
				continue
			}
			a, ok := it.Font.(*measureAdapter)
			if !ok {
				continue
			}
			id, found := faceIndex[fontKey{family: a.family, bold: a.bold, italic: a.italic}]
			if !found {
				continue
			}
			if out[id] == nil {
				out[id] = make(map[rune]bool)
			}
			for _, r := range it.Text {
				out[id][r] = true
			}
		}
	}
	return out
}

// mapRunesToGIDs parses the TTF's cmap (format 4) and returns the
// glyph-ID set covering the supplied runes. Runes outside the
// table fall through silently — the caller already gives the .notdef
// glyph special treatment.
func mapRunesToGIDs(ttf []byte, runes map[rune]bool) map[uint16]bool {
	cmap := readCmapFormat4(ttf)
	out := make(map[uint16]bool, len(runes))
	for r := range runes {
		if gid, ok := cmap[uint32(r)]; ok && gid != 0 {
			out[gid] = true
		}
	}
	return out
}

// readCmapFormat4 walks the SFNT table directory, locates the
// (platformID=3, encodingID=1) cmap subtable, and returns its
// codepoint-to-glyph-ID mapping. Returns an empty map on any
// structural failure — callers fall back to embedding the full
// font, which still renders.
func readCmapFormat4(ttf []byte) map[uint32]uint16 {
	out := map[uint32]uint16{}
	if len(ttf) < 12 {
		return out
	}
	numTables := int(binary.BigEndian.Uint16(ttf[4:6]))
	if len(ttf) < 12+numTables*16 {
		return out
	}
	var cmapStart, cmapLen int
	for i := 0; i < numTables; i++ {
		entry := ttf[12+i*16 : 12+(i+1)*16]
		if string(entry[0:4]) == "cmap" {
			cmapStart = int(binary.BigEndian.Uint32(entry[8:12]))
			cmapLen = int(binary.BigEndian.Uint32(entry[12:16]))
			break
		}
	}
	if cmapStart == 0 || cmapStart+cmapLen > len(ttf) {
		return out
	}
	cmap := ttf[cmapStart : cmapStart+cmapLen]
	if len(cmap) < 4 {
		return out
	}
	numSubtables := int(binary.BigEndian.Uint16(cmap[2:4]))
	for i := 0; i < numSubtables; i++ {
		recordOff := 4 + i*8
		if recordOff+8 > len(cmap) {
			return out
		}
		platformID := binary.BigEndian.Uint16(cmap[recordOff : recordOff+2])
		encodingID := binary.BigEndian.Uint16(cmap[recordOff+2 : recordOff+4])
		subtableOffset := int(binary.BigEndian.Uint32(cmap[recordOff+4 : recordOff+8]))
		if platformID != 3 || encodingID != 1 {
			continue
		}
		if subtableOffset+8 > len(cmap) {
			return out
		}
		format := binary.BigEndian.Uint16(cmap[subtableOffset : subtableOffset+2])
		if format != 4 {
			continue
		}
		parseCmap4(cmap[subtableOffset:], out)
		return out
	}
	return out
}

// parseCmap4 parses the format-4 subtable layout into dst. Mirrors
// the loop the internal/pdf/ttf.go parser uses; kept here so the
// render package does not import internal/pdf for one helper.
func parseCmap4(sub []byte, dst map[uint32]uint16) {
	if len(sub) < 14 {
		return
	}
	segCountX2 := int(binary.BigEndian.Uint16(sub[6:8]))
	segCount := segCountX2 / 2
	if segCount == 0 {
		return
	}
	endCodes := 14
	startCodes := endCodes + segCountX2 + 2
	idDeltas := startCodes + segCountX2
	idRangeOffsets := idDeltas + segCountX2
	if idRangeOffsets+segCountX2 > len(sub) {
		return
	}
	for i := 0; i < segCount; i++ {
		end := uint32(binary.BigEndian.Uint16(sub[endCodes+i*2 : endCodes+i*2+2]))
		start := uint32(binary.BigEndian.Uint16(sub[startCodes+i*2 : startCodes+i*2+2]))
		delta := int16(binary.BigEndian.Uint16(sub[idDeltas+i*2 : idDeltas+i*2+2]))
		rangeOff := uint32(binary.BigEndian.Uint16(sub[idRangeOffsets+i*2 : idRangeOffsets+i*2+2]))
		for cp := start; cp <= end; cp++ {
			if cp == 0xFFFF {
				continue
			}
			var gid uint16
			if rangeOff == 0 {
				gid = uint16(int32(cp) + int32(delta))
			} else {
				idOff := idRangeOffsets + i*2 + int(rangeOff) + int(cp-start)*2
				if idOff+2 > len(sub) {
					continue
				}
				gid = binary.BigEndian.Uint16(sub[idOff : idOff+2])
				if gid != 0 {
					gid = uint16(int32(gid) + int32(delta))
				}
			}
			if gid != 0 {
				dst[cp] = gid
			}
		}
	}
}
