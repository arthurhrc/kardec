package render

import (
	"testing"
	"time"

	"github.com/arthurhrc/kardec"
)

func TestSubsetFontsShrinksRenderedPDF(t *testing.T) {
	build := func(subset bool) int {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			SetCreationDate(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
		if subset {
			doc.EnableFontSubsetting()
		}
		doc.Heading(1, kardec.Text("Hello, Kardec")).
			Paragraph(
				kardec.Text("Subsetting drops glyph data the document never references, "),
				kardec.Text("so the rendered PDF compresses to a fraction of the full font."),
			)
		out, err := Bytes(doc)
		if err != nil {
			t.Fatalf("Bytes: %v", err)
		}
		return len(out)
	}
	full := build(false)
	subsetted := build(true)
	if subsetted >= full {
		t.Errorf("expected subset PDF to be smaller; full=%d subsetted=%d", full, subsetted)
	}
	// Savings should be significant — at least 30 percent below the
	// baseline. Looser threshold than the typical 70-80 percent
	// reality but resilient to default-font changes.
	if subsetted >= full*7/10 {
		t.Errorf("subset savings under 30 percent: full=%d subsetted=%d", full, subsetted)
	}
}

func TestSubsetFontsOffByDefault(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("body"))
	if doc.FontSubsetEnabled() {
		t.Errorf("subsetFonts must be off by default")
	}
}

// TestFontSubsettingToggles exercises the EnableFontSubsetting /
// DisableFontSubsetting pair that replaced the v0.x variadic-bool
// SubsetFonts at v1.0.
func TestFontSubsettingToggles(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		EnableFontSubsetting()
	if !doc.FontSubsetEnabled() {
		t.Errorf("EnableFontSubsetting should turn the flag on")
	}
	doc.DisableFontSubsetting()
	if doc.FontSubsetEnabled() {
		t.Errorf("DisableFontSubsetting should turn the flag off")
	}
}
