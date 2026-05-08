package layout

import (
	"github.com/arthurhrc/kardec"
	mathast "github.com/arthurhrc/kardec/internal/math"
	"github.com/arthurhrc/kardec/internal/mathadapter"
	"github.com/arthurhrc/kardec/internal/mathlayout"
)

// placeMath lays out a kardec.Math block by feeding its source through
// the math parser, the math layout engine, and finally walking the
// resulting Box tree to emit PlacedItem glyph entries on the page
// cursor.
//
// v0.3 limitation — Box.Rules (fraction bars, square-root overlines)
// are not yet emitted. The math glyphs land on the page and remain
// visually identifiable, but frac and sqrt look "naked". Rule emission
// requires a rectangle/line primitive in the PDF writer; that lands in
// v0.3.x once the writer grows a RectDraw type. Documented in CHANGELOG.
func (e Engine) placeMath(cur *pageCursor, flush func(), doc *kardec.Document, m kardec.Math, blockStyle blockStyle) error {
	expr, err := mathast.Parse(m.Source())
	if err != nil {
		// Parse errors degrade gracefully into a plain-text fallback so
		// a malformed expression never aborts the whole page render.
		fallback := []kardec.Run{kardec.Text("[math: " + err.Error() + "]")}
		return e.placeTextBlock(cur, flush, fallback, blockStyle, &mathTextProvider{base: blockStyle.family})
	}

	font := doc.MathFont()
	if font == nil {
		// Math font failed to load (deferred error already captured by
		// Document); skip this block silently.
		return nil
	}

	box := mathlayout.Layout(mathadapter.WrapExpr(expr), mathadapter.WrapFont(font), blockStyle.sizePt, m.Display())
	if box.Width == 0 && box.Height == 0 && len(box.Glyphs) == 0 && len(box.Children) == 0 {
		return nil
	}

	totalHeight := box.Height + box.Depth
	if totalHeight <= 0 {
		totalHeight = blockStyle.sizePt
	}
	if cur.remainingHeight() < totalHeight {
		flush()
	}

	originX := cur.x0
	available := cur.availableWidth()
	if box.Width < available {
		// Display math centers; inline currently aligns left so it sits
		// where ordinary paragraphs do.
		if m.Display() {
			originX += (available - box.Width) / 2
		}
	}
	originY := cur.cursorY + box.Height

	emitMathBox(cur, box, originX, originY, blockStyle.color)
	cur.cursorY += totalHeight + blockStyle.spaceAfterPt
	return nil
}

// emitMathBox walks a Box tree and appends each glyph as a PlacedItem
// on the cursor. Coordinates accumulate down the tree: the absolute
// glyph X is the sum of every ancestor's X plus the glyph's own X; Y
// is referenced to the math baseline at originY.
//
// PlacedItem.Y stores the glyph's top-left in the page's top-left
// coordinate space, matching the contract used by text blocks.
func emitMathBox(cur *pageCursor, box mathlayout.Box, originX, baselineY float64, color kardec.Color) {
	for _, g := range box.Glyphs {
		glyphTopY := baselineY - 0.7*g.SizePt + g.Y
		cur.items = append(cur.items, PlacedItem{
			X:      kardec.Pt(originX + box.X + g.X),
			Y:      kardec.Pt(glyphTopY),
			Text:   string(g.Rune),
			Font:   &mathFont{rune: g.Rune},
			Size:   kardec.Pt(g.SizePt),
			Color:  color,
			IsMath: true,
		})
	}
	for _, child := range box.Children {
		emitMathBox(cur, child, originX+box.X+child.X, baselineY+child.Y, color)
	}
	for _, rule := range box.Rules {
		// Rule coordinates are relative to the box; the math layout
		// engine already places them on the math axis (frac bar /
		// sqrt overline). Translate to absolute page coordinates.
		ruleX := originX + box.X + rule.X
		// rule.Y in the math layout is reported relative to the box
		// baseline; emitMathBox's baselineY is the absolute baseline,
		// and the math engine emits negative offsets for content
		// above it. The renderer expects top-left coords, so the
		// rule's top-left is baselineY - |Height_above| + rule.Y.
		ruleY := baselineY + rule.Y
		cur.items = append(cur.items, PlacedItem{
			X: kardec.Pt(ruleX),
			Y: kardec.Pt(ruleY),
			Rect: &PlacedRect{
				Width:     kardec.Pt(rule.Width),
				Thickness: kardec.Pt(rule.Thickness),
				Color:     color,
			},
		})
	}
}

// mathFont is a marker Font implementation carrying a single math rune.
// PlacedItem requires a Font value — for math glyphs the actual
// measurement is irrelevant downstream because the renderer reads
// Text/Size/Color directly. This satisfies the type contract without
// forcing the math typography to satisfy the layout text Font shape.
type mathFont struct{ rune rune }

func (f *mathFont) Measure(text string, sizePt float64) (float64, float64, float64) {
	// Approximate metrics so the layout engine can still reason about
	// math glyphs that bleed into surrounding text. Real measurement
	// happens through mathlayout, not here.
	return float64(len(text)) * sizePt * 0.5, sizePt * 0.7, sizePt * 0.2
}

// mathTextProvider is a degenerate FontProvider used only for the parse
// fallback: when math parsing fails the engine emits a "[math: ...]"
// text run so the user sees the diagnostic on the page.
type mathTextProvider struct{ base string }

func (p *mathTextProvider) Resolve(family string, bold, italic bool) Font {
	return &mathFont{rune: 0}
}
