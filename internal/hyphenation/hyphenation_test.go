package hyphenation

import "testing"

func TestBreakPointsTooShortNoSplit(t *testing.T) {
	if got := BreakPoints("the", "en"); len(got) != 0 {
		t.Errorf("3-letter word should not split, got %v", got)
	}
}

func TestBreakPointsConsonantPair(t *testing.T) {
	got := BreakPoints("rabbit", "en")
	if len(got) == 0 {
		t.Errorf("rabbit should yield at least one break point")
	}
	for _, b := range got {
		if b < 3 || 6-b < 3 {
			t.Errorf("break %d violates 3-char minimum on each side", b)
		}
	}
}

func TestBreakPointsAfterPrefix(t *testing.T) {
	got := BreakPoints("preconfigure", "en")
	// "pre" + "configure" — 3 vs 9, both ≥ 3, so split at 3 is valid.
	var sawPrefixSplit bool
	for _, b := range got {
		if b == 3 {
			sawPrefixSplit = true
			break
		}
	}
	if !sawPrefixSplit {
		t.Errorf("expected prefix-aware split after 'pre', got %v", got)
	}
}

func TestBreakPointsVcCcv(t *testing.T) {
	// computer: c-o-m-p-u-t-e-r → between 'm' (idx 2) and 'p' (idx 3),
	// expecting break at index 3.
	got := BreakPoints("computer", "en")
	var sawMid bool
	for _, b := range got {
		if b == 3 {
			sawMid = true
			break
		}
	}
	if !sawMid {
		t.Errorf("expected vowel-CC-vowel split at index 3, got %v", got)
	}
}

func TestBreakPointsRejectsShortFragments(t *testing.T) {
	got := BreakPoints("foggy", "en")
	for _, b := range got {
		if b < 3 || 5-b < 3 {
			t.Errorf("break %d violates 3-char minimum", b)
		}
	}
}
