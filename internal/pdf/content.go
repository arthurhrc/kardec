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

		encoded := encodeWinAnsi([]rune(it.Text))
		literal := escapeLiteralString(string(encoded))

		fmt.Fprintf(&buf,
			"q\n%.4f %.4f %.4f rg\nBT\n/%s %.4f Tf\n%.4f %.4f Td\n%s Tj\nET\nQ\n",
			r, g, b,
			fh.Name, it.FontSize,
			it.X, it.Y,
			literal,
		)
	}
	return buf.Bytes()
}

// resourcesDict returns the /Resources dict body for a page. Fonts go
// under /Font; images go under /XObject. /ProcSet stays for legacy
// Acrobat compatibility and grows /ImageC when images are present.
func resourcesDict(fonts []*fontHandle, images []*imageHandle) string {
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
	buf.WriteString(" >>")
	return buf.String()
}
