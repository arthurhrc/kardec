package kardec

import (
	"errors"
	"strings"
	"testing"
)

func TestMergeStyleChildOverridesParent(t *testing.T) {
	parent := Style{
		Family:      "Parent Sans",
		Size:        Pt(11),
		Weight:      WeightRegular,
		Color:       ColorBlack,
		SpaceBefore: Pt(0),
		SpaceAfter:  Pt(6),
		LineHeight:  1.15,
		Alignment:   AlignLeft,
	}
	child := Style{
		Size:       Pt(20),
		Weight:     WeightBold,
		Color:      HexColor("#FF0000"),
		LineHeight: 1.4,
	}

	got := mergeStyle(child, parent)

	if got.Family != "Parent Sans" {
		t.Errorf("Family should fall through, got %q", got.Family)
	}
	if got.Size != Pt(20) {
		t.Errorf("Size should be overridden to 20pt, got %v", got.Size)
	}
	if got.Weight != WeightBold {
		t.Errorf("Weight should be Bold, got %v", got.Weight)
	}
	if got.Color != (Color{R: 0xFF}) {
		t.Errorf("Color should be red, got %+v", got.Color)
	}
	if got.LineHeight != 1.4 {
		t.Errorf("LineHeight should be 1.4, got %v", got.LineHeight)
	}
	// Fields child left unset must come from parent.
	if got.SpaceAfter != Pt(6) {
		t.Errorf("SpaceAfter should fall through to 6pt, got %v", got.SpaceAfter)
	}
	if got.Alignment != AlignLeft {
		t.Errorf("Alignment should fall through to AlignLeft, got %v", got.Alignment)
	}
}

func TestMergeStyleAdditiveBooleansSurvive(t *testing.T) {
	parent := Style{KeepWithNext: false, KeepTogether: false}
	child := Style{KeepWithNext: true, PageBreakBefore: true}

	got := mergeStyle(child, parent)
	if !got.KeepWithNext || !got.PageBreakBefore {
		t.Errorf("additive booleans should propagate, got %+v", got)
	}
	if got.KeepTogether {
		t.Errorf("KeepTogether should remain false")
	}
}

func TestBuiltinStylesContainsAllExpectedNames(t *testing.T) {
	want := []string{
		StyleDefault, StyleH1, StyleH2, StyleH3, StyleH4, StyleH5, StyleH6,
		StyleCaption, StyleQuote, StyleCode,
		StyleTableHeader, StyleTableCell,
		StyleFooter, StyleHeader, StyleListItem, StyleLink,
	}
	got := BuiltinStyles()
	if len(got) < len(want) {
		t.Fatalf("BuiltinStyles size = %d, want >= %d", len(got), len(want))
	}
	for _, name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("BuiltinStyles missing %q", name)
		}
	}
}

func TestBuiltinStylesReturnsFreshMap(t *testing.T) {
	a := BuiltinStyles()
	a[StyleH1] = Style{Family: "tampered"}
	b := BuiltinStyles()
	if b[StyleH1].Family == "tampered" {
		t.Error("BuiltinStyles should return an independent map per call")
	}
}

func TestResolveStyleFallsThroughToDefault(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	got := doc.ResolveStyle("does-not-exist")

	if got.Family != DefaultStyle.Family {
		t.Errorf("unknown style should resolve to Default family, got %q", got.Family)
	}
	if got.Size != DefaultStyle.Size {
		t.Errorf("unknown style should resolve to Default size, got %v", got.Size)
	}
}

func TestResolveStyleInheritanceChain(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.DefineStyle("MyBase", Style{
		Family: "Base Sans",
		Size:   Pt(13),
		Color:  HexColor("#112233"),
	})
	doc.DefineStyle("MyChild", Style{
		ParentStyle: "MyBase",
		Weight:      WeightBold,
		LineHeight:  1.6,
	})

	got := doc.ResolveStyle("MyChild")

	if got.Family != "Base Sans" {
		t.Errorf("Family should inherit from MyBase, got %q", got.Family)
	}
	if got.Size != Pt(13) {
		t.Errorf("Size should inherit from MyBase, got %v", got.Size)
	}
	if got.Weight != WeightBold {
		t.Errorf("Weight should be Bold from MyChild, got %v", got.Weight)
	}
	if got.LineHeight != 1.6 {
		t.Errorf("LineHeight should be 1.6 from MyChild, got %v", got.LineHeight)
	}
	if got.Color != (Color{R: 0x11, G: 0x22, B: 0x33}) {
		t.Errorf("Color should inherit from MyBase, got %+v", got.Color)
	}
	if doc.Err() != nil {
		t.Errorf("clean chain should not surface an error, got %v", doc.Err())
	}
}

