package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

// fakeICCProfile is a small placeholder byte slice used only to
// exercise the writer plumbing. A real PDF/A-2b profile would be the
// sRGB IEC 61966-2.1 ICC profile (~3 KB); the writer doesn't validate
// the bytes, so a sentinel works for plumbing tests.
var fakeICCProfile = []byte("FAKE-ICC-PROFILE-BYTES")

func TestPDFAWithICCProfileEmitsOutputIntent(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		EnablePDFA().
		SetICCProfile(fakeICCProfile, 3).
		Paragraph(kardec.Text("body"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	checks := []string{
		"/OutputIntents",
		"/Type /OutputIntent",
		"/S /GTS_PDFA1",
		"/OutputConditionIdentifier (sRGB IEC61966-2.1)",
		"/RegistryName (http://www.color.org)",
		"/DestOutputProfile",
		"FAKE-ICC-PROFILE-BYTES",
	}
	for _, c := range checks {
		if !bytes.Contains(out, []byte(c)) {
			t.Errorf("expected %q in PDF byte stream, missing", c)
		}
	}
}

func TestPDFAWithoutICCProfileSkipsOutputIntent(t *testing.T) {
	// Lite PDF/A: markers without OutputIntent. Acrobat accepts;
	// veraPDF flags as non-conformant. Backward-compat path.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		EnablePDFA().
		Paragraph(kardec.Text("body"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if bytes.Contains(out, []byte("/OutputIntents")) {
		t.Errorf("OutputIntents leaked into a doc without ICC profile")
	}
	// Markers should still be there.
	if !bytes.Contains(out, []byte("<pdfaid:part>2</pdfaid:part>")) {
		t.Errorf("PDF/A markers missing from lite output")
	}
}

func TestSetICCProfileWithoutPDFAIsNoop(t *testing.T) {
	// ICC profile without EnablePDFA shouldn't emit OutputIntent —
	// the catalog entry only makes sense alongside the PDF/A
	// markers it complements.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetICCProfile(fakeICCProfile, 3).
		Paragraph(kardec.Text("body"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if bytes.Contains(out, []byte("/OutputIntents")) {
		t.Errorf("OutputIntents emitted without EnablePDFA")
	}
}
