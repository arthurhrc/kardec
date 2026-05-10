// Watermark demonstrates Kardec's per-page diagonal watermark stamp.
// SetWatermark configures the text, color, opacity, angle, and font
// size; the renderer paints it on every page after primary content.
//
//	go run ./examples/watermark
//
// Produces watermark.pdf with a "DRAFT" stamp diagonally across each
// page in 30%-opacity gray.
package main

import (
	"log"
	"strings"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetWatermark("DRAFT", kardec.WatermarkOptions{
			Color:    kardec.Color{R: 0xC0, G: 0x40, B: 0x40},
			Opacity:  0.20,
			AngleDeg: 45,
			FontSize: 80,
		}).
		Heading(1, kardec.Text("Quarterly Report — DRAFT")).
		Paragraph(
			kardec.Text("This is an unfinalised draft. Numbers may change "),
			kardec.Text("before the public release. The diagonal watermark sits "),
			kardec.Text("on every page so a reader who lifts a single page out "),
			kardec.Text("of context still sees the warning."),
		)

	// Add enough filler paragraphs to make the doc spill across
	// pages, so the per-page watermark is visible on each one.
	for i := 0; i < 12; i++ {
		doc.Paragraph(kardec.Text(strings.Repeat("Filler text. ", 40)))
	}

	if err := doc.Render("watermark.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered watermark.pdf")
}
