package pdf

import (
	"bytes"
	"fmt"
)

// buildContentStream renders a Page's text items and image draws into
// the byte payload of a PDF content stream (PDF 7.8.2). The returned
// bytes are the raw operators; the caller wraps them with the stream
// dictionary and the stream/endstream markers via writeStreamObject.
//
// Text op sequence per item:
//
//	q                       % save graphics state
//	r g b rg                % nonstroking RGB fill color
//	BT                      % begin text object
//	/Fn size Tf             % select font + size
//	x y Td                  % set text-matrix translation
//	(escaped string) Tj     % show string
//	ET                      % end text object
//	Q                       % restore graphics state
//
// Image op sequence per draw:
//
//	q                       % save graphics state
//	W 0 0 H X Y cm          % scale + translate matrix (W/H = width/height pt)
//	/Im0 Do                 % paint the named XObject
//	Q
//
// Rect op sequence per draw:
//
//	q                       % save graphics state
//	r g b rg                % nonstroking RGB fill color
//	X Y W H re              % construct rectangle path
//	f                       % fill
//	Q
//
// Rects are drawn before images and images before text so glyphs
// overlap rules / images cleanly — the PDF renderer paints in op order.
func buildContentStream(page Page, fonts []*fontHandle, images []*imageHandle) []byte {
	var buf bytes.Buffer

	for _, r := range page.Rects {
		fmt.Fprintf(&buf,
			"q\n%.4f %.4f %.4f rg\n%.4f %.4f %.4f %.4f re\nf\nQ\n",
			float64(r.Color.R)/255.0,
			float64(r.Color.G)/255.0,
			float64(r.Color.B)/255.0,
			r.X, r.Y, r.W, r.H,
		)
	}

	for _, im := range page.Images {
		if im.ImageID < 0 || im.ImageID >= len(images) {
			continue
		}
		ih := images[im.ImageID]
		fmt.Fprintf(&buf,
			"q\n%.4f 0 0 %.4f %.4f %.4f cm\n/%s Do\nQ\n",
			im.W, im.H, im.X, im.Y, ih.Name,
		)
	}

	for _, it := range page.Items {
		if it.FontID < 0 || it.FontID >= len(fonts) {
			continue // skip silently rather than panicking on bad input
		}
		fh := fonts[it.FontID]
		r := float64(it.Color.R) / 255.0
		g := float64(it.Color.G) / 255.0
		b := float64(it.Color.B) / 255.0

		// Encoding is font-specific: simple TrueType uses single-byte
		// WinAnsi escaped literals, Type 0 / CIDFontType0 (CFF math
		// fonts) uses 2-byte hex glyph IDs in angle brackets.
		var operand string
		switch fh.Kind {
		case fontKindCFF:
			operand = encodeCFFHex([]rune(it.Text), fh.Metrics)
		default:
			encoded := encodeWinAnsi([]rune(it.Text))
			operand = escapeLiteralString(string(encoded))
		}

		fmt.Fprintf(&buf,
			"q\n%.4f %.4f %.4f rg\nBT\n/%s %.4f Tf\n%.4f %.4f Td\n%s Tj\nET\nQ\n",
			r, g, b,
			fh.Name, it.FontSize,
			it.X, it.Y,
			operand,
		)
	}
	return buf.Bytes()
}

// buildContentStreamWithWatermark builds the page content stream and
// appends a watermark after the primary content. fonts indexes by
// FontID; the watermark's FontID picks its rendered face. alphaName
// is the resource name of the page's /ExtGState alpha entry (empty
// when opacity is 1.0 or absent).
func buildContentStreamWithWatermark(page Page, fonts []*fontHandle, images []*imageHandle, watermark *Watermark, alphaName string) []byte {
	raw := buildContentStream(page, fonts, images)
	if watermark == nil || watermark.Text == "" {
		return raw
	}
	if watermark.FontID < 0 || watermark.FontID >= len(fonts) {
		return raw
	}
	var buf bytes.Buffer
	buf.Write(raw)
	appendWatermark(&buf, watermark, fonts[watermark.FontID], page.Width, page.Height, alphaName)
	return buf.Bytes()
}

// encodeCFFHex turns a slice of Unicode runes into the
// `<00410042...>` hex string a Type 0 / Identity-H content stream
// expects: each glyph index is two bytes, big-endian. Glyph indices
// come from the cmap parsed at font-load time; runes the cmap
// doesn't cover collapse to glyph 0 (.notdef), the conventional
// missing-glyph slot.
func encodeCFFHex(runes []rune, m *ttfMetrics) string {
	var b []byte
	b = append(b, '<')
	for _, r := range runes {
		gid := uint16(0)
		if g, ok := m.CmapUnicode[uint32(r)]; ok {
			gid = g
		}
		b = append(b,
			hexNibble(byte(gid>>12)),
			hexNibble(byte((gid>>8)&0x0F)),
			hexNibble(byte((gid>>4)&0x0F)),
			hexNibble(byte(gid&0x0F)),
		)
	}
	b = append(b, '>')
	return string(b)
}

func hexNibble(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}

// resourcesDict returns the /Resources dict body for a page. Fonts go
// under /Font; images go under /XObject. /ProcSet stays for legacy
// Acrobat compatibility and grows /ImageC when images are present.
//
// extGStateName / extGStateID populate the page's /ExtGState dict
// when the watermark needs alpha blending. Empty extGStateName
// means "no /ExtGState entry" — keeps non-watermarked output
// byte-identical to v0.19.
func resourcesDict(fonts []*fontHandle, images []*imageHandle, extGStateName string, extGStateID int) string {
	var buf bytes.Buffer
	buf.WriteString("<< /ProcSet [/PDF /Text")
	if len(images) > 0 {
		buf.WriteString(" /ImageC")
	}
	buf.WriteString("] /Font <<")
	for _, fh := range fonts {
		fmt.Fprintf(&buf, " /%s %s", fh.Name, ref(fh.DictID))
	}
	buf.WriteString(" >>")
	if len(images) > 0 {
		buf.WriteString(" /XObject <<")
		for _, ih := range images {
			fmt.Fprintf(&buf, " /%s %s", ih.Name, ref(ih.ID))
		}
		buf.WriteString(" >>")
	}
	if extGStateName != "" && extGStateID > 0 {
		fmt.Fprintf(&buf, " /ExtGState << /%s %s >>", extGStateName, ref(extGStateID))
	}
	buf.WriteString(" >>")
	return buf.String()
}
