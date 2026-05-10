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

// TestSubsetFontsOptIn exercises the deprecated SubsetFonts(...)
// variadic-bool form. The replacement is EnableFontSubsetting /
// DisableFontSubsetting (covered by toggles_test.go in the root
// package); this test remains so the deprecated path keeps working
// for v0.x consumers until v1.0 removes it.
func TestSubsetFontsOptIn(t *testing.T) {
	//lint:ignore SA1019 testing deprecated path
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SubsetFonts(true)
	if !doc.FontSubsetEnabled() {
		t.Errorf("SubsetFonts(true) should enable")
	}
	//lint:ignore SA1019 testing deprecated path
	doc.SubsetFonts(false)
	if doc.FontSubsetEnabled() {
		t.Errorf("SubsetFonts(false) should disable")
	}
	//lint:ignore SA1019 testing deprecated path
	doc.SubsetFonts()
	if !doc.FontSubsetEnabled() {
		t.Errorf("SubsetFonts() with no args should enable")
	}
}
