package layout

import (
	"github.com/arthurhrc/kardec"
)

// token is a single shaped fragment the line breaker treats as an atomic
// unit: either a run of non-whitespace characters or a single whitespace
// gap. Whitespace tokens are collapsed at line boundaries; non-whitespace
// tokens are never split (no hyphenation in v0.1).
type token struct {
	text        string
	isSpace     bool
	width       float64
	font        Font
	sizePt      float64
	color       kardec.Color
	ascentPt    float64
	descentPt   float64
	link        string // copied from the originating Run; empty when plain
	footnoteRef int    // 1-based footnote number; 0 when not a footnote marker
}

// shapeRuns turns a slice of kardec.Run values into the flat token stream
// the line breaker consumes. Each Run becomes one or more tokens following
// the rule "split on whitespace, keep whitespace as its own token". The
// blockStyle's family / bold / italic flow as defaults; per-Run bold and
// italic flags are ORed on top so an inline Bold(...) run inside a
// regular paragraph still resolves to a bold face.
func shapeRuns(runs []kardec.Run, fonts FontProvider, style blockStyle, defaultSize kardec.Length, defaultColor kardec.Color) []token {
	var out []token
	for _, r := range runs {
		text := r.Text()
		if text == "" {
			continue
		}
		font := fonts.Resolve(style.family, style.bold || r.Bold(), style.italic || r.Italic())
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
				text:        piece,
				isSpace:     isAllSpace(piece),
				width:       w,
				font:        font,
				sizePt:      size,
				color:       color,
				ascentPt:    asc,
				descentPt:   desc,
				link:        r.Link(),
				footnoteRef: r.FootnoteRef(),
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

// tryHyphenate looks for a word break that lets a prefix of t.text
// (plus a trailing hyphen) fit within remaining points. Returns the
// head token (with the hyphen appended) and the tail token (the
// remainder of the word) when a viable split exists.
func tryHyphenate(t token, remaining float64) (token, token, bool) {
	if t.isSpace || t.text == "" || remaining <= 0 {
		return token{}, token{}, false
	}
	breaks := hyphenBreakPoints(t.text)
	if len(breaks) == 0 {
		return token{}, token{}, false
	}
	// Width per character at this size — use Measure on a single
	// rune to derive the average advance for the slicing math.
	hyphenWidth, _, _ := t.font.Measure("-", t.sizePt)
	// Walk from the rightmost candidate down so the line fills as
	// much as possible before breaking.
	for i := len(breaks) - 1; i >= 0; i-- {
		b := breaks[i]
		head := t.text[:b]
		tail := t.text[b:]
		headWidth, asc, desc := t.font.Measure(head, t.sizePt)
		if headWidth+hyphenWidth > remaining {
			continue
		}
		hToken := token{
			text:        head + "-",
			width:       headWidth + hyphenWidth,
			font:        t.font,
			sizePt:      t.sizePt,
			color:       t.color,
			ascentPt:    asc,
			descentPt:   desc,
			link:        t.link,
			footnoteRef: t.footnoteRef,
		}
		tw, tasc, tdesc := t.font.Measure(tail, t.sizePt)
		tToken := token{
			text:        tail,
			width:       tw,
			font:        t.font,
			sizePt:      t.sizePt,
			color:       t.color,
			ascentPt:    tasc,
			descentPt:   tdesc,
			link:        t.link,
			footnoteRef: t.footnoteRef,
		}
		return hToken, tToken, true
	}
	return token{}, token{}, false
}

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

	pushToken := func(t token) {
		cur.tokens = append(cur.tokens, t)
		cur.width += t.width
		if t.ascentPt > cur.ascent {
			cur.ascent = t.ascentPt
		}
		if t.descentPt > cur.descent {
			cur.descent = t.descentPt
		}
	}

	for _, t := range tokens {
		if t.isSpace && len(cur.tokens) == 0 {
			// Skip leading whitespace on a fresh line.
			continue
		}
		if !t.isSpace && cur.width+t.width > available {
			// Try hyphenation before forcing the whole word onto a
			// new line. If a prefix + soft hyphen fits in the
			// remaining width — including when the line is empty
			// and the word would otherwise overflow on its own —
			// emit it, flush, and continue with the suffix. When
			// the line already carries tokens, fall back to flush
			// even if hyphenation declines, preserving the greedy
			// "fit at least one token per line" invariant.
			remaining := available - cur.width
			if remaining <= 0 {
				remaining = available
			}
			if head, tail, ok := tryHyphenate(t, remaining); ok {
				pushToken(head)
				flush()
				t = tail
			} else if len(cur.tokens) > 0 {
				flush()
			}
		}
		pushToken(t)
	}
	flush()
	return lines
}
