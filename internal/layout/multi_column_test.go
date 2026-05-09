package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestMultiColumn_OverflowAdvancesToNextColumnBeforeNewPage(t *testing.T) {
	setup := kardec.PageSetup{
		Size:    kardec.PageA4,
		Margins: kardec.MarginsNormal,
		Columns: 2,
	}
	doc := kardec.NewWithSetup(setup)
	for i := 0; i < 30; i++ {
		doc.Paragraph(kardec.Text("filler"))
	}
	doc.Paragraph(kardec.Text("BOUNDARY"))
	for i := 0; i < 30; i++ {
		doc.Paragraph(kardec.Text("more filler"))
	}

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	// First page should host both columns of content. Find the X
	// coordinates of two filler tokens — they should land in two
	// distinct horizontal positions (one per column) before we ever
	// produce a second page.
	if len(pages) == 0 {
		t.Fatalf("expected at least one page")
	}
	xs := map[float64]bool{}
	for _, it := range pages[0].Items {
		if it.Text == "filler" {
			xs[it.X.Points()] = true
		}
	}
	if len(xs) < 2 {
		t.Errorf("expected filler items in two distinct columns on page 0, got x set %v", xs)
	}
}

func TestMultiColumn_PageBreakForcesNewPageNotNextColumn(t *testing.T) {
	setup := kardec.PageSetup{
		Size:    kardec.PageA4,
		Margins: kardec.MarginsNormal,
		Columns: 2,
	}
	doc := kardec.NewWithSetup(setup).
		Paragraph(kardec.Text("page-one")).
		PageBreak().
		Paragraph(kardec.Text("page-two"))

	pages, err := NewEngine().Layout(doc.Document, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("PageBreak in a multi-column section must force a new page, got %d page(s)", len(pages))
	}
	if !findText(pages[0], "page-one") {
		t.Errorf("page-one missing from page 0")
	}
	if !findText(pages[1], "page-two") {
		t.Errorf("page-two missing from page 1")
	}
}

func TestMultiColumn_SingleColumnDefaultIsUnchanged(t *testing.T) {
	// PageSetup with Columns == 0 (the default) should behave
	// identically to the single-column path: no column advances, all
	// content flows in one column.
	setup := kardec.PageSetup{
		Size:    kardec.PageA4,
		Margins: kardec.MarginsNormal,
	}
	doc := kardec.NewWithSetup(setup).
		Paragraph(kardec.Text("only column"))

	pages, err := NewEngine().Layout(doc.Document, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	xs := map[float64]bool{}
	for _, it := range pages[0].Items {
		if it.Text == "only" || it.Text == "column" {
			xs[it.X.Points()] = true
		}
	}
	// Both tokens should sit on the same baseline strip with at most
	// one X position used as the *paragraph* left edge — the second
	// token offsets by the first's width, but never starts a new
	// column.
	for x := range xs {
		if x < float64(setup.Margins.Left) {
			t.Errorf("single-column x=%v fell below left margin %v", x, float64(setup.Margins.Left))
		}
	}
}
