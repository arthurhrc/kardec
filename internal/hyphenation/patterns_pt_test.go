package hyphenation

import "testing"

// TestPortugueseBreaksOnOpenSyllables exercises the pt-BR pattern
// table: a long Portuguese word should produce at least one break
// candidate from the bundled subset.
func TestPortugueseBreaksOnOpenSyllables(t *testing.T) {
	// "palavra" (word): expected break candidates around the
	// open-syllable boundaries pa|la|vra. Even just one break
	// suffices for the test — the bundled subset is conservative.
	got := BreakPoints("palavra", "pt-BR")
	if len(got) == 0 {
		t.Errorf("BreakPoints(palavra, pt-BR) = nil — pt subset should produce at least one break")
	}
}

func TestPortugueseDigraphsStayUnsplit(t *testing.T) {
	// The "nh" digraph in "tenho" should NOT split between n and h.
	// Verify by checking that no break offset == position of 'h'.
	got := BreakPoints("tenho", "pt")
	for _, off := range got {
		if off == 3 { // position between 'n' and 'h'
			t.Errorf("digraph 'nh' should not split: got break at offset 3 in 'tenho'")
		}
	}
}

func TestPortugueseAliasNormalises(t *testing.T) {
	a := BreakPoints("palavra", "pt-BR")
	b := BreakPoints("palavra", "pt")
	if len(a) != len(b) {
		t.Errorf("pt-BR and pt should resolve to the same subset; got %v vs %v", a, b)
	}
}

func TestSpanishBundled(t *testing.T) {
	// Spanish has the same open-syllable rule as Portuguese for
	// "trabajar" — at least one break should fall out.
	got := BreakPoints("trabajar", "es")
	if len(got) == 0 {
		t.Errorf("BreakPoints(trabajar, es) = nil — es subset should produce a break")
	}
}

func TestFrenchBundled(t *testing.T) {
	// "ordinateur" — open-syllable rule yields or-di-na-teur, several
	// candidate breaks.
	got := BreakPoints("ordinateur", "fr")
	if len(got) == 0 {
		t.Errorf("BreakPoints(ordinateur, fr) = nil — fr subset should produce a break")
	}
}
