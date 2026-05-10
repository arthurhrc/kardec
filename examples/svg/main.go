// SVG demonstrates Kardec's vector SVG embedding. The image is
// emitted as a Form XObject so it stays sharp at any rendered size.
//
//	go run ./examples/svg
//
// Produces svg.pdf showing a simple icon rendered three times at
// different scales, all from the same source.
package main

import (
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

const checkmark = `<svg xmlns="http://www.w3.org/2000/svg" width="60" height="60" viewBox="0 0 60 60">
  <circle cx="30" cy="30" r="25" fill="#3366cc" stroke="#1a3366" stroke-width="2" />
  <path d="M 15 30 L 25 40 L 45 20" stroke="white" stroke-width="3" fill="none" />
</svg>`

const arrow = `<svg xmlns="http://www.w3.org/2000/svg" width="80" height="40" viewBox="0 0 80 40">
  <line x1="5" y1="20" x2="60" y2="20" stroke="black" stroke-width="3" />
  <polygon points="55,10 75,20 55,30" fill="black" />
</svg>`

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Vector SVG embedding"))

	doc.Paragraph(kardec.Text(
		"The same SVG source rendered at three sizes — vector means " +
			"the largest stays as crisp as the smallest:"))

	for _, size := range []float64{20, 40, 80} {
		doc.Image([]byte(checkmark)).Width(kardec.Pt(size)).Build()
	}

	doc.Paragraph(kardec.Text(
		"Arrow icon, drawn from <line> + <polygon> primitives:"))
	doc.Image([]byte(arrow)).Width(kardec.Pt(120)).Build()

	if err := doc.Render("svg.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered svg.pdf")
}
