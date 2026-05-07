package layout

import "github.com/arthurhrc/kardec"

// Layout is the package-level convenience entry point: it constructs a
// default Engine and runs it against the supplied document. The renderer
// track calls this from its top-level Render function so the call site
// stays a single line.
//
// Equivalent to:
//
//	NewEngine().Layout(doc, fonts)
func Layout(doc *kardec.Document, fonts FontProvider) ([]Page, error) {
	return NewEngine().Layout(doc, fonts)
}
