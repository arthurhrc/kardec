package pdf

import (
	"encoding/binary"
	"errors"
)

// subsetTrueType returns a copy of ttfBytes whose `glyf` table has
// every glyph not in keepGIDs zeroed out. The structural tables
// (loca, hmtx, cmap, maxp.numGlyphs) are left untouched, so the
// glyph IDs callers reference stay valid; the savings appear once
// the file is FlateDecode-compressed inside the PDF FontFile2
// stream — long runs of zeros compress to nearly nothing.
//
// composite glyphs in keepGIDs are recursively expanded so their
// component glyphs survive the zero-out pass.
//
// Returns the original bytes unchanged when keepGIDs is nil — the
// renderer keeps the full font for documents that opted out of
// subsetting.
//
// Errors only on a malformed TTF (missing required tables, bad
// offsets). The caller falls back to the full font in that case.
func subsetTrueType(ttfBytes []byte, keepGIDs map[uint16]bool) ([]byte, error) {
	if keepGIDs == nil {
		return ttfBytes, nil
	}
	if len(ttfBytes) < 12 {
		return nil, errors.New("pdf/subset: file too short")
	}

	tables, err := readTableDirectory(ttfBytes)
	if err != nil {
		return nil, err
	}

	headStart, headEnd, ok := tables.span("head")
	if !ok || headEnd-headStart < 54 {
		return nil, errors.New("pdf/subset: missing or short head table")
	}
	maxpStart, _, ok := tables.span("maxp")
	if !ok {
		return nil, errors.New("pdf/subset: missing maxp")
	}
	locaStart, locaEnd, ok := tables.span("loca")
	if !ok {
		return nil, errors.New("pdf/subset: missing loca")
	}
	glyfStart, glyfEnd, ok := tables.span("glyf")
	if !ok {
		return nil, errors.New("pdf/subset: missing glyf")
	}

	indexToLocFormat := binary.BigEndian.Uint16(ttfBytes[headStart+50 : headStart+52])
	numGlyphs := int(binary.BigEndian.Uint16(ttfBytes[maxpStart+4 : maxpStart+6]))

	offsets, err := readLocaOffsets(ttfBytes[locaStart:locaEnd], numGlyphs, indexToLocFormat)
	if err != nil {
		return nil, err
	}

	expanded := expandKeepWithComposites(keepGIDs, offsets, ttfBytes, glyfStart, glyfEnd)

	out := make([]byte, len(ttfBytes))
	copy(out, ttfBytes)
	zeroUnusedGlyphs(out, glyfStart, offsets, expanded, numGlyphs)
	rewriteHeadChecksum(out, headStart)
	return out, nil
}

// tableSet maps SFNT tag → (start, end) byte ranges in the input.
// span unpacks the [2]int range into two named ints plus a presence
// flag, which the caller would otherwise need to splat manually.
type tableSet map[string][2]int

func (s tableSet) span(tag string) (start, end int, ok bool) {
	r, found := s[tag]
	return r[0], r[1], found
}

// readTableDirectory parses the SFNT table directory and returns a
// tableSet keyed by tag.
func readTableDirectory(data []byte) (tableSet, error) {
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 12+numTables*16 {
		return nil, errors.New("pdf/subset: truncated table directory")
	}
	out := make(tableSet, numTables)
	for i := 0; i < numTables; i++ {
		entry := data[12+i*16 : 12+(i+1)*16]
		tag := string(entry[0:4])
		off := int(binary.BigEndian.Uint32(entry[8:12]))
		ln := int(binary.BigEndian.Uint32(entry[12:16]))
		if off+ln > len(data) {
			return nil, errors.New("pdf/subset: table overflows file")
		}
		out[tag] = [2]int{off, off + ln}
	}
	return out, nil
}

// readLocaOffsets returns the per-glyph offsets within glyf, plus
// one trailing offset that marks the end of the last glyph. Length
// of the returned slice is numGlyphs+1.
func readLocaOffsets(loca []byte, numGlyphs int, indexToLocFormat uint16) ([]int, error) {
	out := make([]int, numGlyphs+1)
	if indexToLocFormat == 0 { // short — uint16 offset / 2
		if len(loca) < 2*(numGlyphs+1) {
			return nil, errors.New("pdf/subset: short loca truncated")
		}
		for i := 0; i <= numGlyphs; i++ {
			out[i] = int(binary.BigEndian.Uint16(loca[i*2:i*2+2])) * 2
		}
	} else { // long — uint32 raw offset
		if len(loca) < 4*(numGlyphs+1) {
			return nil, errors.New("pdf/subset: long loca truncated")
		}
		for i := 0; i <= numGlyphs; i++ {
			out[i] = int(binary.BigEndian.Uint32(loca[i*4 : i*4+4]))
		}
	}
	return out, nil
}

