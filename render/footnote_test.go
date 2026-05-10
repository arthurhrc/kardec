package render

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestRenderEmitsFootnoteAtPageBottom(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Paragraph(
		kardec.Text("Sales grew "),
		doc.Footnote("see appendix B for the breakdown."),
		kardec.Text(" this quarter."),
	)
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := findContentStreams(out)
	if !bytes.Contains(streams, []byte("appendix")) {
		t.Errorf("expected footnote body in PDF content stream")
	}
	// The marker "1" should also appear (inline + footnote area).
	if !bytes.Contains(streams, []byte("(1) Tj")) {
		t.Errorf("expected the auto-numbered marker '1' to be rendered")
	}
}
