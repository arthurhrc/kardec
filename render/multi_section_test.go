package render

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestRenderMultiSectionMixesPageSizes(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Cover")).
		NewSection(kardec.PageSetup{
			Size:        kardec.PageA4,
			Orientation: kardec.Landscape,
			Margins:     kardec.MarginsNormal,
		}).
		Heading(2, kardec.Text("Wide chart")).
		Paragraph(kardec.Text("body"))

	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	// The first /MediaBox should reflect portrait A4 (~595 × 842),
	// the second landscape A4 (~842 × 595). We assert both literal
	// dimension pairs appear in the byte stream.
	wantPortrait := []byte("/MediaBox [0 0 595.2756 841.8898]")
	wantLandscape := []byte("/MediaBox [0 0 841.8898 595.2756]")
	if !bytes.Contains(out, wantPortrait) {
		t.Errorf("expected portrait A4 MediaBox in PDF byte stream")
	}
	if !bytes.Contains(out, wantLandscape) {
		t.Errorf("expected landscape A4 MediaBox in PDF byte stream")
	}
}
