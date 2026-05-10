package layout

import (
	"github.com/arthurhrc/kardec"
)

// emitFootnotesForPage paints the footnote area at the bottom of the
// page: a thin separator rule plus one line per footnote, ordered by
// the page-collected refs.
//
// The chrome receives the same set of refs the layout engine accrued
// while placing body text. Footnotes whose Run is split across page
// boundaries land on the page where the marker first appeared,
// matching common typographic practice.
func emitFootnotesForPage(
	cur *pageCursor,
	refs []kardec.FootnoteRef,
	style blockStyle,
	fonts FontProvider,
) {
	if len(refs) == 0 {
		return
	}
	// Reserve a small gap above the separator and below the body
	// content. The separator itself is 30 % of the available width
	// — enough to be visually distinct without claiming the whole
	// margin.
	gap := 0.5 * style.sizePt
	separatorWidth := (cur.x1 - cur.x0) * 0.3
	separatorY := cur.y1 - footnoteAreaHeight(refs, style) - gap
	cur.items = append(cur.items, PlacedItem{
		X: kardec.Pt(cur.x0),
		Y: kardec.Pt(separatorY),
		Rect: &PlacedRect{
			Width:     kardec.Pt(separatorWidth),
			Thickness: kardec.Pt(0.4),
			Color:     kardec.Color{R: 120, G: 120, B: 120},
		},
	})

	y := separatorY + gap
	for _, ref := range refs {
		runs := append([]kardec.Run{kardec.Bold(ref.Marker() + " ")}, ref.Body()...)
		tokens := shapeRuns(runs, fonts, style, kardec.Pt(style.sizePt), style.color, nil)
		x := cur.x0
		for _, t := range tokens {
			if t.isSpace {
				x += t.width
				continue
			}
			cur.items = append(cur.items, PlacedItem{
				X:     kardec.Pt(x),
				Y:     kardec.Pt(y),
				Text:  t.text,
				Font:  t.font,
				Size:  kardec.Pt(t.sizePt),
				Color: t.color,
			})
			x += t.width
		}
		y += style.sizePt * 1.2
	}
}

// footnoteAreaHeight reports how much vertical space the footnote
// chrome will consume on a page given the supplied refs and footnote
// style. Used by the engine to shrink the body's available height.
func footnoteAreaHeight(refs []kardec.FootnoteRef, style blockStyle) float64 {
	if len(refs) == 0 {
		return 0
	}
	const separatorAndPadding = 4.0 // separator + breathing space
	return separatorAndPadding + float64(len(refs))*style.sizePt*1.2
}
