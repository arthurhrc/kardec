package kardec

import "testing"

func TestFootnoteAutoNumbersPerDocument(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	a := doc.Footnote("first body")
	b := doc.Footnote("second body")

	if a.FootnoteRef() != 1 || b.FootnoteRef() != 2 {
		t.Errorf("auto-numbering: a=%d b=%d, want 1 and 2", a.FootnoteRef(), b.FootnoteRef())
	}
	notes := doc.Footnotes()
	if len(notes) != 2 {
		t.Fatalf("Document.Footnotes len = %d, want 2", len(notes))
	}
	if notes[0].Marker() != "1" || notes[1].Marker() != "2" {
		t.Errorf("markers = %q / %q, want 1 / 2", notes[0].Marker(), notes[1].Marker())
	}
	if got := notes[0].Body()[0].Text(); got != "first body" {
		t.Errorf("first body = %q", got)
	}
}

func TestFootnoteWithMarkerHonorsCustomLabel(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	r := doc.FootnoteWith("*", Text("starred body"))
	if r.Text() != "*" {
		t.Errorf("custom marker text = %q, want *", r.Text())
	}
	notes := doc.Footnotes()
	if notes[0].Marker() != "*" {
		t.Errorf("Marker() = %q, want *", notes[0].Marker())
	}
}

func TestFootnoteReturnsEmptyOnNilDocument(t *testing.T) {
	var doc *Document
	r := doc.Footnote("ignored")
	if r.Text() != "" || r.FootnoteRef() != 0 {
		t.Errorf("nil document should return zero Run, got %+v", r)
	}
}
