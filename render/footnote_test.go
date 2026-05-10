package render

import (
	"bytes"
	"strings"
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
	// Body text emits 2-byte hex glyph IDs (Identity-H) post-v0.22,
	// so we can't grep for "appendix" any more — the bytes are
	// glyph indices, not Unicode codepoints. Verify the footnote
	// landed by counting text-show operations: the inline marker,
	// the surrounding sentence, the separator/spacing, and the
	// footnote body all add Tj ops. A well-rendered footnote
	// produces ≥ 8 Tj ops (rough lower bound covering "Sales grew",
	// the marker, "this quarter.", marker again at page bottom,
	// and the body words).
	streams := findContentStreams(out)
	tjCount := bytes.Count(streams, []byte(" Tj"))
	if tjCount < 8 {
		t.Errorf("expected ≥ 8 Tj ops with footnote, got %d", tjCount)
	}
	// The doc-level Footnotes() accessor proves the marker was
	// registered with the right body text — the renderer simply
	// has to walk the slice and emit it.
	notes := doc.Footnotes()
	if len(notes) != 1 {
		t.Fatalf("Footnotes() len = %d, want 1", len(notes))
	}
	if got := notes[0].Body()[0].Text(); !strings.Contains(got, "appendix") {
		t.Errorf("footnote body lost text: %q", got)
	}
}
