// Bookstyle demonstrates two book-grade chrome features:
//
//   - SetBackgroundImage for a letterhead-style decorative band on
//     every page.
//   - FirstPageHeader / EvenPageHeader for asymmetric verso/recto
//     running heads (chapter title left, document title right).
//
//	go run ./examples/bookstyle
//
// Produces bookstyle.pdf with 5 pages exercising all the variants.
package main

import (
	"log"
	"strings"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

const watermarkBg = `<svg xmlns="http://www.w3.org/2000/svg" width="595" height="842" viewBox="0 0 595 842">
  <rect x="0" y="0" width="595" height="842" fill="#fdfdfd" />
  <line x1="40" y1="40" x2="555" y2="40" stroke="#888888" stroke-width="1" />
  <line x1="40" y1="802" x2="555" y2="802" stroke="#888888" stroke-width="1" />
  <rect x="40" y="40" width="3" height="762" fill="#aa3333" />
</svg>`

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Bookstyle demo").
		SetBackgroundImage([]byte(watermarkBg)).
		// First page: no running head, just the cover band.
		FirstPageHeader().
		FirstPageFooter().
		// Default header (odd / recto pages from page 3 onward).
		Header(kardec.Italic("Bookstyle demo")).
		// Even (verso) pages: chapter title on the left.
		EvenPageHeader(kardec.Italic("Chapter 1 — The setup")).
		Footer(kardec.Text("— {{page}} —"))

	doc.Heading(1, kardec.Text("Cover"))
	doc.Paragraph(kardec.Text(
		"This page is the first in its section. The FirstPageHeader / " +
			"FirstPageFooter were set to empty calls, so the renderer " +
			"suppresses any running chrome here — only the decorative band " +
			"from SetBackgroundImage shows."))
	doc.PageBreak()

	// Pages 2..5: alternate even/odd to exercise both heads.
	for i := 0; i < 4; i++ {
		doc.Heading(2, kardec.Text("Chapter section"))
		doc.Paragraph(kardec.Text(strings.Repeat("Body text fills the page. ", 50)))
		doc.PageBreak()
	}

	if err := doc.Render("bookstyle.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered bookstyle.pdf")
}
