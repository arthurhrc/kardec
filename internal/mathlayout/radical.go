package mathlayout

const (
	// radicalSymbol is the Unicode square-root sign drawn as the
	// leading glyph. The body's overline starts at the symbol's right
	// edge.
	radicalSymbol = "√"

	// radicalOverlineThickness is the thickness of the overline drawn
	// above the body, expressed as a fraction of the surrounding font
	// size. It matches fracRuleThickness so radicals and fractions look
	// visually consistent.
	radicalOverlineThickness = 0.06

	// radicalBodyMargin is the vertical breathing room added above the
	// body so the overline does not visually clip ascenders.
	radicalBodyMargin = 0.05

	// nthRootIndexScale is the size factor applied to the index of an
	// n-th root, drawn small at the top-left of the radical. TeX uses
	// scriptscript style here, which is roughly 0.5 of the surrounding
	// size; we adopt that ratio directly.
	nthRootIndexScale = 0.50

	// nthRootIndexShiftUp is the distance the index baseline sits
	// above the parent baseline, expressed as a fraction of the
	// surrounding size. The index attaches to the top-left of the
	// radical glyph, so it shifts up roughly the radical's height.
	nthRootIndexShiftUp = 0.55
)

// layoutSqrt typesets a square root: the radical glyph at the baseline,
// followed by the body, with an overline rule spanning the body's
// width. The body is shifted right by the radical's width and slightly
// down so the overline has breathing room above it.
func layoutSqrt(s Sqrt, font Font, sizePt float64, display bool) Box {
	body := layoutNode(s.Body(), font, sizePt, display)
	return composeRadical(Box{}, 0, body, font, sizePt)
}

// layoutNthRoot typesets a generalised n-th root: same structure as a
// square root, with a small index drawn at the top-left of the radical.
// The index shifts the entire radical+body to the right so the index
// has room before the radical glyph.
func layoutNthRoot(r NthRoot, font Font, sizePt float64, display bool) Box {
	indexSize := sizePt * nthRootIndexScale
	idx := layoutNode(r.Index(), font, indexSize, false)
	body := layoutNode(r.Body(), font, sizePt, display)
	return composeRadical(idx, idx.Width, body, font, sizePt)
}

// composeRadical performs the shared assembly of square-root and
// n-th-root boxes. indexBox is the (already laid out) small index, or
// the zero Box when none is present; indexAdvance is the horizontal
// shift it imposes on the radical glyph (zero for plain sqrt).
//
// The function emits the radical glyph as a PlacedGlyph on the parent
// (rather than a child Box) because the glyph carries no nested
// structure of its own, mirroring layoutGlyphRun.
func composeRadical(indexBox Box, indexAdvance float64, body Box, font Font, sizePt float64) Box {
	g, ok := font.GlyphFor(radicalSymbol)
	var radWidth, radAscent, radDescent float64
	if ok {
		radWidth = font.Measure(g, sizePt)
		radAscent, radDescent = font.AscentDescent(g, sizePt)
	}
	margin := radicalBodyMargin * sizePt
	overlineThickness := radicalOverlineThickness * sizePt

	// Vertical metrics:
	//   parent height = max(body.Height + margin + overlineThickness,
	//                       radical ascent, index ascent shifted up)
	//   parent depth  = max(body.Depth, radical descent)
	bodyAscent := body.Height + margin + overlineThickness
	parentHeight := bodyAscent
	if radAscent > parentHeight {
		parentHeight = radAscent
	}
	if indexBox.Height > 0 {
		idxTotal := nthRootIndexShiftUp*sizePt + indexBox.Height
		if idxTotal > parentHeight {
			parentHeight = idxTotal
		}
	}
	parentDepth := body.Depth
	if radDescent > parentDepth {
		parentDepth = radDescent
	}

	parent := Box{
		Width:  indexAdvance + radWidth + body.Width,
		Height: parentHeight,
		Depth:  parentDepth,
	}

	// Index (n-th root only): top-left of the radical. Place its
	// baseline nthRootIndexShiftUp × sizePt above the parent baseline.
	if indexBox.Width > 0 || indexBox.Height > 0 {
		indexBox.X = 0
		indexBox.Y = parentHeight - nthRootIndexShiftUp*sizePt - indexBox.Height
		parent.Children = append(parent.Children, indexBox)
	}

	// Radical glyph at the baseline.
	if ok {
		parent.Glyphs = append(parent.Glyphs, PlacedGlyph{
			X:      indexAdvance,
			Y:      parentHeight - radAscent,
			Rune:   g.Rune,
			SizePt: sizePt,
		})
	}

	// Body: shifted right of the radical glyph. Baseline-aligned with
	// the parent.
	body.X = indexAdvance + radWidth
	body.Y = parentHeight - body.Height
	parent.Children = append(parent.Children, body)

	// Overline: spans the body, sits margin above the body's top edge
	// (which is parent baseline - body.Height = parentHeight -
	// body.Height - body.Y... but body.Y is parentHeight - body.Height
	// in baseline coords, so its top edge is at body.Y).
	overlineY := body.Y - margin - overlineThickness
	if overlineY < 0 {
		overlineY = 0
	}
	parent.Rules = append(parent.Rules, Rule{
		X:         indexAdvance + radWidth,
		Y:         overlineY,
		Width:     body.Width,
		Thickness: overlineThickness,
	})

	return parent
}