// expandKeepWithComposites walks the kept glyph IDs and adds the
// component glyph IDs of any composite glyph encountered, repeating
// until no more components are discovered. Components of components
// are picked up by the outer fixed-point loop.
func expandKeepWithComposites(initial map[uint16]bool, offsets []int, data []byte, glyfStart, glyfEnd int) map[uint16]bool {
	out := make(map[uint16]bool, len(initial))
	for k := range initial {
		out[k] = true
	}
	frontier := make([]uint16, 0, len(initial))
	for k := range initial {
		frontier = append(frontier, k)
	}
	for len(frontier) > 0 {
		gid := frontier[0]
		frontier = frontier[1:]
		if int(gid)+1 >= len(offsets) {
			continue
		}
		start := glyfStart + offsets[gid]
		end := glyfStart + offsets[gid+1]
		if start == end || end > glyfEnd {
			continue
		}
		// numberOfContours int16; negative means composite.
		numContours := int16(binary.BigEndian.Uint16(data[start : start+2]))
		if numContours >= 0 {
			continue
		}
		components := readCompositeComponents(data, start+10, end)
		for _, comp := range components {
			if !out[comp] {
				out[comp] = true
				frontier = append(frontier, comp)
			}
		}
	}
	return out
}

// readCompositeComponents walks the component-record list of a
// composite glyph entry, returning the referenced glyph IDs. The
// flags layout follows the OpenType "Composite Glyph Description"
// spec — only the subset we need is handled here:
//
//	bit 0  ARG_1_AND_2_ARE_WORDS
//	bit 3  WE_HAVE_A_SCALE
//	bit 5  MORE_COMPONENTS
//	bit 6  WE_HAVE_AN_X_AND_Y_SCALE
//	bit 7  WE_HAVE_A_TWO_BY_TWO
//
// Anything we don't model (instructions, overlap flags) doesn't
// affect glyph-ID enumeration.
func readCompositeComponents(data []byte, start, end int) []uint16 {
	var out []uint16
	cursor := start
	for cursor+4 <= end {
		flags := binary.BigEndian.Uint16(data[cursor : cursor+2])
		gid := binary.BigEndian.Uint16(data[cursor+2 : cursor+4])
		out = append(out, gid)
		cursor += 4
		if flags&0x0001 != 0 {
			cursor += 4 // arg1/arg2 as int16
		} else {
			cursor += 2 // arg1/arg2 as int8
		}
		switch {
		case flags&0x0080 != 0: // 2x2 matrix — 4 F2DOT14
			cursor += 8
		case flags&0x0040 != 0: // x and y scale — 2 F2DOT14
			cursor += 4
		case flags&0x0008 != 0: // single scale — 1 F2DOT14
			cursor += 2
		}
		if flags&0x0020 == 0 { // MORE_COMPONENTS clear → done
			break
		}
	}
	return out
}

// zeroUnusedGlyphs walks every glyph and zeroes the bytes in glyf
// for any glyph not in keep. Glyphs with zero length (already
// empty) are skipped silently.
func zeroUnusedGlyphs(data []byte, glyfStart int, offsets []int, keep map[uint16]bool, numGlyphs int) {
	for gid := 0; gid < numGlyphs; gid++ {
		if keep[uint16(gid)] {
			continue
		}
		start := glyfStart + offsets[gid]
		end := glyfStart + offsets[gid+1]
		if start >= end {
			continue
		}
		for i := start; i < end; i++ {
			data[i] = 0
		}
	}
}

// rewriteHeadChecksum recomputes the file-wide checksum stored at
// head[8:12] (the checksumAdjustment field). The value is the
// difference between 0xB1B0AFBA and the running sum of the file
// (with the field treated as zero during the calculation).
func rewriteHeadChecksum(data []byte, headStart int) {
	if headStart+12 > len(data) {
		return
	}
	// Zero the field for the calculation.
	binary.BigEndian.PutUint32(data[headStart+8:headStart+12], 0)
	var sum uint32
	for i := 0; i+4 <= len(data); i += 4 {
		sum += binary.BigEndian.Uint32(data[i : i+4])
	}
	// Tail bytes (file length not multiple of 4) — pad with zero.
	if rem := len(data) % 4; rem != 0 {
		var tail [4]byte
		copy(tail[:], data[len(data)-rem:])
		sum += binary.BigEndian.Uint32(tail[:])
	}
	adjust := uint32(0xB1B0AFBA) - sum
	binary.BigEndian.PutUint32(data[headStart+8:headStart+12], adjust)
}
