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
		// Image XObjects (raster) live in the unit square [0,1]×[0,1],
		// so a cm of [W 0 0 H X Y] scales them straight to W×H points.
		// Form XObjects (SVG) live in their declared /BBox; the
		// caller-requested W×H must therefore be divided by the BBox
		// dimensions before being baked into the matrix, otherwise the
		// drawing renders at BBox×W, BBox×H — which trivially flies
		// off-page for a 60-pt SVG asked to draw at 60 pt (was the
		// silent-blank-SVG bug through v0.21).
		sx, sy := im.W, im.H
		if ih.IsForm && ih.Width > 0 && ih.Height > 0 {
			sx = im.W / float64(ih.Width)
			sy = im.H / float64(ih.Height)
		}
		fmt.Fprintf(&buf,
			"q\n%.4f 0 0 %.4f %.4f %.4f cm\n/%s Do\nQ\n",
			sx, sy, im.X, im.Y, ih.Name,
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

		// Both TrueType and CFF embed paths produce Type 0 /
		// Identity-H composite fonts (post-v0.22), so the content
		// stream emits two-byte hex glyph IDs uniformly.
		operand := encodeCFFHex([]rune(it.Text), fh.Metrics)

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

// wrapMarkedContentByBlocks emits the page's draw operators
// partitioned into per-leaf marked-content sequences. Each leaf
// block's items + images are wrapped in `/<role> << /MCID N >>
// BDC ... EMC`. Inner blocks (Table, TR, Sect, …) carry no MCIDs
// — only their leaf descendants do.
//
// MCIDs are assigned in pre-order traversal of the tree, matching
// the order emitStructTree emits StructElem objects.
func wrapMarkedContentByBlocks(page Page, fonts []*fontHandle, images []*imageHandle, watermark *Watermark, alphaName string) []byte {
	var buf bytes.Buffer
	// Rects (no PDF/UA role, just visual chrome) come first.
	for _, r := range page.Rects {
		fmt.Fprintf(&buf,
			"q\n%.4f %.4f %.4f rg\n%.4f %.4f %.4f %.4f re\nf\nQ\n",
			float64(r.Color.R)/255.0,
			float64(r.Color.G)/255.0,
			float64(r.Color.B)/255.0,
			r.X, r.Y, r.W, r.H,
		)
	}
	// Recurse the block tree, emitting BDC/EMC per leaf with a
	// running MCID counter.
	mcid := 0
	for _, block := range page.StructBlocks {
		mcid = emitBlockTree(&buf, block, fonts, images, page, mcid)
	}
	// Watermark sits outside any block — purely decorative.
	if watermark != nil && watermark.Text != "" && watermark.FontID >= 0 && watermark.FontID < len(fonts) {
		appendWatermark(&buf, watermark, fonts[watermark.FontID], page.Width, page.Height, alphaName)
	}
	return buf.Bytes()
}

// emitBlockTree recurses through one StructBlock's tree, emitting
// BDC/EMC marked-content pairs around each LEAF block's draw
// operators. Inner blocks only frame their children — no MCID is
// assigned to them. Returns the next free MCID (so siblings keep
// counting up).
func emitBlockTree(buf *bytes.Buffer, b StructBlock, fonts []*fontHandle, images []*imageHandle, page Page, mcid int) int {
	if b.IsLeaf() {
		fmt.Fprintf(buf, "/%s << /MCID %d >> BDC\n", b.Role, mcid)
		for i := b.ImageStart; i < b.ImageEnd && i < len(page.Images); i++ {
			emitImageDraw(buf, page.Images[i], images)
		}
		for i := b.ItemStart; i < b.ItemEnd && i < len(page.Items); i++ {
			emitTextItem(buf, page.Items[i], fonts)
		}
		buf.WriteString("EMC\n")
		return mcid + 1
	}
	for i := range b.Children {
		mcid = emitBlockTree(buf, b.Children[i], fonts, images, page, mcid)
	}
	return mcid
}

// emitImageDraw writes the operators for one ImageDraw — extracted
// from buildContentStream so the per-block path can call it without
// duplicating the cm-matrix logic.
func emitImageDraw(buf *bytes.Buffer, im ImageDraw, images []*imageHandle) {
	if im.ImageID < 0 || im.ImageID >= len(images) {
		return
	}
	ih := images[im.ImageID]
	sx, sy := im.W, im.H
	if ih.IsForm && ih.Width > 0 && ih.Height > 0 {
		sx = im.W / float64(ih.Width)
		sy = im.H / float64(ih.Height)
	}
	fmt.Fprintf(buf,
		"q\n%.4f 0 0 %.4f %.4f %.4f cm\n/%s Do\nQ\n",
		sx, sy, im.X, im.Y, ih.Name,
	)
}

// emitTextItem writes the operators for one TextItem — also
// extracted for the per-block path.
func emitTextItem(buf *bytes.Buffer, it TextItem, fonts []*fontHandle) {
	if it.FontID < 0 || it.FontID >= len(fonts) {
		return
	}
	fh := fonts[it.FontID]
	r := float64(it.Color.R) / 255.0
	g := float64(it.Color.G) / 255.0
	b := float64(it.Color.B) / 255.0
	operand := encodeCFFHex([]rune(it.Text), fh.Metrics)
	fmt.Fprintf(buf,
		"q\n%.4f %.4f %.4f rg\nBT\n/%s %.4f Tf\n%.4f %.4f Td\n%s Tj\nET\nQ\n",
		r, g, b,
		fh.Name, it.FontSize,
		it.X, it.Y,
		operand,
	)
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
