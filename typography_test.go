package kardec

import (
	"testing"
)

func TestNewDocumentLoadsBuiltinFonts(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	if doc.Err() != nil {
		t.Fatalf("New produced error: %v", doc.Err())
	}
	if doc.fonts == nil {
		t.Fatal("New should instantiate the font registry")
	}
	if doc.fonts.Default() == nil {
		t.Error("expected at least one builtin font registered as default")
	}
}

func TestMeasureTextOnBuiltinFamily(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	w, ok := doc.MeasureText("Hello", "Liberation Sans", Pt(12), WeightRegular, false)
	if !ok {
		t.Skip("Liberation Sans not bundled")
	}
	if w <= 0 {
		t.Errorf("MeasureText returned %g, want > 0", float64(w))
	}
}

func TestMeasureTextUnknownFamily(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	if _, ok := doc.MeasureText("x", "DefinitelyNotARealFont", Pt(12), WeightRegular, false); ok {
		t.Error("expected miss for unknown family")
	}
}

func TestRegisterFontRejectsEmptyBytes(t *testing.T) {
	doc := New(PageA4, MarginsNormal).RegisterFont("Custom", WeightRegular, false, nil)
	if doc.Err() == nil {
		t.Error("RegisterFont with nil bytes should populate Err")
	}
}
