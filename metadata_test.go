package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestSetTitleAuthorSubjectKeywordsRoundTrip(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Quarterly Report").
		SetAuthor("Jane Doe").
		SetSubject("Q4 financial summary").
		SetKeywords("revenue, growth, q4")

	if got := doc.Title(); got != "Quarterly Report" {
		t.Errorf("Title = %q, want Quarterly Report", got)
	}
	if got := doc.Author(); got != "Jane Doe" {
		t.Errorf("Author = %q, want Jane Doe", got)
	}
	if got := doc.Subject(); got != "Q4 financial summary" {
		t.Errorf("Subject = %q", got)
	}
	if got := doc.Keywords(); got != "revenue, growth, q4" {
		t.Errorf("Keywords = %q", got)
	}
}

func TestMetadataDefaultsAreEmpty(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	for label, got := range map[string]string{
		"Title":    doc.Title(),
		"Author":   doc.Author(),
		"Subject":  doc.Subject(),
		"Keywords": doc.Keywords(),
	} {
		if got != "" {
			t.Errorf("%s default = %q, want empty", label, got)
		}
	}
}

func TestMetadataSettersChain(t *testing.T) {
	// Each setter returns *Document so they compose with the rest of
	// the fluent builder.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("A").
		SetAuthor("B").
		Heading(1, kardec.Text("body")).
		SetSubject("C").
		SetKeywords("d")
	if doc.Subject() != "C" || doc.Keywords() != "d" {
		t.Errorf("setters after Heading didn't apply: subject=%q keywords=%q",
			doc.Subject(), doc.Keywords())
	}
}
