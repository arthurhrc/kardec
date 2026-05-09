package render

import (
	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/pdf"
)

// buildDestinations walks every layout Page and converts its
// AnchorMark slice into pdf.NamedDestination entries. Y is flipped
// from the layout's top-left origin to PDF's bottom-left so the
// writer can embed the value directly in the /Dests array.
//
// Anchors with an empty Name are filtered out — those are programmatic
// no-ops produced when a caller passes an empty string. Duplicate
// names are kept in source order; the last definition wins under PDF
// reader resolution because it overwrites the earlier entry in the
// /Dests dictionary.
func buildDestinations(pages []layout.Page) []pdf.NamedDestination {
	var out []pdf.NamedDestination
	for pageIdx, p := range pages {
		pageHeight := p.Height.Points()
		for _, a := range p.Anchors {
			if a.Name == "" {
				continue
			}
			out = append(out, pdf.NamedDestination{
				Name:      a.Name,
				PageIndex: pageIdx,
				Y:         pageHeight - a.Y.Points(),
			})
		}
	}
	return out
}
