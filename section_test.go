package kardec

import "testing"

func TestDocumentHeaderAndFooterAttachToCurrentSection(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		Header(Text("Top — page {{page}}")).
		Footer(Text("Bottom — {{date}}"))

	sec := doc.Sections()[0]
	if len(sec.Header) != 1 {
		t.Fatalf("Header length = %d, want 1", len(sec.Header))
	}
	if got := sec.Header[0].Text(); got != "Top — page {{page}}" {
		t.Errorf("Header text = %q", got)
	}
	if len(sec.Footer) != 1 {
		t.Fatalf("Footer length = %d, want 1", len(sec.Footer))
	}
}

func TestDocumentHeaderEmptyArgsClearsExisting(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		Header(Text("first")).
		Header() // explicit clear

	if len(doc.Sections()[0].Header) != 0 {
		t.Errorf("Header() with no args should clear, got %d", len(doc.Sections()[0].Header))
	}
}

func TestDocumentHeaderInertAfterDeferredError(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errInternalForSectionTest())
	doc.Header(Text("ignored"))
	if len(doc.Sections()[0].Header) != 0 {
		t.Errorf("Header should be inert once an error is captured")
	}
}

func errInternalForSectionTest() error { return &sectionSimpleErr{msg: "synthetic"} }

type sectionSimpleErr struct{ msg string }

func (e *sectionSimpleErr) Error() string { return e.msg }
