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

func TestRenderMathEmbedsLatinModernMath(t *testing.T) {
	// v0.15 wired CFF font embedding (Type 0 + CIDFontType0 +
	// FontFile3 / Subtype /CIDFontType0C). Math content now
	// embeds Latin Modern Math itself instead of falling back to
	// the body font's accidental glyph coverage. The PDF byte
	// stream should reference the math font by name and carry
	// the CIDFontType0C subtype on the FontFile3 stream.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).Math(`\alpha + \beta`)
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	for _, want := range []string{
		"LatinModernMath",
		"/Subtype /Type0",
		"/Encoding /Identity-H",
		"/Subtype /CIDFontType0",
		"/Subtype /CIDFontType0C",
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("math-font embedding marker %q missing from PDF byte stream", want)
		}
	}
}
