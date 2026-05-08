package math

import "testing"

// TestLookupSymbolGreek covers the lowercase greek alphabet entries that the
// parser must resolve to canonical Unicode runes. The list is intentionally
// exhaustive so adding or moving a row in symbols.go is caught immediately.
func TestLookupSymbolGreek(t *testing.T) {
	cases := []struct {
		name string
		want rune
	}{
		{"\\alpha", 0x03B1},
		{"\\beta", 0x03B2},
		{"\\gamma", 0x03B3},
		{"\\delta", 0x03B4},
		{"\\epsilon", 0x03F5},
		{"\\zeta", 0x03B6},
		{"\\eta", 0x03B7},
		{"\\theta", 0x03B8},
		{"\\iota", 0x03B9},
		{"\\kappa", 0x03BA},
		{"\\lambda", 0x03BB},
		{"\\mu", 0x03BC},
		{"\\nu", 0x03BD},
		{"\\xi", 0x03BE},
		{"\\omicron", 0x03BF},
		{"\\pi", 0x03C0},
		{"\\rho", 0x03C1},
		{"\\sigma", 0x03C3},
		{"\\tau", 0x03C4},
		{"\\upsilon", 0x03C5},
		{"\\phi", 0x03D5},
		{"\\chi", 0x03C7},
		{"\\psi", 0x03C8},
		{"\\omega", 0x03C9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := LookupSymbol(tc.name)
			if !ok {
				t.Fatalf("LookupSymbol(%q) returned ok=false", tc.name)
			}
			if info.Rune != tc.want {
				t.Fatalf("LookupSymbol(%q) rune = U+%04X, want U+%04X", tc.name, info.Rune, tc.want)
			}
			if info.Category != CategoryGreek {
				t.Fatalf("LookupSymbol(%q) category = %d, want CategoryGreek", tc.name, info.Category)
			}
		})
	}
}

// TestLookupSymbolUppercaseGreek covers the uppercase greek letters with
// distinct LaTeX commands.
func TestLookupSymbolUppercaseGreek(t *testing.T) {
	cases := []struct {
		name string
		want rune
	}{
		{"\\Gamma", 0x0393},
		{"\\Delta", 0x0394},
		{"\\Theta", 0x0398},
		{"\\Lambda", 0x039B},
		{"\\Xi", 0x039E},
		{"\\Pi", 0x03A0},
		{"\\Sigma", 0x03A3},
		{"\\Phi", 0x03A6},
		{"\\Psi", 0x03A8},
		{"\\Omega", 0x03A9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := LookupSymbol(tc.name)
			if !ok {
				t.Fatalf("LookupSymbol(%q) returned ok=false", tc.name)
			}
			if info.Rune != tc.want {
				t.Fatalf("LookupSymbol(%q) rune = U+%04X, want U+%04X", tc.name, info.Rune, tc.want)
			}
			if info.Category != CategoryGreek {
				t.Fatalf("LookupSymbol(%q) category = %d, want CategoryGreek", tc.name, info.Category)
			}
		})
	}
}

// TestLookupSymbolBigOps asserts \sum, \int and \prod are categorised as
// big operators with the correct Unicode glyphs.
func TestLookupSymbolBigOps(t *testing.T) {
	cases := []struct {
		name string
		want rune
	}{
		{"\\sum", 0x2211},
		{"\\int", 0x222B},
		{"\\prod", 0x220F},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := LookupSymbol(tc.name)
			if !ok {
				t.Fatalf("LookupSymbol(%q) returned ok=false", tc.name)
			}
			if info.Rune != tc.want {
				t.Fatalf("LookupSymbol(%q) rune = U+%04X, want U+%04X", tc.name, info.Rune, tc.want)
			}
			if info.Category != CategoryBigOp {
				t.Fatalf("LookupSymbol(%q) category = %d, want CategoryBigOp", tc.name, info.Category)
			}
			if !IsBigOpCommand(tc.name) {
				t.Fatalf("IsBigOpCommand(%q) = false, want true", tc.name)
			}
		})
	}
}

// TestLookupSymbolNamed covers the named relational, binary-operator and
// miscellaneous symbols promised by the parser surface.
func TestLookupSymbolNamed(t *testing.T) {
	cases := []struct {
		name     string
		want     rune
		category SymbolCategory
	}{
		{"\\infty", 0x221E, CategorySymbol},
		{"\\partial", 0x2202, CategorySymbol},
		{"\\pm", 0x00B1, CategoryBinaryOp},
		{"\\mp", 0x2213, CategoryBinaryOp},
		{"\\cdot", 0x22C5, CategoryBinaryOp},
		{"\\times", 0x00D7, CategoryBinaryOp},
		{"\\leq", 0x2264, CategoryRelation},
		{"\\geq", 0x2265, CategoryRelation},
		{"\\neq", 0x2260, CategoryRelation},
		{"\\approx", 0x2248, CategoryRelation},
		{"\\to", 0x2192, CategoryRelation},
		{"\\rightarrow", 0x2192, CategoryRelation},
		{"\\leftarrow", 0x2190, CategoryRelation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := LookupSymbol(tc.name)
			if !ok {
				t.Fatalf("LookupSymbol(%q) returned ok=false", tc.name)
			}
			if info.Rune != tc.want {
				t.Fatalf("LookupSymbol(%q) rune = U+%04X, want U+%04X", tc.name, info.Rune, tc.want)
			}
			if info.Category != tc.category {
				t.Fatalf("LookupSymbol(%q) category = %d, want %d", tc.name, info.Category, tc.category)
			}
		})
	}
}

// TestLookupSymbolUnknown asserts an unknown command does not resolve.
func TestLookupSymbolUnknown(t *testing.T) {
	if _, ok := LookupSymbol("\\nopesuchcommand"); ok {
		t.Fatalf("LookupSymbol returned ok=true for an unknown command")
	}
	if IsBigOpCommand("\\nopesuchcommand") {
		t.Fatalf("IsBigOpCommand returned true for an unknown command")
	}
}
