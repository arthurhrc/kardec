// Hello demonstrates the smallest meaningful Kardec document. The Render
// path is wired end-to-end through the internal/pdf writer, so running
// this example actually produces hello.pdf — open it in Chrome, Acrobat
// or pdftk to confirm.
//
// Note that the Layout track is still stubbed at the time of writing,
// so the rendered page is currently blank (a valid PDF nonetheless).
// When Layout lands, the same builder calls below will lay out the
// heading and paragraphs without changes to this example.
package main

import (
	"fmt"
	"log"

	"github.com/arthurhrc/kardec"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Hello, Kardec")).
		Paragraph(
			kardec.Text("Kardec produces "),
			kardec.Bold("document-like"),
			kardec.Text(" PDFs without containers or LibreOffice."),
		).
		Spacer(kardec.Pt(12)).
		Paragraph(kardec.Italic("Renderer track wired — Layout integration coming next."))

	if err := doc.Err(); err != nil {
		log.Fatalf("builder error: %v", err)
	}

	if err := doc.Render("hello.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Println("rendered hello.pdf")
}
