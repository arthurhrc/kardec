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
		return e.placeTextBlock(cur, flush, doc, fallback, blockStyle, &mathTextProvider{base: blockStyle.family})
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
	// Math box top-left in page top-left coords. emitMathBox now uses
	// top-left convention exclusively, so we hand it the cursor row
	// directly — no baseline arithmetic at the call site.
	originY := cur.cursorY

	emitMathBox(cur, box, originX, originY, blockStyle.color)
	cur.cursorY += totalHeight + blockStyle.spaceAfterPt
	return nil
}

// emitMathBox walks a Box tree and appends each glyph + rule as a
// PlacedItem on the cursor. Coordinates accumulate down the tree
// using the same top-left-origin convention mathlayout uses
// internally: every Box.X / Box.Y / Glyph.X / Glyph.Y is the
// top-left offset within the parent Box, in points.
//
// originX, originY are the absolute (page top-left) coordinates of
// THIS box's top-left corner. Children recurse with originX+child.X,
// originY+child.Y.
//
// PlacedItem.Y carries each glyph's BASELINE in top-left page
// coordinates because the PDF writer's Td operator positions the
// text baseline (not the top edge) at the supplied Y. The baseline
// for a glyph at top g.Y is g.Y + glyph_ascent — we approximate the
// ascent at 0.7 × SizePt, the same fallback the typography Measure
// helper uses.
//
// (Through v0.21 emitMathBox confused the convention: the old
// formula `baselineY - 0.7*g.SizePt + g.Y` mixed parent-baseline
// and glyph-top semantics, so superscripts and subscripts in inline
// math collapsed onto the base glyph's row. Fixed in v0.21.1.)
func emitMathBox(cur *pageCursor, box mathlayout.Box, originX, originY float64, color kardec.Color) {
	for _, g := range box.Glyphs {
		glyphAscent := 0.7 * g.SizePt
		cur.items = append(cur.items, PlacedItem{
			X:      kardec.Pt(originX + g.X),
			Y:      kardec.Pt(originY + g.Y + glyphAscent),
			Text:   string(g.Rune),
			Font:   &mathFont{rune: g.Rune},
			Size:   kardec.Pt(g.SizePt),
			Color:  color,
			IsMath: true,
		})
	}
	for _, child := range box.Children {
		emitMathBox(cur, child, originX+child.X, originY+child.Y, color)
	}
	for _, rule := range box.Rules {
		cur.items = append(cur.items, PlacedItem{
			X: kardec.Pt(originX + rule.X),
			Y: kardec.Pt(originY + rule.Y),
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
