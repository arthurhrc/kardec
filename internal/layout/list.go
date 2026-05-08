package layout

import (
	"strconv"

	"github.com/arthurhrc/kardec"
)

// placeList lays out a kardec.List by emitting one synthetic paragraph
// per item, prefixed with the level's marker and shifted right by an
// indent that grows with depth. Nested children recurse one level
// deeper, rotating the bullet shape so successive levels are visually
// distinct.
//
// v0.3 keeps the marker model simple: bullets / hollow circles /
// squares for unordered lists, decimal numerals for ordered ones. Per
// item style (definition lists, lettered ordering, marker
// punctuation) is left for v0.4 once the public surface settles.
func (e Engine) placeList(cur *pageCursor, flush func(), list kardec.List, baseStyle blockStyle, fonts FontProvider) error {
	return e.placeListLevel(cur, flush, list, baseStyle, fonts, 0)
}

// placeListLevel is the recursive worker behind placeList. depth is
// 0-based; each step adds one indent's worth of horizontal offset and
// rotates the bullet shape.
func (e Engine) placeListLevel(cur *pageCursor, flush func(), list kardec.List, baseStyle blockStyle, fonts FontProvider, depth int) error {
	indent := listIndent(depth, baseStyle.sizePt)

	itemStyle := baseStyle
	itemStyle.spaceBeforePt = 0
	itemStyle.spaceAfterPt = 0.4 * baseStyle.sizePt

	// Reserve indentation by shifting x0 for the duration of this
	// level. Restoring at the end keeps siblings unaffected.
	originalX0 := cur.x0
	cur.x0 += indent
	defer func() { cur.x0 = originalX0 }()

	for i, item := range list.Items() {
		marker := listMarker(list.Style(), depth, i)
		runs := append([]kardec.Run{kardec.Text(marker)}, item.Runs...)
		if err := e.placeTextBlock(cur, flush, runs, itemStyle, fonts); err != nil {
			return err
		}
		for _, child := range item.Children {
			if err := e.placeListLevel(cur, flush, child, baseStyle, fonts, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// listIndent returns the horizontal offset, in points, for items at
// the given nesting depth. depth 0 keeps the document's body left
// margin; each deeper level adds a fixed multiple of the body font
// size so text alignment scales with typography.
func listIndent(depth int, sizePt float64) float64 {
	return float64(depth+1) * 1.4 * sizePt
}

// listMarker returns the marker prefix that decorates an item at the
// given (style, depth, index). Markers include trailing whitespace so
// callers can prepend them directly to the item's runs.
func listMarker(style kardec.ListStyle, depth, index int) string {
	switch style {
	case kardec.ListOrdered:
		return strconv.Itoa(index+1) + ".  "
	default:
		switch depth % 3 {
		case 0:
			return "•  "
		case 1:
			return "◦  "
		default:
			return "▪  "
		}
	}
}
