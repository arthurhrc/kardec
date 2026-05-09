package layout

import (
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestCrossref_RefPagePlaceholderResolvesToAnchorPage(t *testing.T) {
	// Section that fills page 1, drops a labeled-table anchor on
	// page 2, then references it via RefPage from page 3-content.
	// Use Anchor + Table-ish stand-ins through the public builder
	// so the anchor name aligns with what doc.Ref / doc.RefPage
	// would consult in real use.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	for i := 0; i < 60; i++ {
		doc.Paragraph(kardec.Text("filler line"))
	}
	doc.Anchor(kardec.RefAnchorName("rev"))
	doc.Paragraph(kardec.Text("anchor home"))
	for i := 0; i < 60; i++ {
		doc.Paragraph(kardec.Text("more filler"))
	}
	doc.Paragraph(kardec.Text("page is "), kardec.Text(kardec.RefPagePlaceholder+"rev}}"))

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("expected the doc to span at least two pages, got %d", len(pages))
	}

	// Locate which page hosts the anchor.
	anchorPage := -1
	for i, p := range pages {
		for _, a := range p.Anchors {
			if a.Name == kardec.RefAnchorName("rev") {
				anchorPage = i + 1
			}
		}
	}
	if anchorPage <= 0 {
		t.Fatalf("anchor not found on any page")
	}

	// Confirm no remaining placeholder anywhere.
	for _, p := range pages {
		for _, it := range p.Items {
			if strings.Contains(it.Text, kardec.RefPagePlaceholder) {
				t.Errorf("post-pass left a placeholder behind: %q", it.Text)
			}
		}
	}

	// Confirm the resolved page number is somewhere in the output.
	want := itoa(anchorPage)
	found := false
	for _, p := range pages {
		for _, it := range p.Items {
			if it.Text == want {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected an item with text %q somewhere in the output", want)
	}
}

func TestCrossref_UnknownLabelLeavesQuestionMark(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text(kardec.RefPagePlaceholder + "ghost}}")).Document

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	got := false
	for _, p := range pages {
		for _, it := range p.Items {
			if it.Text == "?" {
				got = true
			}
		}
	}
	if !got {
		t.Errorf("unresolved RefPage placeholder should resolve to '?'")
	}
}
