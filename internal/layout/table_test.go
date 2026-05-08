package layout

import (
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

// uniformFontProvider returns a font where every character advances 6pt
// at any size (linear scaling). Predictable widths let tests assert on
// the column geometry without depending on real TTF metrics.
type uniformFontProvider struct{}

func (uniformFontProvider) Resolve(string, bool, bool) Font { return uniformFont{} }

type uniformFont struct{}

func (uniformFont) Measure(text string, sizePt float64) (float64, float64, float64) {
	return float64(len(text)) * sizePt * 0.5, sizePt * 0.75, sizePt * 0.25
}

func TestComputeColumnWidthsFractional(t *testing.T) {
	cols := []kardec.Column{
		{Width: 0.4},
		{Width: 0.3},
		{Width: 0.3},
	}
	got := computeColumnWidths(cols, 100)
	want := []float64{40, 30, 30}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("col %d width = %v, want %v", i, got[i], w)
		}
	}
}

func TestComputeColumnWidthsFixedAndAuto(t *testing.T) {
	cols := []kardec.Column{
		{Width: 50}, // fixed
		{Width: 0},  // auto
		{Width: 0},  // auto
	}
	got := computeColumnWidths(cols, 200)
	if got[0] != 50 {
		t.Errorf("fixed col = %v, want 50", got[0])
	}
	// remaining 150 / 2 unsized cols
	if got[1] != 75 || got[2] != 75 {
		t.Errorf("auto cols = %v, want [75, 75]", got[1:])
	}
}

func TestPlaceTableEmitsItemsForEachCell(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Table().
		Columns(kardec.Col("Mes"), kardec.Col("Receita")).
		Row("Janeiro", "1000").
		Row("Fevereiro", "1500").
		Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(pages))
	}

	var seen []string
	for _, item := range pages[0].Items {
		seen = append(seen, item.Text)
	}
	joined := strings.Join(seen, "|")
	for _, expected := range []string{"Janeiro", "1000", "Fevereiro", "1500"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("expected %q in placed items %q", expected, joined)
		}
	}
}

func TestPlaceTableSplitsAcrossPages(t *testing.T) {
	doc := kardec.New(kardec.PageA5, kardec.MarginsNormal)
	tbl := doc.Table().Columns(kardec.Col("A"), kardec.Col("B"))
	for i := 0; i < 80; i++ {
		tbl.Row("data-A", "data-B")
	}
	tbl.Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("80-row table should span multiple A5 pages, got %d", len(pages))
	}
}

func TestPlaceTableRepeatHeader(t *testing.T) {
	doc := kardec.New(kardec.PageA5, kardec.MarginsNormal)
	tbl := doc.Table().
		Columns(kardec.Col("HeaderA"), kardec.Col("HeaderB")).
		RepeatHeader().
		Row("HEADER1", "HEADER2") // row 0 = header
	for i := 0; i < 80; i++ {
		tbl.Row("body-A", "body-B")
	}
	tbl.Build()

	pages, err := Layout(doc, uniformFontProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("expected >=2 pages, got %d", len(pages))
	}
	// Each page must contain HEADER1 — verify by counting distinct pages
	// where the header text appears.
	pagesWithHeader := 0
	for _, p := range pages {
		for _, item := range p.Items {
			if item.Text == "HEADER1" {
				pagesWithHeader++
				break
			}
		}
	}
	if pagesWithHeader < 2 {
		t.Errorf("RepeatHeader should reprint header on every page, found on %d pages", pagesWithHeader)
	}
}

func TestPlaceTableEmptyColumnsTriggersDeferredError(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Table().Row("a", "b").Build()
	if err := doc.Err(); err == nil {
		t.Error("Build with no columns should record a deferred error")
	}
}
