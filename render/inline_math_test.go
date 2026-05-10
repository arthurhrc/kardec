package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestInlineMathRendersAlongsideProse(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(
			kardec.Text("By Pythagoras, "),
			kardec.InlineMath("a^2 + b^2 = c^2"),
			kardec.Text(" for any right triangle."),
		).Document
	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	// Math glyphs need the math face — Latin Modern Math is
	// embedded via a Type 0 / Identity-H wrapper, so the byte
	// stream must reference both the body face and the math one.
	for _, want := range []string{
		"/Subtype /Type0",        // math font wrapped as composite
		"/Subtype /CIDFontType0", // CFF descendant
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("inline-math marker %q missing from PDF byte stream", want)
		}
	}
}

func TestInlineMathParseFailureDropsRunSilently(t *testing.T) {
	// Malformed expression — current behaviour is to drop the run
	// entirely so the surrounding paragraph still renders.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(
			kardec.Text("before "),
			kardec.InlineMath(`\frac`), // unfinished, parser-rejected
			kardec.Text(" after"),
		).Document
	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Contains(out, []byte("before")) || !bytes.Contains(out, []byte("after")) {
		t.Errorf("paragraph should still render the surrounding prose; got %q", out[:200])
	}
}

func TestInlineMathAccessorRoundtrip(t *testing.T) {
	r := kardec.InlineMath(`x^2 + y^2 = z^2`)
	if r.MathSource() != `x^2 + y^2 = z^2` {
		t.Errorf("MathSource roundtrip lost: got %q", r.MathSource())
	}
	plain := kardec.Text("hello")
	if plain.MathSource() != "" {
		t.Errorf("plain Run should report empty MathSource, got %q", plain.MathSource())
	}
}
