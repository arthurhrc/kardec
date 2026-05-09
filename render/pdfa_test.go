package render

import (
	"bytes"
	"testing"
	"time"

	"github.com/arthurhrc/kardec"
)

func TestPDFAOptInEmitsConformanceMarkers(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		PDFA().
		SetCreationDate(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)).
		Heading(1, kardec.Text("Title")).
		Paragraph(kardec.Text("body"))
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	checks := []string{
		"/Metadata",
		"/Type /Metadata",
		"/Subtype /XML",
		"<pdfaid:part>2</pdfaid:part>",
		"<pdfaid:conformance>B</pdfaid:conformance>",
		"/ID [<",
	}
	for _, c := range checks {
		if !bytes.Contains(out, []byte(c)) {
			t.Errorf("PDF/A marker %q missing from output", c)
		}
	}
}

func TestPDFAOffByDefault(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("body"))
	if doc.PDFAEnabled() {
		t.Errorf("PDFA must be off by default")
	}
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if bytes.Contains(out, []byte("pdfaid:part")) {
		t.Errorf("PDF/A marker leaked into a non-opted-in document")
	}
}

func TestPDFADocumentIDStableAcrossRuns(t *testing.T) {
	build := func() []byte {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			PDFA().
			SetCreationDate(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)).
			Paragraph(kardec.Text("body"))
		out, err := Bytes(doc)
		if err != nil {
			t.Fatalf("Bytes: %v", err)
		}
		return out
	}
	if !bytes.Equal(build(), build()) {
		t.Errorf("PDF/A output should be byte-reproducible when SetCreationDate is fixed")
	}
}
