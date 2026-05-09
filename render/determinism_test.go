package render

import (
	"bytes"
	"testing"
	"time"

	"github.com/arthurhrc/kardec"
)

// TestRenderByteReproducibleWithFixedClock is the headline guarantee
// the strategic audit's competitive analysis pinned as the unique
// angle for Kardec in Go: same input plus same fixed clock yields
// byte-identical output. The test renders the same document twice
// and asserts bytes.Equal — a single non-deterministic value (the
// info-dict timestamp) was the only known source of drift.
func TestRenderByteReproducibleWithFixedClock(t *testing.T) {
	build := func() []byte {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			SetCreationDate(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)).
			Heading(1, kardec.Text("Reproducible report")).
			Paragraph(
				kardec.Text("Same input, "),
				kardec.Bold("same bytes"),
				kardec.Text(" — every render."),
			).
			Math(`a^2 + b^2 = c^2`)
		out, err := Bytes(doc)
		if err != nil {
			t.Fatalf("Bytes: %v", err)
		}
		return out
	}

	first := build()
	second := build()
	if !bytes.Equal(first, second) {
		t.Errorf("rendered bytes differ between runs (lens %d vs %d) — non-determinism still present", len(first), len(second))
	}
}

// TestRenderDifferentClockProducesDifferentBytes guards against the
// inverse: two distinct fixed clocks must produce distinct PDFs, so
// the seam is actually wired and not silently ignored.
func TestRenderDifferentClockProducesDifferentBytes(t *testing.T) {
	build := func(at time.Time) []byte {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			SetCreationDate(at).
			Paragraph(kardec.Text("body"))
		out, err := Bytes(doc.Document)
		if err != nil {
			t.Fatalf("Bytes: %v", err)
		}
		return out
	}

	a := build(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := build(time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC))
	if bytes.Equal(a, b) {
		t.Error("different clocks should produce different PDF bytes")
	}
}

// TestCreationDateFlowsIntoInfoDict checks the literal stamp shows up
// in the rendered byte stream, so the reproducibility test above is
// not just asserting "same bytes for unrelated reasons".
func TestCreationDateFlowsIntoInfoDict(t *testing.T) {
	at := time.Date(2026, 5, 7, 14, 23, 45, 0, time.UTC)
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetCreationDate(at).
		Paragraph(kardec.Text("body"))
	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	want := "(D:20260507142345Z)"
	if !bytes.Contains(out, []byte(want)) {
		t.Errorf("expected fixed CreationDate %q in PDF; not found", want)
	}
}
