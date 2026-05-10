package kardec

// FootnoteRef is the inline marker (typically a small superscript
// numeral) referenced from body text. Layout collects every
// FootnoteRef encountered while laying out a page and emits the
// matching text at the bottom of that page, separated by a thin
// rule.
//
// Numbering is sequential across the whole document. The marker text
// shows the auto-assigned number; callers who want a custom marker
// (e.g. an asterisk) use FootnoteRefMarker.
type FootnoteRef struct {
	number int
	body   []Run
	marker string // empty = auto numeric
}

// Marker returns the visible label for the footnote — either the
// caller-supplied custom string or the decimal form of Number.
func (f FootnoteRef) Marker() string {
	if f.marker != "" {
		return f.marker
	}
	return decimalString(f.number)
}

// Number returns the 1-based footnote index inside the document.
// Useful for tests and renderers that want to format the marker
// differently from the default decimal.
func (f FootnoteRef) Number() int { return f.number }

// Body returns the inline runs that compose the footnote text shown
// at the bottom of the page.
func (f FootnoteRef) Body() []Run { return f.body }

// Footnote builds a Run that renders an auto-numbered superscript
// marker inline, while registering the supplied body runs for
// emission at the foot of the same page. Numbering increments per
// call and is shared with FootnoteWith / Footnote-as-method.
//
// Use it inline in a Paragraph:
//
//	doc.Paragraph(
//	    kardec.Text("Sales grew "),
//	    doc.Footnote("see appendix B for the breakdown."),
//	    kardec.Text(" this quarter."),
//	)
func (d *Document) Footnote(body string) Run {
	if d == nil {
		return Run{}
	}
	return d.FootnoteWith("", Text(body))
}

// FootnoteWith is the rich-content variant of Footnote: a caller-
// supplied marker (e.g. "*", "†") plus inline runs. An empty marker
// falls back to the auto-numbered decimal form.
func (d *Document) FootnoteWith(marker string, body ...Run) Run {
	if d == nil {
		return Text("")
	}
	d.footnoteCounter++
	ref := FootnoteRef{
		number: d.footnoteCounter,
		body:   body,
		marker: marker,
	}
	d.footnotes = append(d.footnotes, ref)
	return Run{
		text:        ref.Marker(),
		footnoteRef: ref.number,
	}
}

// decimalString turns a non-negative int into its decimal string
// without depending on strconv — keeps the footnote helper free of
// extra imports and matches the style of the section_chrome itoa.
func decimalString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + decimalString(-n)
	}
	digits := make([]byte, 0, 4)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
