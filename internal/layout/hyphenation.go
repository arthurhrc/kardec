package layout

import "github.com/arthurhrc/kardec/internal/hyphenation"

// hyphenBreakPoints is the layout-side wrapper around the
// hyphenation package's English heuristic. Layout always asks for
// English breaks today; multi-language support lands once a Liang
// pattern table is added.
func hyphenBreakPoints(word string) []int {
	return hyphenation.BreakPoints(word, "en")
}
