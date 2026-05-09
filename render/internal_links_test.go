package render

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestRenderEmitsNamedDestinationForAnchor(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Anchor("introduction").
		Heading(1, kardec.Text("Introduction")).
		Paragraph(kardec.Text("body"))

	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("/Dests")) {
		t.Errorf("expected /Dests dictionary in catalog")
	}
	if !bytes.Contains(out, []byte("(introduction)")) {
		t.Errorf("expected named destination 'introduction' in PDF byte stream")
	}
}

func TestRenderInternalLinkUsesGoToAction(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Link("Jump to chapter 2", "#chapter-2")).
		PageBreak().
		Anchor("chapter-2").
		Heading(2, kardec.Text("Chapter 2"))

	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("/A << /Type /Action /S /GoTo /D")) {
		t.Errorf("expected /GoTo /D action for internal link")
	}
	if bytes.Contains(out, []byte("/A << /Type /Action /S /URI /URI (#chapter-2)")) {
		t.Errorf("internal link should not be emitted as /URI")
	}
}

func TestRenderExternalLinkStillUsesURI(t *testing.T) {
	// Guard: the routing change does not regress external links.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Link("docs", "https://example.com/docs"))

	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("/S /URI")) {
		t.Errorf("external links must still emit /URI")
	}
}
