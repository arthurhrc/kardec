package layout

import (
	"math"
	"testing"
)

// stubFontMeasure builds tokens with deterministic fixed widths so
// the breaker DP can be exercised without pulling in typography.
// Each space has width 4, each non-space token width = len(text)*7.
func makeTokens(words []string) []token {
	var out []token
	for i, w := range words {
		if i > 0 {
			out = append(out, token{
				text:    " ",
				isSpace: true,
				width:   4,
			})
		}
		out = append(out, token{
			text:    w,
			isSpace: false,
			width:   float64(len(w)) * 7,
		})
	}
	return out
}

func TestKnuthPlassEmptyInputReturnsNil(t *testing.T) {
	if got := breakLinesOptimal(nil, 100); got != nil {
		t.Errorf("expected nil for empty tokens, got %v", got)
	}
}

func TestKnuthPlassFitsAllInOneLineWhenWidthIsAmple(t *testing.T) {
	toks := makeTokens([]string{"foo", "bar", "baz"})
	lines := breakLinesOptimal(toks, 1000)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line on ample width, got %d", len(lines))
	}
}

func TestKnuthPlassBreaksIntoMultipleLinesWhenNarrow(t *testing.T) {
	toks := makeTokens([]string{"alpha", "beta", "gamma", "delta", "epsilon"})
	lines := breakLinesOptimal(toks, 60)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines on narrow width, got %d", len(lines))
	}
	for i, ln := range lines {
		if len(ln.tokens) == 0 {
			t.Errorf("line %d is empty", i)
		}
		if ln.width > 60+1e-6 {
			t.Errorf("line %d width %.2f exceeds available 60", i, ln.width)
		}
	}
}

// TestKnuthPlassDistributesMoreEvenlyThanGreedy exercises the
// classic motivating example: a paragraph where greedy crams the
// first lines tight and leaves the last with one short word.
// Optimal mode should produce more uniform line widths (lower
// variance).
func TestKnuthPlassDistributesMoreEvenlyThanGreedy(t *testing.T) {
	words := []string{
		"the", "quick", "brown", "fox",
		"jumps", "over", "the", "lazy", "dog",
		"and", "then", "runs", "back", "home",
	}
	toks := makeTokens(words)
	const w = 80

	greedy := breakLines(toks, w)
	optimal := breakLinesOptimal(toks, w)

	if len(greedy) == 0 || len(optimal) == 0 {
		t.Fatalf("both breakers must produce lines (greedy=%d, optimal=%d)",
			len(greedy), len(optimal))
	}

	varOf := func(ls []line) float64 {
		var mean float64
		for _, l := range ls {
			mean += l.width
		}
		mean /= float64(len(ls))
		var v float64
		for _, l := range ls {
			d := l.width - mean
			v += d * d
		}
		return v / float64(len(ls))
	}

	gv := varOf(greedy)
	ov := varOf(optimal)
	if math.IsNaN(gv) || math.IsNaN(ov) {
		t.Fatalf("variance computation produced NaN")
	}
	if ov > gv {
		t.Errorf("optimal variance (%.2f) should be ≤ greedy variance (%.2f)", ov, gv)
	}
}

func TestKnuthPlassFallsBackOnOversizedToken(t *testing.T) {
	// One word wider than the column triggers the greedy fallback.
	toks := []token{
		{text: "aaaaaaaaaaaa", isSpace: false, width: 200},
	}
	lines := breakLinesOptimal(toks, 50)
	if len(lines) == 0 {
		t.Fatalf("fallback should produce at least one line, got none")
	}
}
