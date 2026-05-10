package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

// countRectItems returns the number of PlacedItems on the page that
// carry a Rect payload (shading or border).
func countRectItems(p Page) int {
	n := 0
	for _, it := range p.Items {
		if it.Rect != nil {
			n++
		}
	}
	return n
}

func TestPlaceTableNoBordersEmitsNoRects(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Table().
		Columns(kardec.Col("A"), kardec.Col("B")).
		Row("a1", "b1").
		Row("a2", "b2").
		Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if got := countRectItems(pages[0]); got != 0 {
		t.Errorf("BordersNone + no shading should emit 0 rects, got %d", got)
	}
}

func TestPlaceTableHorizontalBordersEmitsLines(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Table().
		Columns(kardec.Col("A"), kardec.Col("B")).
		Borders(kardec.TableBordersHorizontal).
		Row("a1", "b1").
		Row("a2", "b2").
		Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	// Two rows -> top border per row + final bottom = 3 horizontals.
	if got := countRectItems(pages[0]); got < 3 {
		t.Errorf("BordersHorizontal with 2 rows should emit at least 3 rect items, got %d", got)
	}
}

func TestPlaceTableAllBordersEmitsGrid(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Table().
		Columns(kardec.Col("A"), kardec.Col("B"), kardec.Col("C")).
		Borders(kardec.TableBordersAll).
		Row("a", "b", "c").
		Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	// 1 row + 1 final-bottom horizontal + 4 vertical column boundaries (3 cols → 4 lines)
	if got := countRectItems(pages[0]); got < 6 {
		t.Errorf("BordersAll with 1 row × 3 cols should emit at least 6 rects, got %d", got)
	}
}

func TestPlaceTableHeaderShadingEmitsBackgroundRect(t *testing.T) {
	red := kardec.Color{R: 200, G: 50, B: 50}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Table().
		Columns(kardec.Col("A"), kardec.Col("B")).
		HeaderShading(red).
		Row("h1", "h2").
		Row("a", "b").
		Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	var sawRed bool
	for _, it := range pages[0].Items {
		if it.Rect != nil && it.Rect.Color == red {
			sawRed = true
			break
		}
	}
	if !sawRed {
		t.Errorf("HeaderShading should emit a rect with the configured color")
	}
}

func TestPlaceTableAlternateRowShadingSkipsHeader(t *testing.T) {
	gray := kardec.Color{R: 240, G: 240, B: 240}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Table().
		Columns(kardec.Col("A"), kardec.Col("B")).
		AlternateRowShading(gray).
		Row("h1", "h2"). // row 0 — should NOT be shaded
		Row("a1", "b1"). // row 1 — shaded (odd)
		Row("a2", "b2"). // row 2 — not shaded (even)
		Row("a3", "b3"). // row 3 — shaded (odd)
		Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	count := 0
	for _, it := range pages[0].Items {
		if it.Rect != nil && it.Rect.Color == gray {
			count++
		}
	}
	if count != 2 {
		t.Errorf("AlternateRowShading with 4 rows (1 header + 3 body) should shade 2 rows, got %d", count)
	}
}
