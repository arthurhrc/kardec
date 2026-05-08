package render

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestRenderEmitsLinkAnnotForExplicitLinkRun(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(
			kardec.Text("Visit "),
			kardec.Link("Kardec", "https://github.com/arthurhrc/kardec"),
			kardec.Text(" on GitHub."),
		)
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("/Subtype /Link")) {
		t.Errorf("expected /Subtype /Link annotation in PDF")
	}
	if !bytes.Contains(out, []byte("https://github.com/arthurhrc/kardec")) {
		t.Errorf("expected URI literal in PDF byte stream")
	}
	if !bytes.Contains(out, []byte("/A << /Type /Action /S /URI")) {
		t.Errorf("expected /URI action descriptor")
	}
}

func TestRenderEmitsOutlineForHeadings(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Top")).
		Paragraph(kardec.Text("body")).
		Heading(2, kardec.Text("Inner")).
		Paragraph(kardec.Text("more"))

	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("/Type /Outlines")) {
		t.Errorf("expected /Type /Outlines root in PDF")
	}
	if !bytes.Contains(out, []byte("/Title (Top)")) {
		t.Errorf("expected outline entry for 'Top' heading")
	}
	if !bytes.Contains(out, []byte("/Title (Inner)")) {
		t.Errorf("expected outline entry for nested 'Inner' heading")
	}
	if !bytes.Contains(out, []byte("/PageMode /UseOutlines")) {
		t.Errorf("expected catalog /PageMode /UseOutlines so readers open the sidebar")
	}
}

func TestRenderMarkdownLinkBecomesAnnot(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		AppendMarkdown(`See [the docs](https://example.com/docs) for details.`)
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("https://example.com/docs")) {
		t.Errorf("Markdown link URL did not flow into the PDF")
	}
	if !bytes.Contains(out, []byte("/Subtype /Link")) {
		t.Errorf("Markdown link did not produce a /Link annotation")
	}
}
