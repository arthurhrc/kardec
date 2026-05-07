// Hello demonstrates the smallest meaningful Kardec document. The Render
// call currently returns kardec.ErrNotImplemented; this example will start
// producing actual PDFs once the layout, typography, and renderer tracks
// land on main.
package main

import (
	"errors"
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
		Paragraph(kardec.Italic("Skeleton phase — output is not yet generated."))

	if err := doc.Err(); err != nil {
		log.Fatalf("builder error: %v", err)
	}

	err := doc.Render("hello.pdf")
	switch {
	case errors.Is(err, kardec.ErrNotImplemented):
		fmt.Println("builder OK — render path lands in v0.1 layout/typography/renderer tracks")
	case err != nil:
		log.Fatalf("render: %v", err)
	default:
		fmt.Println("rendered hello.pdf")
	}
}
