// Tagged demonstrates Kardec's PDF/UA tagging integration.
// SetTagged opts the document into structure-tag emission; the
// renderer wraps each block (heading, paragraph, figure) in its
// own marked-content sequence and emits a StructTreeRoot whose
// elements carry the matching role (H1..H6, P, Figure).
//
//	go run ./examples/tagged
//
// Produces tagged.pdf. Open in Acrobat, view the document
// properties → "Tagged PDF: Yes". Screen readers walk the H1 →
// paragraph → H2 → paragraph hierarchy in logical order.
package main

import (
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTagged("en"). // BCP-47 language code
		SetTitle("Tagged PDF Demo").
		Heading(1, kardec.Text("Tagged PDF — accessibility-ready")).
		Paragraph(
			kardec.Text("This document opts in to PDF/UA-1 lite tagging via "),
			kardec.Text("SetTagged(\"en\"). Each heading carries an Hn role; "),
			kardec.Text("each paragraph carries a P role; figures get the "),
			kardec.Text("Figure role. Screen readers walk the structure tree "),
			kardec.Text("in logical order rather than chasing visual position."),
		).
		Heading(2, kardec.Text("Why it matters")).
		Paragraph(
			kardec.Text("Untagged PDFs force assistive tech to guess the "),
			kardec.Text("reading order from glyph position, which fails on "),
			kardec.Text("multi-column layouts, sidebars, and figure captions. "),
			kardec.Text("A tagged structure makes intent explicit."),
		).
		Heading(2, kardec.Text("Conformance"))

	doc.Paragraph(kardec.Text(
		"v0.22 ships role-classified blocks (H1–H6, P, Figure). " +
			"Strict PDF/UA-1 conformance still wants nested Sect groupings " +
			"and table TR/TD/TH, queued for v0.22.x."))

	if err := doc.Render("tagged.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered tagged.pdf")
}
