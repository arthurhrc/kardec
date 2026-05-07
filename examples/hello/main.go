// Hello demonstrates the smallest meaningful Kardec document. The full
// rendering pipeline (layout + typography + pdf) is wired through the
// kardec/render package, which the blank import below installs as the
// implementation behind Document.Render.
//
// Run from the repository root:
//
//	go run ./examples/hello
//
// hello.pdf is produced in the current directory; open it in Chrome,
// Acrobat or pdftk to confirm. v0.1 embeds Liberation Sans Regular for
// every text run; multi-face embedding (real bold / italic glyphs) lands
// in v0.2.
package main

import (
	"fmt"
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
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
