package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestRenderEmitsMetadataInInfoDict(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Quarterly Report").
		SetAuthor("Jane Doe").
		SetSubject("Q4 summary").
		SetKeywords("revenue, growth, q4").
		Paragraph(kardec.Text("body"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	checks := []string{
		"/Title (Quarterly Report)",
		"/Author (Jane Doe)",
		"/Subject (Q4 summary)",
		"/Keywords (revenue, growth, q4)",
	}
	for _, c := range checks {
		if !bytes.Contains(out, []byte(c)) {
			t.Errorf("expected %q in /Info dict, missing", c)
		}
	}
}

func TestRenderOmitsEmptyMetadataFields(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Only Title").
		Paragraph(kardec.Text("body"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("/Title (Only Title)")) {
		t.Errorf("Title should appear in /Info")
	}
	for _, key := range []string{"/Author", "/Subject", "/Keywords"} {
		if bytes.Contains(out, []byte(key+" (")) {
			t.Errorf("empty metadata field %s should not appear in /Info", key)
		}
	}
}
