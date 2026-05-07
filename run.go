package kardec

// Run is an inline fragment of text with optional style overrides. Runs are
// the leaf content nodes that paragraphs and headings carry. Construct them
// via Text, Bold, Italic and similar helpers; do not instantiate the struct
// directly so future fields stay backward-compatible.
type Run struct {
	text     string
	bold     bool
	italic   bool
	override styleOverride
}

// Text returns a plain Run carrying no inline overrides.
func Text(s string) Run { return Run{text: s} }

// Bold returns a Run rendered in bold weight.
func Bold(s string) Run { return Run{text: s, bold: true} }

// Italic returns a Run rendered in italic style.
func Italic(s string) Run { return Run{text: s, italic: true} }

// BoldItalic returns a Run rendered in both bold weight and italic style.
func BoldItalic(s string) Run { return Run{text: s, bold: true, italic: true} }

// Colored wraps a Run, attaching an explicit color override.
func Colored(r Run, c Color) Run {
	r.override.color = &c
	return r
}

// styleOverride captures the optional inline style fields a Run may carry.
// Fields are pointers so "absent" remains distinguishable from "zero value".
type styleOverride struct {
	color *Color
	size  *Length
}
