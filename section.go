package kardec

// Header attaches inline content reprinted at the top of every page in
// the current section. Pass an empty call (no runs) to clear a header
// previously set on the section.
//
// The runs may carry these substitution tokens — replaced at render
// time with the appropriate values:
//
//	{{page}}        the 1-based page number
//	{{totalPages}}  the document's total page count
//	{{section}}     the 1-based section number (1 in single-section docs)
//	{{date}}        the render date in YYYY-MM-DD form
//
// Example:
//
//	doc.Header(kardec.Text("Page {{page}} of {{totalPages}}"))
func (d *Document) Header(runs ...Run) *Document {
	if d.err != nil {
		return d
	}
	d.cur.Header = runs
	return d
}

// Footer mirrors Header for the bottom of every page in the current
// section. Same token substitutions apply.
func (d *Document) Footer(runs ...Run) *Document {
	if d.err != nil {
		return d
	}
	d.cur.Footer = runs
	return d
}
