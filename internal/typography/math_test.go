package typography

import (
	"testing"
	"unicode"
)

// TestLatinModernMath_Construct ensures the bundled OTF parses and the
// returned MathFont is non-nil.
func TestLatinModernMath_Construct(t *testing.T) {
	mf, err := LatinModernMath()
	if err != nil {
		t.Fatalf("LatinModernMath: %v", err)
	}
	if mf == nil {
		t.Fatalf("LatinModernMath returned nil MathFont")
	}
}

// TestLatinModernMath_GlyphFor_Greek covers the lowercase Greek alphabet —
// the fallback table MUST resolve every entry, and each glyph MUST report
// a positive advance and non-negative ascent/descent at a typical body
// size.
func TestLatinModernMath_GlyphFor_Greek(t *testing.T) {
	mf, err := LatinModernMath()
	if err != nil {
		t.Fatalf("LatinModernMath: %v", err)
	}
	cmds := []string{
		"\\alpha", "\\beta", "\\gamma", "\\delta", "\\epsilon",
		"\\zeta", "\\eta", "\\theta", "\\iota", "\\kappa",
		"\\lambda", "\\mu", "\\nu", "\\xi", "\\omicron",
		"\\pi", "\\rho", "\\sigma", "\\tau", "\\upsilon",
		"\\phi", "\\chi", "\\psi", "\\omega",
	}
	for _, c := range cmds {
		g, ok := mf.GlyphFor(c)
		if !ok {
			t.Errorf("GlyphFor(%q): not resolved", c)
			continue
		}
		if !unicode.Is(unicode.Greek, g.Rune) && !unicode.IsLetter(g.Rune) {
			t.Errorf("GlyphFor(%q) returned non-letter rune %U", c, g.Rune)
		}
		if w := mf.Measure(g, 12); w <= 0 {
			t.Errorf("Measure(%q) = %v; want > 0", c, w)
		}
		asc, desc := mf.AscentDescent(g, 12)
		if asc <= 0 {
			t.Errorf("AscentDescent(%q): ascent %v; want > 0", c, asc)
		}
		if desc < 0 {
			t.Errorf("AscentDescent(%q): descent %v; want >= 0", c, desc)
		}
	}
}

// TestLatinModernMath_GlyphFor_Operators covers the named big operators
// the layout track will use for \\sum-style atoms.
func TestLatinModernMath_GlyphFor_Operators(t *testing.T) {
	mf, err := LatinModernMath()
	if err != nil {
		t.Fatalf("LatinModernMath: %v", err)
	}
	cases := map[string]rune{
		"\\sum":  '∑',
		"\\int":  '∫',
		"\\prod": '∏',
	}
	for cmd, want := range cases {
		g, ok := mf.GlyphFor(cmd)
		if !ok {
			t.Errorf("GlyphFor(%q): not resolved", cmd)
			continue
		}
		if g.Rune != want {
			t.Errorf("GlyphFor(%q).Rune = %U; want %U", cmd, g.Rune, want)
		}
		if w := mf.Measure(g, 14); w <= 0 {
			t.Errorf("Measure(%q) = %v; want > 0", cmd, w)
		}
	}
}

// TestLatinModernMath_GlyphFor_Relations covers the relation set used by
// inequality / approximation atoms.
func TestLatinModernMath_GlyphFor_Relations(t *testing.T) {
	mf, err := LatinModernMath()
	if err != nil {
		t.Fatalf("LatinModernMath: %v", err)
	}
	cases := map[string]rune{
		"\\leq":    '≤',
		"\\geq":    '≥',
		"\\neq":    '≠',
		"\\approx": '≈',
		"\\to":     '→',
	}
	for cmd, want := range cases {
		g, ok := mf.GlyphFor(cmd)
		if !ok {
			t.Errorf("GlyphFor(%q): not resolved", cmd)
			continue
		}
		if g.Rune != want {
			t.Errorf("GlyphFor(%q).Rune = %U; want %U", cmd, g.Rune, want)
		}
	}
}

// TestLatinModernMath_GlyphFor_Passthrough verifies that ASCII letters,
// digits, and operators come through as their literal rune.
func TestLatinModernMath_GlyphFor_Passthrough(t *testing.T) {
	mf, err := LatinModernMath()
	if err != nil {
		t.Fatalf("LatinModernMath: %v", err)
	}
	literals := []string{
		"a", "z", "A", "Z",
		"0", "5", "9",
		"+", "-", "=", "(", ")",
	}
	for _, s := range literals {
		g, ok := mf.GlyphFor(s)
		if !ok {
			t.Errorf("GlyphFor(%q): not resolved", s)
			continue
		}
		if g.Rune != []rune(s)[0] {
			t.Errorf("GlyphFor(%q).Rune = %U; want %U", s, g.Rune, []rune(s)[0])
		}
		if w := mf.Measure(g, 12); w <= 0 {
			t.Errorf("Measure(%q) = %v; want > 0", s, w)
		}
	}
}

// TestLatinModernMath_GlyphFor_UnknownCommand asserts unknown LaTeX
// commands return false rather than degenerating into an empty glyph.
func TestLatinModernMath_GlyphFor_UnknownCommand(t *testing.T) {
	mf, err := LatinModernMath()
	if err != nil {
		t.Fatalf("LatinModernMath: %v", err)
	}
	if g, ok := mf.GlyphFor("\\notacommand"); ok {
		t.Errorf("GlyphFor unknown command returned ok=true (rune %U)", g.Rune)
	}
	if g, ok := mf.GlyphFor(""); ok {
		t.Errorf("GlyphFor(\"\") returned ok=true (rune %U)", g.Rune)
	}
}

// TestLatinModernMath_AscentDescent_Sane verifies ascent/descent are
// monotonic in size — doubling the size doubles both metrics.
func TestLatinModernMath_AscentDescent_Sane(t *testing.T) {
	mf, err := LatinModernMath()
	if err != nil {
		t.Fatalf("LatinModernMath: %v", err)
	}
	g, ok := mf.GlyphFor("\\alpha")
	if !ok {
		t.Fatalf("\\alpha did not resolve")
	}
	asc1, desc1 := mf.AscentDescent(g, 12)
	asc2, desc2 := mf.AscentDescent(g, 24)
	if asc1 <= 0 || asc2 <= 0 {
		t.Fatalf("ascent expected positive: %v / %v", asc1, asc2)
	}
	// Allow a small floating-point slack.
	if asc2 < 1.9*asc1 || asc2 > 2.1*asc1 {
		t.Errorf("ascent at 24pt (%v) should be ~2x 12pt (%v)", asc2, asc1)
	}
	if desc1 < 0 || desc2 < 0 {
		t.Errorf("descent reported as negative: %v / %v", desc1, desc2)
	}
}
