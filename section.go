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

// NewSection starts a new section configured from the supplied page
// size and margins. Subsequent block / header / footer calls land in
// the new section; the previous section's blocks are frozen as-is.
//
// Use this to interleave landscape pages, tighter-margin appendices,
// or a Letter-sized cover before the body switches to A4. Each
// section carries its own Header / Footer; pass them right after
// NewSection to override the defaults inherited from PageSetup.
//
// orientation defaults to Portrait — pass `Landscape` explicitly via
// the returned PageSetup if a landscape break is wanted. The helper
// below preserves the orientation of the current section so callers
// who only want margin or page-size changes don't have to repeat it.
func (d *Document) NewSection(size PageSize, margins Margins) *Document {
	if d.err != nil {
		return d
	}
	sec := &Section{
		Setup: PageSetup{
			Size:        size,
			Orientation: d.cur.Setup.Orientation,
			Margins:     margins,
		},
	}
	d.sections = append(d.sections, sec)
	d.cur = sec
	return d
}

// NewSectionWithSetup is the explicit-setup counterpart to NewSection,
// useful when the caller has a pre-built PageSetup (e.g. landscape
// orientation) it wants applied verbatim.
func (d *Document) NewSectionWithSetup(setup PageSetup) *Document {
	if d.err != nil {
		return d
	}
	sec := &Section{Setup: setup}
	d.sections = append(d.sections, sec)
	d.cur = sec
	return d
}

// CurrentSection returns the section receiving subsequent block
// calls. Read-only; useful for tests and integrations that need to
// inspect Setup or attached Header / Footer values without going
// through Sections().
func (d *Document) CurrentSection() *Section { return d.cur }
