package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestTable_ColspanCellGetsMergedWidth(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Table().
		Columns(kardec.Col("Q1"), kardec.Col("Q2"), kardec.Col("Q3"), kardec.Col("Q4")).
		RowCells(kardec.SpanCell(2, kardec.Text("First half")), kardec.SpanCell(2, kardec.Text("Second half"))).
		RowCells(kardec.NewCell(kardec.Text("100")), kardec.NewCell(kardec.Text("110")), kardec.NewCell(kardec.Text("120")), kardec.NewCell(kardec.Text("130"))).
		Build()

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	var firstHalf, hundred *PlacedItem
	for i := range pages[0].Items {
		it := &pages[0].Items[i]
		switch it.Text {
		case "First":
			firstHalf = it
		case "100":
			hundred = it
		}
	}
	if firstHalf == nil || hundred == nil {
		t.Fatalf("expected to find both 'First' and '100' tokens; firstHalf=%v hundred=%v",
			firstHalf, hundred)
	}
	// Both should sit at the same X (left edge of column 0); a non-
	// spanning header cell would appear at a quarter of the row width
	// instead of the half-width offset.
	if firstHalf.X.Points() != hundred.X.Points() {
		t.Errorf("spanning header X=%v should match column-0 cell X=%v",
			firstHalf.X.Points(), hundred.X.Points())
	}
}

func TestTable_ColspanClampsAtTableEdge(t *testing.T) {
	// Span asks for 5 columns but the table has only 3 — clamp.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Table().
		Columns(kardec.Col("A"), kardec.Col("B"), kardec.Col("C")).
		RowCells(kardec.SpanCell(5, kardec.Text("clamped"))).
		Build()

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	for _, it := range pages[0].Items {
		if it.Text == "clamped" && it.X.Points() < 0 {
			t.Errorf("clamped cell rendered at negative X: %v", it.X.Points())
		}
	}
}

func TestTable_ColspanWithBordersAllEmitsAtCellBoundaries(t *testing.T) {
	// 4 columns, header row with one 4-span cell. BordersAll should
	// produce exactly 2 vertical lines (left + right of the merged
	// header), not 5 (one between every column).
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Table().
		Columns(kardec.Col("A"), kardec.Col("B"), kardec.Col("C"), kardec.Col("D")).
		Borders(kardec.BordersAll).
		RowCells(kardec.SpanCell(4, kardec.Text("merged"))).
		Build()

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	verticals := 0
	for _, it := range pages[0].Items {
		if it.Rect != nil && it.Rect.Width.Points() < 1.0 && it.Rect.Thickness.Points() > 1.0 {
			verticals++
		}
	}
	if verticals != 2 {
		t.Errorf("merged header row should produce 2 vertical lines, got %d", verticals)
	}
}
