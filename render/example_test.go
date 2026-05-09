package render_test

import (
	"fmt"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

// ExampleBytes shows the in-memory render path: build a Document,
// call render.Bytes, and inspect the resulting PDF byte stream.
// The output is asserted as the PDF magic header so godoc shows a
// runnable, deterministic example.
func ExampleBytes() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Monthly Report")).
		Paragraph(
			kardec.Text("Sales grew "),
			kardec.Bold("12%"),
			kardec.Text(" this quarter."),
		)
	out, err := render.Bytes(doc.Document)
	if err != nil {
		fmt.Println("render error:", err)
		return
	}
	fmt.Println("PDF starts with:", string(out[:8]))
	// Output: PDF starts with: %PDF-1.7
}

// ExampleToFile writes the document to a path. Errors from os.Create
// or the underlying renderer surface through the returned error.
func ExampleToFile() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("Hello, file."))
	_ = doc       // build steps
	_ = render.ToFile // render.ToFile(doc.Document, "out.pdf")
}
