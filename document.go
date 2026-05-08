package kardec

import (
	"errors"
	"io"

	"github.com/arthurhrc/kardec/internal/typography"
)

// Document is the central builder a caller composes content onto. It holds
// one or more sections (a section owns a PageSetup, an optional header /
// footer, and a list of blocks).
//
// A *Document is not safe for concurrent use; see the package overview.
type Document struct {
	sections []*Section
	cur      *Section // pointer into sections; the section currently receiving blocks

	fonts *typography.Registry // registry of registered + bundled font faces

	err error // first error encountered during builder usage; surfaced by Err and Render
}

// Section couples a PageSetup with the ordered list of blocks that flow
// inside it. Most documents use a single Section; multi-section support
// (different page setups within one document) lands later in v0.x.
type Section struct {
	Setup  PageSetup
	Blocks []Block
}

// New creates an empty Document with a single Section configured from the
// supplied page size and margins. Default orientation is Portrait.
func New(size PageSize, margins Margins) *Document {
	first := &Section{
		Setup: PageSetup{
			Size:        size,
			Orientation: Portrait,
			Margins:     margins,
		},
	}
	d := &Document{
		sections: []*Section{first},
		cur:      first,
		fonts:    typography.NewRegistry(),
	}
	// Best-effort: register the bundled OFL families so MeasureText works
	// out of the box. A failure here is captured in the deferred error
	// chain so Render surfaces it instead of panicking.
	if err := typography.LoadBuiltinFonts(d.fonts); err != nil {
		d.err = err
	}
	return d
}

// Err returns the first error captured during builder usage, or nil. Render
// returns the same error before attempting any I/O, so most callers can
// wait until Render to handle failures.
func (d *Document) Err() error { return d.err }

// fail records err as the document's first error if no error was set yet,
// then returns d for fluent chaining. Subsequent calls remain inert until
// the document is recreated.
func (d *Document) fail(err error) *Document {
	if d.err == nil {
		d.err = err
	}
	return d
}

// append adds a block to the current section, unless a prior error has
// already invalidated the document.
func (d *Document) append(b Block) *Document {
	if d.err != nil {
		return d
	}
	d.cur.Blocks = append(d.cur.Blocks, b)
	return d
}

// Heading appends a heading block at the requested level. Levels outside
// 1..6 are clamped to the nearest valid value.
func (d *Document) Heading(level int, runs ...Run) *Document {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return d.append(Heading{level: level, runs: runs})
}

// Paragraph appends a body paragraph composed of inline runs.
func (d *Document) Paragraph(runs ...Run) *Document {
	return d.append(Paragraph{runs: runs})
}

// PageBreak appends a forced page break.
func (d *Document) PageBreak() *Document { return d.append(PageBreak{}) }

// Spacer appends vertical whitespace of the given height.
func (d *Document) Spacer(h Length) *Document { return d.append(Spacer{Height: h}) }

// ErrNotImplemented is returned by Render and friends until the layout and
// PDF tracks land. It exists so callers can write conditional code today.
var ErrNotImplemented = errors.New("kardec: render path not implemented yet")

// Render produces a PDF and writes it to the named file. Returns ErrNotImplemented
// while the layout / typography / renderer tracks are under construction.
func (d *Document) Render(path string) error {
	if d.err != nil {
		return d.err
	}
	_ = path
	return ErrNotImplemented
}

// RenderTo produces a PDF and writes it to the supplied io.Writer. Returns
// ErrNotImplemented during the skeleton phase.
func (d *Document) RenderTo(w io.Writer) error {
	if d.err != nil {
		return d.err
	}
	_ = w
	return ErrNotImplemented
}

// Bytes returns the rendered PDF as a byte slice. Returns ErrNotImplemented
// during the skeleton phase.
func (d *Document) Bytes() ([]byte, error) {
	if d.err != nil {
		return nil, d.err
	}
	return nil, ErrNotImplemented
}
