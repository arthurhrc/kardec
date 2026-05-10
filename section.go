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

// NewSection starts a new section configured from the supplied
// PageSetup. Subsequent block / header / footer calls land in the
// new section; the previous section's blocks are frozen as-is.
//
// Use SetupOf for the common (size + margins, portrait) path:
//
//	doc.NewSection(kardec.SetupOf(kardec.PageLetter, kardec.MarginsNarrow))
//
// or build a PageSetup directly when orientation, columns or column
// gap differ from the defaults:
//
//	doc.NewSection(kardec.PageSetup{
//	    Size:        kardec.PageA4,
//	    Orientation: kardec.Landscape,
//	    Margins:     kardec.MarginsNormal,
//	})
//
// The previous (size, margins) signature was folded into this one
// during the v0.10 sweep so the document carries a single section
// constructor — old callers should migrate via SetupOf.
func (d *Document) NewSection(setup PageSetup) *Document {
	if d.err != nil {
		return d
	}
	sec := &Section{Setup: setup}
	d.sections = append(d.sections, sec)
	d.cur = sec
	return d
}

// SetupOf composes a PageSetup from a size + margins pair, defaulting
// orientation to Portrait. Convenience for the most common
// NewSection invocation:
//
//	doc.NewSection(kardec.SetupOf(kardec.PageLetter, kardec.MarginsNarrow))
//
// For explicit orientation, columns, or gap, build the PageSetup
// directly.
func SetupOf(size PageSize, margins Margins) PageSetup {
	return PageSetup{
		Size:        size,
		Orientation: Portrait,
		Margins:     margins,
	}
}

// NewSectionWithSetup forwards to NewSection.
//
// Deprecated: use NewSection(setup). The two methods were folded
// into one during the v0.10 sweep; this alias ships only for the
// v0.x line and is removed at v1.0.
func (d *Document) NewSectionWithSetup(setup PageSetup) *Document {
	return d.NewSection(setup)
}

// CurrentSection returns the section receiving subsequent block
// calls. Read-only; useful for tests and integrations that need to
// inspect Setup or attached Header / Footer values without going
// through Sections().
func (d *Document) CurrentSection() *Section { return d.cur }
