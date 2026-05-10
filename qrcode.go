package kardec

import (
	"bytes"
	"fmt"

	"rsc.io/qr"
)

// QRCode renders a QR code carrying data at the given size (a
// square in points) and appends it as an Image block to the
// current section. The block participates in the normal flow:
// it sits between its surrounding blocks, honours alignment via
// the returned ImageBuilder, and respects column / page
// constraints.
//
// QR codes flow through the SVG → Form XObject path so the result
// stays sharp at any rendered size — important when the same
// document is read on screen at 100% and printed at 600 DPI.
//
// errorLevel selects the QR error-correction tier. Higher levels
// produce denser codes that survive more damage (smudges, fold
// lines, partial coverage), at the cost of larger pixel grids.
// Pass kardec.QRMedium for the typical "fits a phone-camera scan
// reliably on print" tradeoff.
//
//	doc.QRCode("https://example.com/checkout/abc123",
//	    kardec.QRMedium, kardec.Pt(80)).Center().Build()
//
// Empty data or invalid input collapses into the document's
// deferred error; Render surfaces it.
func (d *Document) QRCode(data string, errorLevel QRErrorLevel, size Length) *ImageBuilder {
	b := &ImageBuilder{doc: d}
	if d.err != nil {
		return b
	}
	if data == "" {
		b.err = fmt.Errorf("kardec: QRCode: empty data")
		return b
	}
	svg, err := qrCodeSVG(data, errorLevel)
	if err != nil {
		b.err = fmt.Errorf("kardec: QRCode: %w", err)
		return b
	}
	b.img = Image{
		data:   svg,
		format: ImageFormatSVG,
		width:  size,
		height: size,
	}
	return b
}

// QRErrorLevel selects the QR redundancy tier. Higher levels
// recover from more pixel damage at the cost of a denser code.
//
//	QRLow    — 7% recovery; smallest grid; fragile.
//	QRMedium — 15% recovery; the spec's "M" level; standard.
//	QRQuart  — 25% recovery; print-grade; tolerates folds.
//	QRHigh   — 30% recovery; densest; for hostile environments.
type QRErrorLevel uint8

const (
	QRLow QRErrorLevel = iota
	QRMedium
	QRQuart
	QRHigh
)

// qrCodeSVG encodes data with the chosen error level and returns
// an SVG byte slice that, when scaled to the target size, paints
// the QR matrix as a grid of black squares on white.
//
// Inline SVG generation keeps the renderer dependency-light: no
// PNG rasterisation, no /Subtype /Image with raw RGB pixels — the
// vector Form XObject path already in place handles the rest.
func qrCodeSVG(data string, level QRErrorLevel) ([]byte, error) {
	qrLevel := qr.M
	switch level {
	case QRLow:
		qrLevel = qr.L
	case QRMedium:
		qrLevel = qr.M
	case QRQuart:
		qrLevel = qr.Q
	case QRHigh:
		qrLevel = qr.H
	}
	code, err := qr.Encode(data, qrLevel)
	if err != nil {
		return nil, err
	}
	// Compose the SVG. We use viewBox = matrix size so a caller
	// requesting a 100 pt QR gets per-cell scaling of 100/size.
	// A 4-cell-wide "quiet zone" surrounds the matrix per spec.
	const quiet = 4
	dim := code.Size + 2*quiet
	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		dim, dim, dim, dim)
	// White background covers the entire viewBox including the
	// quiet zone.
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="white" />`, dim, dim)
	// Black cells.
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if code.Black(x, y) {
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="1" height="1" fill="black" />`,
					x+quiet, y+quiet)
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.Bytes(), nil
}
