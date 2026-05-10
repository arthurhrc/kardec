package mathlayout

// Layout positions expr at the requested point size, returning a single
// Box whose Width, Height and Depth describe the entire formula.
//
// The display flag selects between the two TeX-flavoured styles:
//
//   - display=false (textstyle) — used for math embedded in a paragraph.
//     Big operators keep their limits as ordinary scripts and fractions
//     are typeset at 90% of the surrounding size.
//   - display=true (displaystyle) — used for stand-alone equations.
//     Big-operator limits centre above and below the operator, and
//     fractions are typeset at full size.
//
// The function is total: a nil expr or an unrecognised Kind yields an
// empty Box rather than panicking, so callers that splice user input
// into a document do not have to pre-validate the AST.
func Layout(expr Expr, font Font, sizePt float64, display bool) Box {
	if expr == nil || font == nil || sizePt <= 0 {
		return Box{}
	}
	return layoutNode(expr, font, sizePt, display)
}

// layoutNode dispatches on Kind and returns a fully-positioned Box for
// the given expression. It is split out from the public Layout entry so
// recursive calls do not re-validate inputs.
func layoutNode(expr Expr, font Font, sizePt float64, display bool) Box {
	if expr == nil {
		return Box{}
	}
	switch expr.Kind() {
	case KindAtom:
		if a, ok := expr.(Atom); ok {
			return layoutGlyphRun(a.Symbol(), font, sizePt)
		}
	case KindOp:
		if o, ok := expr.(Op); ok {
			return layoutGlyphRun(o.Symbol(), font, sizePt)
		}
	case KindNumber:
		if n, ok := expr.(Number); ok {
			return layoutGlyphRun(n.Value(), font, sizePt)
		}
	case KindIdentifier:
		if id, ok := expr.(Identifier); ok {
			return layoutGlyphRun(id.Name(), font, sizePt)
		}
	case KindGroup:
		if g, ok := expr.(Group); ok {
			return layoutGroup(g, font, sizePt, display)
		}
	case KindSubSup:
		if s, ok := expr.(SubSup); ok {
			return layoutSubSup(s, font, sizePt, display)
		}
	case KindFrac:
		if f, ok := expr.(Frac); ok {
			return layoutFrac(f, font, sizePt, display)
		}
	case KindSqrt:
		if s, ok := expr.(Sqrt); ok {
			return layoutSqrt(s, font, sizePt, display)
		}
	case KindNthRoot:
		if r, ok := expr.(NthRoot); ok {
			return layoutNthRoot(r, font, sizePt, display)
		}
	case KindBigOp:
		if op, ok := expr.(BigOp); ok {
			return layoutBigOp(op, font, sizePt, display)
		}
	}
	return Box{}
}

// layoutGlyphRun typesets a flat string of characters as a single Box
// with one PlacedGlyph per token. Tokens are either a backslash-prefixed
// LaTeX command name (`\int`, `\pi`, `\infty`, ...) treated as one
// glyph, or a single rune. Iterating rune-by-rune would split `\int`
// into `\`, `i`, `n`, `t` and emit them as four plain-text glyphs —
// the bug that made integrals and greek letters render as ASCII names
// through v0.21.
//
// Glyphs are placed left-to-right at the baseline; the box's
// Height/Depth are the maximum ascent/descent of the run. This helper
// is shared by atoms, operators, numbers and identifiers, all of which
// have identical layout semantics at this stage of the engine.
func layoutGlyphRun(text string, font Font, sizePt float64) Box {
	var box Box
	if text == "" {
		return box
	}
	x := 0.0
	for _, tok := range tokeniseGlyphRun(text) {
		g, ok := font.GlyphFor(tok)
		if !ok {
			continue
		}
		w := font.Measure(g, sizePt)
		asc, desc := font.AscentDescent(g, sizePt)
		// Y is the top of the glyph relative to the box's top edge,
		// which is the box's max ascent. We don't know the run's max
		// ascent yet, so we record the per-glyph ascent and patch Y in
		// a second pass below — this keeps the helper at O(n).
		box.Glyphs = append(box.Glyphs, PlacedGlyph{
			X:      x,
			Y:      asc, // temporary: holds per-glyph ascent; rewritten below
			Rune:   g.Rune,
			SizePt: sizePt,
		})
		x += w
		if asc > box.Height {
			box.Height = asc
		}
		if desc > box.Depth {
			box.Depth = desc
		}
	}
	// Second pass: convert each glyph's stored ascent into a Y offset
	// from the box's top edge so the baseline of every glyph aligns
	// with the box's baseline (= box.Height from the top).
	for i := range box.Glyphs {
		ascent := box.Glyphs[i].Y
		box.Glyphs[i].Y = box.Height - ascent
	}
	box.Width = x
	return box
}

// tokeniseGlyphRun splits a math-source string into the tokens
// font.GlyphFor accepts. Backslash followed by a run of letters is one
// token (the LaTeX command including the backslash); every other
// character is its own one-rune token.
func tokeniseGlyphRun(s string) []string {
	var out []string
	i := 0
	for i < len(s) {
		if s[i] == '\\' {
			j := i + 1
			for j < len(s) {
				c := s[j]
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					j++
					continue
				}
				break
			}
			if j > i+1 {
				out = append(out, s[i:j])
				i = j
				continue
			}
			// Lone backslash — treat as literal.
			out = append(out, s[i:i+1])
			i++
			continue
		}
		// One rune.
		r := s[i]
		if r < 0x80 {
			out = append(out, s[i:i+1])
			i++
		} else {
			// UTF-8 multi-byte; consume up to 4 bytes.
			n := 1
			for n < 4 && i+n < len(s) && (s[i+n]&0xC0) == 0x80 {
				n++
			}
			out = append(out, s[i:i+n])
			i += n
		}
	}
	return out
}
