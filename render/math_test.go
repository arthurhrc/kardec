package render

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestRenderMathProducesValidPDF(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Equations")).
		Math(`\sum_{i=0}^{n} i^2`).
		Math(`\frac{a}{b}`)

	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.7")) {
		t.Errorf("missing %%PDF-1.7 header")
	}
	if !bytes.HasSuffix(bytes.TrimSpace(out), []byte("%%EOF")) {
		t.Errorf("missing %%EOF trailer")
	}
}

func TestRenderMathDoesNotEmbedMathFontYet(t *testing.T) {
	// v0.3 routes math glyphs through the default body font because
	// the PDF writer cannot embed OpenType/CFF yet (Latin Modern Math
	// is OTF). When CFF support lands the writer will start including
	// "LatinModernMath" — until then this assertion guards the
	// documented limitation.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).Math(`\alpha + \beta`)
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if bytes.Contains(out, []byte("LatinModernMath")) {
		t.Errorf("math font embedding is documented as v0.3.x; should not appear yet")
	}
}
