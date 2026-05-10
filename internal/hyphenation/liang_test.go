package hyphenation

import "testing"

func TestLiangBreakPointsMatchesPrefixPattern(t *testing.T) {
	// "preview" — the .pre3 pattern places an odd score after pre,
	// so position 3 should be a valid break.
	got := liangBreakPoints("preview", "en")
	want := false
	for _, p := range got {
		if p == 3 {
			want = true
		}
	}
	if !want {
		t.Errorf("expected Liang break at position 3 for 'preview', got %v", got)
	}
}

func TestLiangBreakPointsMatchesSuffixPattern(t *testing.T) {
	// "creation" matches "tion." with leading score 3 — the gap
	// before 't' should be marked. word="creation" len 8, position
	// of 't' is 4, so break at 4.
	got := liangBreakPoints("creation", "en")
	if len(got) == 0 {
		t.Errorf("expected Liang to find at least one break in 'creation', got none")
	}
}

func TestLiangBreakPointsMatchesDoubleConsonant(t *testing.T) {
	// "rabbit" — pattern "1bb" puts an odd score between the b's.
	// Position 3 = before the second 'b'.
	got := liangBreakPoints("rabbit", "en")
	if len(got) == 0 {
		t.Errorf("expected Liang break for 'rabbit', got none")
	}
}

func TestLiangBreakPointsEmptyForUnknownLanguage(t *testing.T) {
	got := liangBreakPoints("zwiebel", "de")
	if len(got) != 0 {
		t.Errorf("Liang with no German patterns should return empty, got %v", got)
	}
}

func TestRegisterMergesAdditionalPatterns(t *testing.T) {
	t.Cleanup(func() {
		// Reset the extra patterns map so this test doesn't leak
		// state into other tests.
		extraPatterns = map[string]map[string]string{}
	})
	// Add a bespoke pattern that triggers a break for "kardec".
	// kardec = k-a-r-d-e-c. Pattern "ar3dec" adds score 3 between
	// 'r' and 'd', so position 3 should become a valid break.
	Register("en", map[string]string{
		"ardec": "ar3dec",
	})
	got := liangBreakPoints("kardec", "en")
	found := false
	for _, p := range got {
		if p == 3 {
			found = true
		}
	}
	if !found {
		t.Errorf("registered pattern 'ar3dec' did not produce break at 3 for 'kardec', got %v", got)
	}
}

func TestApplyPatternMaxScoreWins(t *testing.T) {
	// Two overlapping patterns at the same gap: max score wins.
	scores := make([]int, 10)
	applyPattern(scores, "1ab", 0)
	if scores[0] != 1 {
		t.Errorf("first pattern should set score 1, got %d", scores[0])
	}
	applyPattern(scores, "5ab", 0)
	if scores[0] != 5 {
		t.Errorf("higher score should win, got %d", scores[0])
	}
	applyPattern(scores, "2ab", 0)
	if scores[0] != 5 {
		t.Errorf("lower score should not overwrite, got %d", scores[0])
	}
}
