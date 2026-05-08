package kardec

import "testing"

// TestDocumentMathFont covers the lazy-load + memoisation contract on the
// public adapter: the first call returns a non-nil MathFont, the second
// call returns the same instance, and the document's deferred-error chain
// stays clean.
func TestDocumentMathFont(t *testing.T) {
	d := New(PageA4, MarginsNormal)
	mf := d.MathFont()
	if mf == nil {
		t.Fatalf("MathFont returned nil")
	}
	if d.Err() != nil {
		t.Fatalf("MathFont set Err: %v", d.Err())
	}
	if mf2 := d.MathFont(); mf2 != mf {
		t.Errorf("MathFont not memoised: %p != %p", mf, mf2)
	}
}

// TestDocumentMathFont_GlyphLookup spot-checks one Greek letter and one
// big operator through the public surface so callers see the same data
// the layout track will see.
func TestDocumentMathFont_GlyphLookup(t *testing.T) {
	d := New(PageA4, MarginsNormal)
	mf := d.MathFont()
	if mf == nil {
		t.Fatalf("MathFont returned nil")
	}
	g, ok := mf.GlyphFor("\\alpha")
	if !ok {
		t.Fatalf("GlyphFor(\\alpha): not resolved via Document")
	}
	if g.Rune != 'α' {
		t.Errorf("GlyphFor(\\alpha).Rune = %U; want %U", g.Rune, 'α')
	}
	if w := mf.Measure(g, 12); w <= 0 {
		t.Errorf("Measure(\\alpha) = %v; want > 0", w)
	}

	if g, ok := mf.GlyphFor("\\sum"); !ok || g.Rune != '∑' {
		t.Errorf("GlyphFor(\\sum) = (%U, %v); want (∑, true)", g.Rune, ok)
	}
}
