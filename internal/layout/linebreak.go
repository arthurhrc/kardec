package layout

import (
	"github.com/arthurhrc/kardec"
)

// token is a single shaped fragment the line breaker treats as an atomic
// unit: either a run of non-whitespace characters or a single whitespace
// gap. Whitespace tokens are collapsed at line boundaries; non-whitespace
// tokens are never split (no hyphenation in v0.1).
type token struct {
	text      string
	isSpace   bool
	width     float64
	font      Font
	sizePt    float64
	color     kardec.Color
	ascentPt  float64
	descentPt float64
}

// shapeRuns turns a slice of kardec.Run values into the flat token stream
// the line breaker consumes. Each Run becomes one or more tokens following
// the rule "split on whitespace, keep whitespace as its own token".
func shapeRuns(runs []kardec.Run, fonts FontProvider, defaultSize kardec.Length, defaultColor kardec.Color) []token {
	var out []token
	for _, r := range runs {
		text := r.Text()
		if text == "" {
			continue
		}
		font := fonts.Resolve("", r.Bold(), r.Italic())
		size := float64(defaultSize)
		if v, ok := r.SizeOverride(); ok {
			size = float64(v)
		}
		color := defaultColor
		if v, ok := r.ColorOverride(); ok {
			color = v
		}
		for _, piece := range splitKeepingSpaces(text) {
			w, asc, desc := font.Measure(piece, size)
			out = append(out, token{
				text:      piece,
				isSpace:   isAllSpace(piece),
				width:     w,
				font:      font,
				sizePt:    size,
				color:     color,
				ascentPt:  asc,
				descentPt: desc,
			})
		}
	}
	return out
}

// splitKeepingSpaces splits s into a slice of substrings where each entry
// is either entirely whitespace or entirely non-whitespace. This keeps the
// breaker's logic uniform: every gap between words is a discrete token
// whose width can be discarded at line ends.
func splitKeepingSpaces(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	prevSpace := isSpaceByte(s[0])
	for i := 1; i < len(s); i++ {
		curSpace := isSpaceByte(s[i])
		if curSpace != prevSpace {
			out = append(out, s[start:i])
			start = i
			prevSpace = curSpace
		}
	}
	out = append(out, s[start:])
	return out
}

func isSpaceByte(b byte) bool { return b == ' ' || b == '\t' }

func isAllSpace(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isSpaceByte(s[i]) {
			return false
		}
	}
	return true
}

// line is the breaker's intermediate result: the tokens that fit on one
// horizontal strip plus the maximum ascent/descent observed so the page
// engine can advance Y by the right amount.
type line struct {
	tokens  []token
	width   float64 // sum of token widths after trailing-space strip
	ascent  float64
	descent float64
}

// breakLines runs greedy line breaking against the token stream. Whitespace
// at the start of a fresh line is dropped; whitespace at the end of a line
// is dropped from the width total (so justification has the correct gap
// budget) but kept for downstream inspection if needed.
func breakLines(tokens []token, available float64) []line {
	var lines []line
	var cur line

	flush := func() {
		// Strip trailing whitespace from both the slice and the width.
		for len(cur.tokens) > 0 && cur.tokens[len(cur.tokens)-1].isSpace {
			cur.width -= cur.tokens[len(cur.tokens)-1].width
			cur.tokens = cur.tokens[:len(cur.tokens)-1]
		}
		if len(cur.tokens) == 0 {
			cur = line{}
			return
		}
		lines = append(lines, cur)
		cur = line{}
	}

	for _, t := range tokens {
		if t.isSpace && len(cur.tokens) == 0 {
			// Skip leading whitespace on a fresh line.
			continue
		}
		if !t.isSpace && cur.width+t.width > available && len(cur.tokens) > 0 {
			flush()
		}
		cur.tokens = append(cur.tokens, t)
		cur.width += t.width
		if t.ascentPt > cur.ascent {
			cur.ascent = t.ascentPt
		}
		if t.descentPt > cur.descent {
			cur.descent = t.descentPt
		}
	}
	flush()
	return lines
}
