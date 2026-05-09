package layout

import (
	"strings"
	"testing"
)

// hyphenFontProvider gives every glyph a 6-pt advance at 12pt; useful
// because hyphenation tests need predictable widths to assert that a
// prefix exactly fits a constrained line.
type hyphenFontProvider struct{}

func (hyphenFontProvider) Resolve(string, bool, bool) Font { return hyphenFont{} }

type hyphenFont struct{}

func (hyphenFont) Measure(text string, sizePt float64) (float64, float64, float64) {
	return float64(len(text)) * sizePt * 0.5, sizePt * 0.7, sizePt * 0.2
}

// TestBreakLinesHyphenatesOversizedWord checks that when a single
// word would not fit, the breaker emits a hyphenated split rather
// than letting the whole word cascade onto the next line.
func TestBreakLinesHyphenatesOversizedWord(t *testing.T) {
	// "computer" at 12pt is 8 * 6 = 48pt wide. With available=30pt
	// the breaker must split — vowel-CC-vowel rule offers index 3
	// ("com" + "puter"); "com-" = 4 * 6 = 24pt fits in 30.
	tokens := []token{
		{
			text: "computer", width: 48, font: hyphenFont{},
			sizePt: 12, ascentPt: 8, descentPt: 2,
		},
	}
	lines := breakLines(tokens, 30)
	if len(lines) < 2 {
		t.Fatalf("hyphenation should produce at least 2 lines, got %d", len(lines))
	}
	first := lines[0].tokens[0].text
	if !strings.HasSuffix(first, "-") {
		t.Errorf("first line should end in a soft hyphen, got %q", first)
	}
	last := lines[len(lines)-1].tokens[0].text
	if strings.Contains(last, "-") {
		t.Errorf("tail token should not carry a hyphen, got %q", last)
	}
}

// TestBreakLinesNoHyphenWhenWordIsShort verifies the hyphenator
// declines to split words below the 6-character heuristic.
func TestBreakLinesNoHyphenWhenWordIsShort(t *testing.T) {
	tokens := []token{
		{text: "the", width: 18, font: hyphenFont{}, sizePt: 12, ascentPt: 8, descentPt: 2},
	}
	lines := breakLines(tokens, 10) // forced overflow
	// No hyphenation possible: the breaker still emits one line
	// carrying the word — the "must fit at least one token per
	// line" invariant of the greedy breaker.
	if len(lines) != 1 {
		t.Fatalf("short word should produce 1 line, got %d", len(lines))
	}
	if lines[0].tokens[0].text != "the" {
		t.Errorf("short word should travel verbatim, got %q", lines[0].tokens[0].text)
	}
}