func TestResolveStyleCycleDetectionCapturesError(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.DefineStyle("A", Style{ParentStyle: "B"})
	doc.DefineStyle("B", Style{ParentStyle: "A"})

	_ = doc.ResolveStyle("A")

	if doc.Err() == nil {
		t.Fatal("cycle should capture an error via doc.fail")
	}
	if !strings.Contains(doc.Err().Error(), "style cycle") {
		t.Errorf("error should mention style cycle, got %q", doc.Err().Error())
	}
	// Sanity: the error type is the generic errors.New wrapper.
	if !errors.Is(doc.Err(), doc.Err()) {
		t.Error("errors.Is on the captured error should be reflexive")
	}
}

func TestDefineStyleOverridesBuiltin(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	custom := Style{
		Family:     "Custom",
		Size:       Pt(40),
		Weight:     WeightBlack,
		Color:      HexColor("#FF00FF"),
		LineHeight: 1.0,
	}
	doc.DefineStyle(StyleH1, custom)

	got := doc.ResolveStyle(StyleH1)
	if got.Size != Pt(40) {
		t.Errorf("overridden H1 size = %v, want 40pt", got.Size)
	}
	if got.Family != "Custom" {
		t.Errorf("overridden H1 family = %q, want Custom", got.Family)
	}
	if got.Color != (Color{R: 0xFF, G: 0x00, B: 0xFF}) {
		t.Errorf("overridden H1 color = %+v, want magenta", got.Color)
	}
}

func TestResolveBlockStyleHeadingMapsToLevelStyle(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	h2 := Heading{level: 2, runs: []Run{Text("hi")}}

	got := doc.ResolveBlockStyle(h2)
	want := doc.ResolveStyle(StyleH2)

	if got.Size != want.Size {
		t.Errorf("Heading(2) size = %v, want %v (StyleH2)", got.Size, want.Size)
	}
	if got.Color != want.Color {
		t.Errorf("Heading(2) color = %+v, want %+v", got.Color, want.Color)
	}
	if got.Weight != want.Weight {
		t.Errorf("Heading(2) weight = %v, want %v", got.Weight, want.Weight)
	}
}

func TestResolveBlockStyleParagraphHonorsExplicitOverride(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.DefineStyle("Note", Style{Color: HexColor("#00AA00"), LineHeight: 1.3})

	p := Paragraph{runs: []Run{Text("hi")}}
	p = p.WithNamedStyle("Note").WithStyle(Style{Size: Pt(15)})

	got := doc.ResolveBlockStyle(p)

	if got.Size != Pt(15) {
		t.Errorf("inline override size = %v, want 15pt", got.Size)
	}
	if got.Color != (Color{R: 0x00, G: 0xAA, B: 0x00}) {
		t.Errorf("named style color = %+v, want green", got.Color)
	}
	if got.LineHeight != 1.3 {
		t.Errorf("named style LineHeight = %v, want 1.3", got.LineHeight)
	}
}

func TestParagraphBuilderCommitsViaDone(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		AddParagraph(Text("body")).
		WithNamedStyle("Quote").
		LineHeight(1.5).
		Justify().
		Done().
		Heading(1, Text("after"))

	blocks := doc.cur.Blocks
	if len(blocks) != 2 {
		t.Fatalf("want 2 blocks (paragraph + heading), got %d", len(blocks))
	}
	p, ok := blocks[0].(Paragraph)
	if !ok {
		t.Fatalf("first block should be Paragraph, got %T", blocks[0])
	}
	if p.styleName != "Quote" {
		t.Errorf("named style = %q, want Quote", p.styleName)
	}
	if p.lineHeight != 1.5 {
		t.Errorf("line height = %v, want 1.5", p.lineHeight)
	}
	if p.alignment != AlignJustify {
		t.Errorf("alignment = %v, want Justify", p.alignment)
	}
}

func TestHeadingBuilderWithStyleOverridesLevelDefault(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.AddHeading(3, Text("section")).
		WithStyle(Style{Size: Pt(20), Color: HexColor("#FF0000")}).
		Done()

	h := doc.cur.Blocks[0].(Heading)
	if !h.hasStyle {
		t.Fatal("Heading should carry an inline style override")
	}

	got := doc.ResolveBlockStyle(h)
	if got.Size != Pt(20) {
		t.Errorf("override size = %v, want 20pt", got.Size)
	}
	if got.Color != (Color{R: 0xFF}) {
		t.Errorf("override color = %+v, want red", got.Color)
	}
	// Anything not overridden falls through to H3 defaults.
	wantH3 := doc.ResolveStyle(StyleH3)
	if got.Weight != wantH3.Weight {
		t.Errorf("Weight should fall through from H3, got %v want %v", got.Weight, wantH3.Weight)
	}
}
