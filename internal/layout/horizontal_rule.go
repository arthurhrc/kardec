package layout

import "github.com/arthurhrc/kardec"

// Horizontal-rule defaults. A 0.5pt gray line with 6pt of padding on
// each side keeps the divider visually quiet next to body text without
// inheriting block style — the rule is not paragraph-shaped.
const (
	defaultRuleThicknessPt = 0.5
	defaultRulePaddingPt   = 6
)

// placeHorizontalRule lays out a single HorizontalRule block. It
// reserves padding above the rule, paginates if the rule plus padding
// would not fit, emits a thin filled rect spanning the content area,
// then reserves padding below.
func (e Engine) placeHorizontalRule(cur *pageCursor, flush func(), r kardec.HorizontalRule) {
	thickness := r.Thickness.Points()
	if thickness <= 0 {
		thickness = defaultRuleThicknessPt
	}
	padding := r.Padding.Points()
	if padding <= 0 {
		padding = defaultRulePaddingPt
	}
	color := r.Color
	if color == (kardec.Color{}) {
		color = kardec.ColorGray
	}

	total := thickness + 2*padding
	if cur.remainingHeight() < total {
		flush()
	}

	cur.cursorY += padding
	cur.items = append(cur.items, PlacedItem{
		X: kardec.Pt(cur.x0),
		Y: kardec.Pt(cur.cursorY),
		Rect: &PlacedRect{
			Width:     kardec.Pt(cur.availableWidth()),
			Thickness: kardec.Pt(thickness),
			Color:     color,
		},
	})
	cur.cursorY += thickness + padding
}
