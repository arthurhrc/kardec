// Math demonstrates the LaTeX math subset shipping in v0.3. The
// document mixes display equations (one per Math call) with body text
// so the integration of the math layout engine is visible end-to-end.
//
// Run from the repository root:
//
//	go run ./examples/math
//
// The resulting math.pdf carries five display equations plus
// surrounding prose. v0.3 emits the math glyphs only; fraction bars
// and square-root overlines are queued for v0.3.1 once the PDF writer
// grows a rectangle primitive.
package main

import (
	"fmt"
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Math typesetting in Kardec")).
		Paragraph(
			kardec.Text("v0.3 introduces a LaTeX math subset (see RFC-001 §16). "),
			kardec.Text("Greek letters, fractions, square roots, sub/superscripts and "),
			kardec.Text("big operators (sum, integral, product) parse and lay out."),
		).
		Paragraph(kardec.Text("Pythagoras:"))

	doc.Math(`a^2 + b^2 = c^2`)

	doc.Paragraph(kardec.Text("A fraction:"))
	doc.Math(`\frac{a + b}{c}`)

	doc.Paragraph(kardec.Text("Square root:"))
	doc.Math(`\sqrt{x^2 + y^2}`)

	doc.Paragraph(kardec.Text("Summation in display style:"))
	doc.Math(`\sum_{i=0}^{n} i^2`)

	doc.Paragraph(kardec.Text("Integral with bounds:"))
	doc.Math(`\int_0^1 \frac{1}{x} dx`)

	doc.Paragraph(
		kardec.Text("Greek letters in formulas: "),
		kardec.Text("alpha, beta, gamma all map to glyphs in Latin Modern Math."),
	)
	doc.Math(`\alpha + \beta = \gamma`)

	if err := doc.Err(); err != nil {
		log.Fatalf("builder error: %v", err)
	}
	if err := doc.Render("math.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Println("rendered math.pdf")
}
