package kardec

// Cross-reference machinery: figure / table auto-numbering plus the
// Ref / RefPage helpers that resolve a user-supplied label into a
// visible run.
//
// Numbering happens at build time — the moment a label is registered
// via LabeledFigure / LabeledTable, its number is fixed. Ref(label)
// returns a Run carrying the visible "Figure 3" text with an
// internal hyperlink to the auto-anchor placed before the figure.
// Page-number resolution happens after layout: RefPage(label) emits
// a `{{refpage:label}}` placeholder that the post-pass replaces with
// the page on which the matching anchor landed, mirroring the TOC's
// `{{tocpage:hN}}` strategy.

const (
	// RefAnchorPrefix is the leading string of the anchor name that
	// LabeledFigure / LabeledTable emit before each labeled block.
	// Internal — exposed so test code can assert on it without
	// importing layout-internal helpers.
	RefAnchorPrefix = "kardec-ref-"

	// RefPagePlaceholder is the placeholder text RefPage emits.
	// Layout's post-pass searches for it and replaces each occurrence
	// with the page number on which the matching anchor landed.
	RefPagePlaceholder = "{{refpage:"
)

// RefAnchorName returns the anchor name LabeledFigure / LabeledTable
// emit for the given label. Exposed so tests and integrations can
// build clickable cross-references by hand if needed.
func RefAnchorName(label string) string { return RefAnchorPrefix + label }

// registerFigureLabel assigns the next figure number to label and
// records the resolved metadata. Returns the assigned number.
func (d *Document) registerFigureLabel(label string) int {
	d.figureCounter++
	if d.labels == nil {
		d.labels = make(map[string]labelInfo)
	}
	d.labels[label] = labelInfo{kind: labelFigure, number: d.figureCounter}
	return d.figureCounter
}

// registerTableLabel assigns the next table number to label and
// records the resolved metadata.
func (d *Document) registerTableLabel(label string) int {
	d.tableCounter++
	if d.labels == nil {
		d.labels = make(map[string]labelInfo)
	}
	d.labels[label] = labelInfo{kind: labelTable, number: d.tableCounter}
	return d.tableCounter
}

// Ref returns a Run resolving to the canonical visible reference for
// label — "Figure 3" or "Table 2" — with an internal hyperlink to
// the auto-anchor LabeledFigure / LabeledTable emitted before the
// matching block. Pair with RefPage when the prose also wants the
// destination page number.
//
// Unknown labels resolve to "[?ref:<label>]" so missing references
// stand out in the rendered output. The Document does not promote
// missing references to a deferred error: a forward reference whose
// target lands later in the build is allowed by the auto-numbering
// model, so the lookup may legitimately fail at the moment Ref is
// called. Once the document is fully built, callers wanting a strict
// audit can iterate Document.MissingRefs (introduced when needed).
func (d *Document) Ref(label string) Run {
	info, ok := d.labels[label]
	if !ok {
		return Run{text: "[?ref:" + label + "]"}
	}
	return Run{
		text: refVisibleText(info),
		link: "#" + RefAnchorName(label),
	}
}

// RefPage returns a Run carrying a page-number placeholder for label.
// The post-layout pass substitutes the placeholder with the page on
// which the anchor named "kardec-ref-<label>" landed. Compose with
// Ref when the prose wants both the canonical reference and the
// page:
//
//	doc.Paragraph(
//	    kardec.Text("See "),
//	    doc.Ref("growth-2024"),
//	    kardec.Text(" on page "),
//	    doc.RefPage("growth-2024"),
//	    kardec.Text("."),
//	)
//
// Unknown labels resolve to a literal "?" placeholder.
func (d *Document) RefPage(label string) Run {
	if _, ok := d.labels[label]; !ok {
		return Run{text: "?"}
	}
	return Run{text: RefPagePlaceholder + label + "}}"}
}

// refVisibleText composes the canonical "Figure N" / "Table N" text
// for a resolved label.
func refVisibleText(info labelInfo) string {
	prefix := "Figure"
	if info.kind == labelTable {
		prefix = "Table"
	}
	return prefix + " " + itoaSmall(info.number)
}

// itoaSmall is a tiny int-to-string helper that avoids pulling in
// strconv for what is almost always a single- or double-digit
// figure / table number.
func itoaSmall(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	var buf [11]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
