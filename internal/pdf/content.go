package pdf

import (
	"bytes"
	"fmt"
)

// buildContentStream renders a Page's TextItems into the byte payload of
// a PDF content stream (PDF 7.8.2). The returned bytes are the raw
// operators; the caller wraps them with the stream dictionary and the
// stream/endstream markers via objectWriter.writeStreamObject.
//
// The op sequence per text item is:
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
// Wrapping each item in q/Q isolates color state so a later item with no
// explicit color falls back to the page default rather than inheriting
// the previous item's. It costs ~6 bytes per item — negligible.
func buildContentStream(page Page, fonts []*fontHandle) []byte {
	var buf bytes.Buffer
	for _, it := range page.Items {
		if it.FontID < 0 || it.FontID >= len(fonts) {
			continue // skip silently rather than panicking on bad input
		}
		fh := fonts[it.FontID]
		// Convert color (uint8 0..255) to PDF's 0..1 range.
		r := float64(it.Color.R) / 255.0
		g := float64(it.Color.G) / 255.0
		b := float64(it.Color.B) / 255.0

		// WinAnsi-encode the text and escape it for a PDF literal string.
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

// resourcesDict returns the /Resources dict body for a page, listing every
// font under its /Font subdictionary. The /ProcSet entry is legacy (PDF
// 1.4 deprecated requiring it) but Acrobat older than 5 still warns
// without it; including it costs nothing.
func resourcesDict(fonts []*fontHandle) string {
	var buf bytes.Buffer
	buf.WriteString("<< /ProcSet [/PDF /Text] /Font <<")
	for _, fh := range fonts {
		fmt.Fprintf(&buf, " /%s %s", fh.Name, ref(fh.DictID))
	}
	buf.WriteString(" >> >>")
	return buf.String()
}
