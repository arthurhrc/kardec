package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

// fixedFont is a deterministic FontProvider used by integration tests:
// every glyph advances 6 points at 12pt and scales linearly with size.
// Ascent/descent are 0.75/0.25 of the size respectively. Predictable
// metrics let tests assert layout outcomes without TTF parsing.
type fixedFont struct{}

func (fixedFont) Resolve(string, bool, bool) Font { return fixedFontFace{} }

type fixedFontFace struct{}

func (fixedFontFace) Measure(text string, sizePt float64) (float64, float64, float64) {
	return float64(len(text)) * sizePt * 0.5, sizePt * 0.75, sizePt * 0.25
}

// TestDefineStyleChangesHeadingSize verifies that user-supplied style
// definitions flow through ResolveBlockStyle into the layout engine.
// Two documents with the same content but different H1 sizes must
// produce text items at different point sizes.
func TestDefineStyleChangesHeadingSize(t *testing.T) {
	build := func(h1Size kardec.Length) []PlacedItem {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
		doc.DefineStyle(kardec.StyleH1, kardec.Style{Size: h1Size, ParentStyle: kardec.StyleH1})
		doc.Heading(1, kardec.Text("Title"))
		pages, err := Layout(doc, fixedFont{})
		if err != nil {
			t.Fatalf("layout: %v", err)
		}
		if len(pages) != 1 || len(pages[0].Items) == 0 {
			t.Fatalf("want 1 page with items, got %+v", pages)
		}
		return pages[0].Items
	}

	small := build(kardec.Pt(14))
	large := build(kardec.Pt(48))

	if small[0].Size == large[0].Size {
		t.Errorf("DefineStyle did not change heading size: small=%v large=%v",
			small[0].Size, large[0].Size)
	}
	if small[0].Size.Points() != 14 {
		t.Errorf("small heading size = %v pt, want 14", small[0].Size.Points())
	}
	if large[0].Size.Points() != 48 {
		t.Errorf("large heading size = %v pt, want 48", large[0].Size.Points())
	}
}

// TestDefineStyleChangesParagraphColor verifies that DefineStyle on the
// Default style propagates to paragraph runs by inspecting placed item
// colors.
func TestDefineStyleChangesParagraphColor(t *testing.T) {
	red := kardec.Color{R: 255, G: 0, B: 0}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.DefineStyle(kardec.StyleDefault, kardec.Style{Color: red, Size: kardec.Pt(11)})
	doc.Paragraph(kardec.Text("colored body"))

	pages, err := Layout(doc, fixedFont{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) == 0 || len(pages[0].Items) == 0 {
		t.Fatalf("want at least one item, got %+v", pages)
	}
	if got := pages[0].Items[0].Color; got != red {
		t.Errorf("paragraph color = %+v, want %+v", got, red)
	}
}

// TestWithStyleOverridesNamedStyle verifies that the per-block override
// path (Paragraph.WithStyle / Heading.WithStyle) takes precedence over
// the named style — matching the priority chain in RFC-001 §6.
func TestWithStyleOverridesNamedStyle(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.DefineStyle(kardec.StyleH1, kardec.Style{Size: kardec.Pt(20), ParentStyle: kardec.StyleH1})
	doc.AddHeading(1, kardec.Text("Override")).
		WithStyle(kardec.Style{Size: kardec.Pt(40)}).
		Done()

	pages, err := Layout(doc, fixedFont{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if len(pages) == 0 || len(pages[0].Items) == 0 {
		t.Fatalf("want items, got %+v", pages)
	}
	if got := pages[0].Items[0].Size.Points(); got != 40 {
		t.Errorf("inline override size = %v, want 40", got)
	}
}
