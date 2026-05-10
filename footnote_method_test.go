package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestFootnoteMethodAutoNumbersInOrder(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	a := doc.Footnote("first")
	b := doc.Footnote("second")
	if a.FootnoteRef() != 1 {
		t.Errorf("first footnote ref = %d, want 1", a.FootnoteRef())
	}
	if b.FootnoteRef() != 2 {
		t.Errorf("second footnote ref = %d, want 2", b.FootnoteRef())
	}
}

func TestFootnoteWithMethodHonorsCustomMarker(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	r := doc.FootnoteWith("*", kardec.Text("note body"))
	if r.Text() != "*" {
		t.Errorf("custom-marker footnote run text = %q, want %q", r.Text(), "*")
	}
	if got := doc.Footnotes()[0].Body()[0].Text(); got != "note body" {
		t.Errorf("body run = %q, want %q", got, "note body")
	}
}

