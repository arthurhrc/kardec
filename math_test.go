package kardec

import "testing"

func TestDocumentMathAppendsBlock(t *testing.T) {
	doc := New(PageA4, MarginsNormal).Math(`a^2 + b^2 = c^2`)
	if err := doc.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("want 1 block, got %d", len(blocks))
	}
	m, ok := blocks[0].(Math)
	if !ok {
		t.Fatalf("first block should be Math, got %T", blocks[0])
	}
	if m.Source() != `a^2 + b^2 = c^2` {
		t.Errorf("Source = %q, want LaTeX source", m.Source())
	}
	if !m.Display() {
		t.Errorf("doc.Math should default to display style")
	}
}

func TestDocumentMathInlineSetsDisplayFalse(t *testing.T) {
	doc := New(PageA4, MarginsNormal).MathInline(`x^2`)
	m := doc.Sections()[0].Blocks[0].(Math)
	if m.Display() {
		t.Errorf("MathInline should set Display=false")
	}
}

func TestDocumentMathPreservesDeferredError(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errInternalForMathTest())
	doc.Math(`ignored`)
	if got := doc.Sections()[0].Blocks; len(got) != 0 {
		t.Errorf("Math should be inert after a captured error, got %d blocks", len(got))
	}
}

func errInternalForMathTest() error {
	return &mathSimpleErr{msg: "synthetic"}
}

type mathSimpleErr struct{ msg string }

func (e *mathSimpleErr) Error() string { return e.msg }
