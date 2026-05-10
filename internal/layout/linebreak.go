package layout

import (
	"github.com/arthurhrc/kardec"
	mathast "github.com/arthurhrc/kardec/internal/math"
	"github.com/arthurhrc/kardec/internal/mathadapter"
	"github.com/arthurhrc/kardec/internal/mathlayout"
	"github.com/arthurhrc/kardec/internal/typography"
)

// token is a single shaped fragment the line breaker treats as an atomic
// unit: either a run of non-whitespace characters or a single whitespace
// gap. Whitespace tokens are collapsed at line boundaries; non-whitespace
// tokens are never split (no hyphenation in v0.1).
type token struct {
	text          string
	isSpace       bool
	width         float64
	font          Font
	sizePt        float64
	color         kardec.Color
	ascentPt      float64
	descentPt     float64
	link          string // copied from the originating Run; empty when plain
	footnoteRef   int    // 1-based footnote number; 0 when not a footnote marker
	underline     bool
	strikethrough bool
	// mathBox carries the laid-out inline-math expression for tokens
	// produced from kardec.InlineMath runs. The breaker treats math
	// tokens as opaque, non-breakable units: width is the math box
	// width, ascent/descent come from box.Height/Depth. emitLine
	// detects the field and emits the math glyphs + rules at the
	// token's resolved (X, baseline) instead of a single Tj op.
	mathBox *mathlayout.Box
}

// shapeRuns turns a slice of kardec.Run values into the flat token stream
// the line breaker consumes. Each Run becomes one or more tokens following
// the rule "split on whitespace, keep whitespace as its own token". The
// blockStyle's family / bold / italic flow as defaults; per-Run bold and
// italic flags are ORed on top so an inline Bold(...) run inside a
// regular paragraph still resolves to a bold face.
//
// Inline math runs (Run.MathSource() != "") are converted into a
// single math token via the math parser + layout engine. mathFont
// is the doc-resolved Latin Modern Math face; pass nil to drop math
// runs silently (matches the display-math fallback behaviour when
// the math font fails to load).
func shapeRuns(runs []kardec.Run, fonts FontProvider, style blockStyle, defaultSize kardec.Length, defaultColor kardec.Color, mathFont typography.MathFont) []token {
	var out []token
	for _, r := range runs {
		if src := r.MathSource(); src != "" {
			if mt, ok := shapeInlineMath(src, mathFont, float64(defaultSize), defaultColor); ok {
				out = append(out, mt)
			}
			continue
		}
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
				text:          piece,
				isSpace:       isAllSpace(piece),
				width:         w,
				font:          font,
				sizePt:        size,
				color:         color,
				ascentPt:      asc,
				descentPt:     desc,
				link:          r.Link(),
				footnoteRef:   r.FootnoteRef(),
				underline:     r.Underline(),
				strikethrough: r.Strikethrough(),
			})
		}
	}
	return out
}

// shapeInlineMath parses src, runs the math layout engine in inline
// (non-display) style, and packages the result as a single
// non-breakable token. The token's width matches the math box's
// horizontal extent; ascent/descent come from box.Height /
// box.Depth so vertical-aware decoration (e.g. struts) lines up.
//
// Returns ok=false when:
//   - the math font is nil (math face failed to load — caller drops
//     the run rather than rendering Liberation glyphs that would
//     misrepresent the math)
//   - the parser rejects src (a v0.21.x follow-up will surface a
//     [math: ...] inline fallback similar to display math)
//   - layout produces an empty box (degenerate input like "")
func shapeInlineMath(src string, mathFont typography.MathFont, sizePt float64, color kardec.Color) (token, bool) {
	if mathFont == nil {
		return token{}, false
	}
	expr, err := mathast.Parse(src)
	if err != nil {
		return token{}, false
	}
	box := mathlayout.Layout(mathadapter.WrapExpr(expr), mathadapter.WrapFont(mathFont), sizePt, false)
	if box.Width == 0 && len(box.Glyphs) == 0 && len(box.Children) == 0 {
		return token{}, false
	}
	asc := box.Height
	desc := box.Depth
	if asc == 0 {
		asc = sizePt * 0.7
	}
	if desc == 0 {
		desc = sizePt * 0.2
	}
	boxCopy := box
	return token{
		text:      "",
		isSpace:   false,
		width:     box.Width,
		font:      &mathFontMarker{},
		sizePt:    sizePt,
		color:     color,
		ascentPt:  asc,
		descentPt: desc,
		mathBox:   &boxCopy,
	}, true
}

// mathFontMarker is the Font value attached to inline-math tokens.
// PlacedItem requires a Font on every text fragment, but the math
// glyphs go through the math face at emit time, not through the
// regular Measure/shape path. The marker satisfies the type
// contract while being trivially identifiable for downstream
// dispatch.
type mathFontMarker struct{}

func (*mathFontMarker) Measure(text string, sizePt float64) (float64, float64, float64) {
	return float64(len(text)) * sizePt * 0.5, sizePt * 0.7, sizePt * 0.2
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
			text:          head + "-",
			width:         headWidth + hyphenWidth,
			font:          t.font,
			sizePt:        t.sizePt,
			color:         t.color,
			ascentPt:      asc,
			descentPt:     desc,
			link:          t.link,
			footnoteRef:   t.footnoteRef,
			underline:     t.underline,
			strikethrough: t.strikethrough,
		}
		tw, tasc, tdesc := t.font.Measure(tail, t.sizePt)
		tToken := token{
			text:          tail,
			width:         tw,
			font:          t.font,
			sizePt:        t.sizePt,
			color:         t.color,
			ascentPt:      tasc,
			descentPt:     tdesc,
			link:          t.link,
			footnoteRef:   t.footnoteRef,
			underline:     t.underline,
			strikethrough: t.strikethrough,
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
