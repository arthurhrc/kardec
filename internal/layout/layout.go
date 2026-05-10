package layout

import "github.com/arthurhrc/kardec"

// Layout is the package-level convenience entry point: it constructs an
// Engine reflecting the document's configuration and runs it against
// the supplied document. The renderer track calls this from its
// top-level Render function so the call site stays a single line.
//
// The line-break algorithm is propagated from doc.LineBreakAlgorithm()
// so callers that opted in via SetLineBreakAlgorithm don't have to
// touch the layout package directly.
func Layout(doc *kardec.Document, fonts FontProvider) ([]Page, error) {
	eng := NewEngine()
	if doc != nil && doc.LineBreakAlgorithm() == kardec.LineBreakOptimal {
		eng.Algorithm = LineBreakOptimal
	}
	return eng.Layout(doc, fonts)
}
