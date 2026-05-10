// InlineMath demonstrates Kardec's Run-level math: a LaTeX
// expression embedded inside a paragraph, sitting on the body
// baseline next to surrounding prose.
//
//	go run ./examples/inlinemath
//
// Produces inlinemath.pdf. Notice that "a²+b²=c²" flows with the
// surrounding text rather than living on its own display line.
package main

import (
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Inline math in paragraphs"))

	doc.Paragraph(
		kardec.Text("By Pythagoras, "),
		kardec.InlineMath("a^2 + b^2 = c^2"),
		kardec.Text(", which holds for any right triangle."),
	)

	doc.Paragraph(
		kardec.Text("Greek letters render correctly: "),
		kardec.InlineMath(`\alpha + \beta = \gamma`),
		kardec.Text(", and they share the body baseline."),
	)

	doc.Paragraph(
		kardec.Text("For display-style equations (centered, taller), "),
		kardec.Text("use Document.Math instead:"),
	)
	doc.Math(`\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}`)

	doc.Paragraph(
		kardec.Text("Inline-math width factors into line breaking, so "),
		kardec.Text("a long expression like "),
		kardec.InlineMath("x_1 + x_2 + x_3 + x_4"),
		kardec.Text(" stays as a single non-breakable unit and moves "),
		kardec.Text("to the next line if it doesn't fit."),
	)

	if err := doc.Render("inlinemath.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered inlinemath.pdf")
}
