package render

import "github.com/arthurhrc/kardec/internal/pdf"

// linkRangeAccumulator coalesces consecutive PlacedItems carrying the
// same link target into a single rectangular annotation. Without
// coalescing, a multi-word link would emit one tiny click rectangle
// per word; readers would still resolve them but the box around the
// link would visually fragment.
//
// The accumulator assumes items arrive in left-to-right order on a
// single line. When the URI changes or a vertical break is seen, the
// pending range is flushed and a new one starts.
type linkRangeAccumulator struct {
	pending  *pdf.LinkAnnot
	finished []pdf.LinkAnnot
}

func newLinkRangeAccumulator() *linkRangeAccumulator {
	return &linkRangeAccumulator{}
}

// add registers a single text fragment with its bounding box and
// hyperlink target. Consecutive fragments sharing the same target on
// the same vertical band are merged.
func (a *linkRangeAccumulator) add(uri string, x, y, w, h float64) {
	if a.pending != nil && a.pending.URI == uri && sameLine(a.pending, y, h) {
		// Extend the pending range to encompass the new fragment.
		right := x + w
		newRight := maxF(a.pending.X+a.pending.W, right)
		a.pending.X = minF(a.pending.X, x)
		a.pending.W = newRight - a.pending.X
		// Vertical: keep the union of the two bands.
		topOld := a.pending.Y + a.pending.H
		topNew := y + h
		a.pending.Y = minF(a.pending.Y, y)
		a.pending.H = maxF(topOld, topNew) - a.pending.Y
		return
	}
	if a.pending != nil {
		a.finished = append(a.finished, *a.pending)
	}
	a.pending = &pdf.LinkAnnot{X: x, Y: y, W: w, H: h, URI: uri}
}

// flush returns every collected range, including the pending one,
// and resets the accumulator for the next page.
func (a *linkRangeAccumulator) flush() []pdf.LinkAnnot {
	if a.pending != nil {
		a.finished = append(a.finished, *a.pending)
		a.pending = nil
	}
	out := a.finished
	a.finished = nil
	return out
}

// sameLine reports whether the new fragment vertically overlaps the
// pending one enough to be considered part of the same line. The
// threshold is 30 % of the pending height — enough to absorb minor
// baseline drift inside a line, while still splitting on a real
// vertical break.
func sameLine(p *pdf.LinkAnnot, y, h float64) bool {
	overlap := minF(p.Y+p.H, y+h) - maxF(p.Y, y)
	if overlap < 0 {
		overlap = 0
	}
	return overlap >= 0.3*minF(p.H, h)
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
