package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

// findText walks a single page's items and reports whether any item's
// Text equals s. Used to assert which page a piece of content landed on.
func findText(p Page, s string) bool {
	for _, it := range p.Items {
		if it.Text == s {
			return true
		}
	}
	return false
}

func TestKeepTogether_PairFitsOnSamePage(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		KeepTogether(
			kardec.NewHeading(2, kardec.Text("Section")),
			kardec.NewParagraph(kardec.Text("body of the section")),
		)
	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if !findText(pages[0], "Section") {
		t.Errorf("heading missing on page 0")
	}
	if !findText(pages[0], "section") {
		t.Errorf("paragraph word missing on page 0")
	}
}

func TestKeepTogether_PushesGroupToNextPageWhenItWouldSplit(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	// Fill the first page with enough body paragraphs that the
	// KeepTogether group cannot fit at the end. A4 + Normal margins
	// gives roughly 720pt of vertical content area; we want the
	// cursor near the bottom but with some space left — enough for
	// just the heading line, but not for the heading + paragraph
	// pair the keep-together asks to bind.
	for i := 0; i < 60; i++ {
		doc.Paragraph(kardec.Text("filler line that takes a single rendered line"))
	}
	doc.KeepTogether(
		kardec.NewHeading(2, kardec.Text("KeptHeading")),
		kardec.NewParagraph(kardec.Text("paired-paragraph-body")),
	)
	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("expected the group to push to a new page, got %d page(s)", len(pages))
	}
	// Heading and paragraph must end up on the same page.
	var headingPage, bodyPage = -1, -1
	for i, p := range pages {
		if findText(p, "KeptHeading") {
			headingPage = i
		}
		if findText(p, "paired-paragraph-body") {
			bodyPage = i
		}
	}
	if headingPage == -1 || bodyPage == -1 {
		t.Fatalf("could not locate heading (%d) or body (%d) on any page", headingPage, bodyPage)
	}
	if headingPage != bodyPage {
		t.Errorf("KeepTogether broke: heading on page %d, body on page %d", headingPage, bodyPage)
	}
}

func TestKeepTogether_OversizedGroupDegradesGracefully(t *testing.T) {
	// A group that is itself taller than a full page must not loop;
	// the engine flushes once and then lets the inner blocks overflow
	// onto further pages without re-applying the keep-together rule.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("preamble"))
	inner := make([]kardec.Block, 0, 80)
	for i := 0; i < 80; i++ {
		inner = append(inner, kardec.NewParagraph(kardec.Text("oversized-group-line")))
	}
	doc.KeepTogether(inner...)

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) < 2 {
		t.Errorf("oversized group should span multiple pages, got %d", len(pages))
	}
}

func TestKeepTogether_EmptyGroupIsNoOp(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		KeepTogether().
		Paragraph(kardec.Text("after"))
	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if !findText(pages[0], "after") {
		t.Errorf("paragraph after empty KeepTogether is missing")
	}
}
