package pdf

import (
	"bytes"
	"fmt"
	"math"
)

// appendWatermark writes the watermark operators after the page's
// primary content stream (rects, images, text). The stamp is
// painted last so it sits on top of body content, but with optional
// alpha blending it never fully obscures what lies beneath.
//
// Coordinate math: the watermark is placed at the page centre and
// rotated counter-clockwise by AngleDeg. Centring is done by
// computing the rendered text width via the font's metrics and
// shifting the text-matrix origin by half that width and half a
// line height.
//
// alphaName is the resource name of the /ExtGState dict configured
// for the requested opacity. Empty string means "no alpha" — the
// stamp is fully opaque (Opacity ≥ 1).
func appendWatermark(buf *bytes.Buffer, w *Watermark, fh *fontHandle, pageWidth, pageHeight float64, alphaName string) {
	if w == nil || w.Text == "" || fh == nil {
		return
	}
	textWidth := watermarkTextWidth(w.Text, fh, w.FontSize)
	rad := w.AngleDeg * math.Pi / 180.0
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	// Build the cm matrix:
	// 1. translate to page centre,
	// 2. rotate by AngleDeg,
	// 3. translate left by half text width and down by half font
	//    size so the rotated text is visually centred on the page.
	// The combined matrix multiplied out below avoids two cm calls.
	cx := pageWidth / 2
	cy := pageHeight / 2
	tx := cx - cosA*(textWidth/2) + sinA*(w.FontSize/2)
	ty := cy - sinA*(textWidth/2) - cosA*(w.FontSize/2)

	r := float64(w.Color.R) / 255.0
	g := float64(w.Color.G) / 255.0
	b := float64(w.Color.B) / 255.0

	buf.WriteString("q\n")
	if alphaName != "" {
		fmt.Fprintf(buf, "/%s gs\n", alphaName)
	}
	fmt.Fprintf(buf, "%.6f %.6f %.6f rg\n", r, g, b)
	fmt.Fprintf(buf,
		"%.6f %.6f %.6f %.6f %.6f %.6f cm\n",
		cosA, sinA, -sinA, cosA, tx, ty,
	)
	buf.WriteString("BT\n")
	fmt.Fprintf(buf, "/%s %.4f Tf\n", fh.Name, w.FontSize)

	// Encoding follows the same TrueType vs CFF split as body text;
	// watermarks are normally English ASCII so WinAnsi covers the
	// common case, but the CFF path is wired in too so a math-font
	// or Asian-glyph watermark renders correctly when it ships in
	// v0.20.x.
	var operand string
	switch fh.Kind {
	case fontKindCFF:
		operand = encodeCFFHex([]rune(w.Text), fh.Metrics)
	default:
		encoded := encodeWinAnsi([]rune(w.Text))
		operand = escapeLiteralString(string(encoded))
	}
	fmt.Fprintf(buf, "%s Tj\nET\nQ\n", operand)
}

// watermarkTextWidth measures the rendered width of s at fontSize
// using the font's parsed advance-width table. Falls back to a
// conservative estimate (font size × 0.55 × len) when metrics are
// missing — only happens for the synthetic test fonts.
func watermarkTextWidth(s string, fh *fontHandle, fontSize float64) float64 {
	if fh.Metrics == nil || fh.Metrics.UnitsPerEm == 0 {
		return float64(len(s)) * fontSize * 0.55
	}
	scale := fontSize / float64(fh.Metrics.UnitsPerEm)
	var advance float64
	for _, r := range s {
		gid := uint16(0)
		if g, ok := fh.Metrics.CmapUnicode[uint32(r)]; ok {
			gid = g
		}
		if int(gid) < len(fh.Metrics.AdvanceWidth) {
			advance += float64(fh.Metrics.AdvanceWidth[gid]) * scale
		} else {
			advance += fontSize * 0.55
		}
	}
	return advance
}

// emitWatermarkAlpha emits an /ExtGState dict for the requested
// opacity and returns the indirect-object ID plus the resource
// name pages should reference. opacity ≥ 1 (or ≤ 0) returns 0,""
// — caller skips the gs operator entirely.
func emitWatermarkAlpha(ow *objectWriter, opacity float64) (int, string) {
	if opacity <= 0 || opacity >= 1 {
		return 0, ""
	}
	id := ow.allocID()
	ow.writeObject(id, fmt.Sprintf(
		"<< /Type /ExtGState /ca %.4f /CA %.4f >>",
		opacity, opacity,
	))
	return id, "Gsw"
}
