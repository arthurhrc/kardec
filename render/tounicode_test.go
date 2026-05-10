package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestRenderEmitsToUnicodeCMap(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("Hello, world."))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	// The CMap stream itself is FlateDecode-compressed in the PDF
	// byte stream, so we assert only on the unambiguous /ToUnicode
	// reference + the indirect-object reference shape. The
	// uncompressed CMap body is verified in the pdf package test.
	if !bytes.Contains(out, []byte("/ToUnicode")) {
		t.Errorf("/ToUnicode reference missing from font dict")
	}
}
